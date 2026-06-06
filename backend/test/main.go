package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm"

	"test-service/client/auction"
	"test-service/config"
	"test-service/dao"
	"test-service/handler"
	"test-service/mock/partner"
	"test-service/runner"
	"test-service/scenario/antisnipe"
	"test-service/scenario/callback"
	chaosScenario "test-service/scenario/chaos"
	"test-service/scenario/e2e"
	"test-service/scenario/pressure"
	"test-service/scenario/script"
	"test-service/scenario/user_journey"
	"test-service/service/cron"
	"test-service/ws"
)

func main() {
	// 日志级别（环境变量 LOG_LEVEL=debug|info|warn|error，默认 info）
	setupLogLevel(os.Getenv("LOG_LEVEL"))

	cfg := config.LoadFromEnv()

	// DB 初始化（可选：连接失败仅打日志，便于在没有 MySQL 的环境下做演示骨架联调）
	var db *gorm.DB
	if d, err := dao.InitDB(cfg.Database); err != nil {
		hlog.Warnf("[boot] DB init failed (running without DB): %v", err)
	} else {
		db = d
	}

	var resultDAO *dao.ResultDAO
	if db != nil {
		resultDAO = dao.NewResultDAO(db)

		// Demo seed：仅当 ENABLE_DEMO_SEED=true 且表为空时插入
		if isTrue(os.Getenv("ENABLE_DEMO_SEED")) {
			if err := dao.SeedDemoHistory(context.Background(), resultDAO); err != nil {
				hlog.Errorf("[boot] seed demo failed: %v", err)
			}
		}
	}

	// 进度 broker（200ms 节流）
	broker := ws.NewBroker(200 * time.Millisecond)

	// runner + 注册 dummy 场景
	r := runner.New(resultDAO)
	r.SetEmitter(func(testID string, progress int, step string, metrics map[string]any) {
		broker.Publish(testID, ws.Message{Progress: progress, Step: step, Metrics: metrics})
	})
	r.Register(runner.NewDummyScenario(5 * time.Second))
	r.Register(pressure.New(pressureClientFactory{gatewayURL: cfg.Target.GatewayURL, jwtSecret: cfg.Security.JWTSecret, db: db}))
	var bizCli *auction.Client
	var internalCli *auction.Client
	if db != nil {
		seedDAO := dao.NewSeedDAO(db)
		bizCli = auction.NewClient(cfg.Target.GatewayURL, 10*time.Second)
		bizCli.SetJWTSecret(cfg.Security.JWTSecret)
		r.Register(e2e.NewScenario(bizCli, seedDAO))
		internalCli = auction.NewClient(cfg.Target.AuctionURL, 10*time.Second)
		internalCli.SetInternalToken(cfg.Security.InternalToken)
		r.Register(user_journey.NewScenario(bizCli, internalCli, seedDAO))

		// 防狙击场景：通过 SDK 工厂创建 30s 拍卖
		antisnipeFactory := antisnipe.NewSDKAuctionFactory(bizCli, 9001, 30)
		r.Register(antisnipe.NewScenario(bizCli, antisnipeFactory))
	}

	// 启动 Mock Partner Server（独立 :18091）；callback 场景依赖
	mockPartner := partner.NewServer()
	go func() {
		hlog.Infof("[boot] Mock Partner Server listening :18091")
		if err := mockPartner.Start(":18091"); err != nil {
			hlog.Errorf("[boot] Mock Partner Server stopped: %v", err)
		}
	}()
	r.Register(callback.NewScenario())

	// M6 chaos：进程内故障注入（默认 probe gateway /health）
	r.Register(chaosScenario.NewScenario(strings.TrimRight(cfg.Target.GatewayURL, "/") + "/health"))

	// M7.1 script：组合剧本，从注册表按 type 取子场景
	r.Register(script.NewScenario(r))

	// 历史清理 cron：保留 7 天，每 24h 跑一次，启动后立即跑一次
	if resultDAO != nil {
		cleanup := cron.New(resultDAO, cron.Config{
			Retention:  7 * 24 * time.Hour,
			Interval:   24 * time.Hour,
			RunOnStart: true,
		})
		cleanup.Start(context.Background())
	}

	// HTTP 主服务
	h := server.Default(server.WithHostPorts(cfg.Server.HTTPPort))
	h.GET("/health", handler.Health)

	if resultDAO != nil {
		th := handler.NewTestHandler(r, resultDAO)
		demoHandler := handler.NewDemoHandler(bizCli, internalCli, cfg.Security.JWTSecret)
		api := h.Group("/api/test")
		api.POST("/dummy", th.PostDummy)
		api.POST("/pressure", th.PostPressure)
		api.POST("/e2e", th.PostE2E)
		api.POST("/user-journey", th.PostUserJourney)
		api.POST("/antisnipe", th.PostAntiSnipe)
		api.POST("/callback", th.PostCallback)
		api.POST("/chaos", th.PostChaos)
		api.POST("/script/:name", th.PostScript)
		api.POST("/compare", th.PostCompare)
		api.GET("/status/:id", th.GetStatus)
		api.GET("/history", th.GetHistory)
		api.GET("/report/:id", th.GetReport)
		api.POST("/cancel/:id", th.PostCancel)
		demo := api.Group("/demo")
		demo.POST("/follow-bid", demoHandler.PostFollowBid)
		demo.POST("/recharge", demoHandler.PostRecharge)
		hlog.Infof("[boot] /api/test/* registered")
	} else {
		hlog.Warnf("[boot] /api/test/* disabled (no DB)")
	}

	// WebSocket 独立 server
	wsHandler := handler.NewWSHandler(broker)
	go startWSServer(cfg.Server.WSPort, wsHandler)

	hlog.Infof("[boot] test-service starting HTTP=%s WS=%s log_level=%s demo_seed=%v",
		cfg.Server.HTTPPort, cfg.Server.WSPort, os.Getenv("LOG_LEVEL"),
		isTrue(os.Getenv("ENABLE_DEMO_SEED")))
	h.Spin()
}

// startWSServer 启动独立 WS server
func startWSServer(port string, wsHandler *handler.WSHandler) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/test/progress", wsHandler.HandleProgress)
	mux.HandleFunc("/ws/test/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	hlog.Infof("[boot] WS server listening %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("ws server start failed: %v", err)
	}
}

func setupLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		hlog.SetLevel(hlog.LevelDebug)
	case "warn":
		hlog.SetLevel(hlog.LevelWarn)
	case "error":
		hlog.SetLevel(hlog.LevelError)
	default:
		hlog.SetLevel(hlog.LevelInfo)
	}
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// pressureClientFactory 适配 pressure.ClientFactory，注入 gateway 地址
type pressureClientFactory struct {
	gatewayURL string
	jwtSecret  string
	db         *gorm.DB
}

func (f pressureClientFactory) NewClient() pressure.Bidder {
	return pressure.NewJWTClient(f.gatewayURL, f.jwtSecret, -1)
}

func (f pressureClientFactory) PrepareFixture(ctx context.Context, cfg pressure.Config) (pressure.Fixture, error) {
	if f.jwtSecret == "" {
		return pressure.Fixture{}, fmt.Errorf("JWT_SECRET is required for pressure fixture")
	}
	if f.db == nil {
		return pressure.Fixture{}, fmt.Errorf("database is required for pressure fixture")
	}
	if err := f.ensurePressureUsers(ctx, cfg.ConcurrentUsers); err != nil {
		return pressure.Fixture{}, err
	}

	auctionCount := 1
	if cfg.Scenario == "throughput" {
		auctionCount = cfg.ConcurrentUsers
		if cfg.FixtureCount > 0 && cfg.FixtureCount < auctionCount {
			auctionCount = cfg.FixtureCount
		}
	}
	if auctionCount < 1 {
		auctionCount = 1
	}

	cli := auction.NewClient(f.gatewayURL, 10*time.Second)
	cli.SetJWTSecret(f.jwtSecret)

	seller := auction.Actor{UserID: 9001, Username: "pressure_merchant_9001", Role: auction.RoleMerchant}
	if auctionCount == 1 {
		auctionID, err := f.createPressureAuction(ctx, cli, seller, cfg, 0)
		if err != nil {
			return pressure.Fixture{}, err
		}
		return pressure.Fixture{AuctionID: auctionID, AuctionIDs: []int64{auctionID}}, nil
	}

	auctionIDs := make([]int64, auctionCount)
	errs := make(chan error, auctionCount)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := 0; i < auctionCount; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			auctionID, err := f.createPressureAuction(ctx, cli, seller, cfg, i)
			if err != nil {
				errs <- err
				return
			}
			auctionIDs[i] = auctionID
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return pressure.Fixture{}, err
		}
	}
	return pressure.Fixture{AuctionID: auctionIDs[0], AuctionIDs: auctionIDs}, nil
}

func (f pressureClientFactory) createPressureAuction(ctx context.Context, cli *auction.Client, seller auction.Actor, cfg pressure.Config, index int) (int64, error) {
	productStep := cli.CreateProductAs(ctx, seller, auction.CreateProductReq{
		Name:        fmt.Sprintf("压测拍品 %d-%d", time.Now().UnixNano(), index),
		Description: "pressure auto fixture",
		Status:      1,
	})
	if !productStep.OK || productStep.RefID <= 0 {
		return 0, fmt.Errorf("pressure create product failed: %s", productStep.Message)
	}

	ruleStep := cli.CreateAuctionRule(ctx, seller, productStep.RefID, auction.CreateAuctionRuleReq{
		StartPrice:         cfg.BidAmount,
		Increment:          1,
		Duration:           cfg.DurationSec + 30,
		TriggerDelayBefore: 5,
		DelayDuration:      5,
		MaxDelayTime:       30,
	})
	if !ruleStep.OK {
		return 0, fmt.Errorf("pressure create auction rule failed: %s", ruleStep.Message)
	}

	auctionStep := cli.CreateAuctionAs(ctx, seller, auction.CreateAuctionReq{
		ProductID:  productStep.RefID,
		StartPrice: cfg.BidAmount,
		Increment:  1,
		Duration:   cfg.DurationSec + 30,
	})
	if !auctionStep.OK || auctionStep.RefID <= 0 {
		return 0, fmt.Errorf("pressure create auction failed: %s", auctionStep.Message)
	}

	if step := cli.WaitAuctionStarted(ctx, auctionStep.RefID, 100*time.Millisecond, 5*time.Second); !step.OK {
		return 0, fmt.Errorf("pressure wait auction started failed: %s", step.Message)
	}

	return auctionStep.RefID, nil
}

func (f pressureClientFactory) ensurePressureUsers(ctx context.Context, concurrentUsers int) error {
	const passwordHash = "$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy"
	now := time.Now()
	users := make([]map[string]any, 0, concurrentUsers)
	for i := 0; i < concurrentUsers; i++ {
		userID := int64(100000 + i)
		users = append(users, map[string]any{
			"id":         userID,
			"name":       fmt.Sprintf("压测用户 %d", userID),
			"avatar":     "",
			"email":      nil,
			"phone":      nil,
			"password":   passwordHash,
			"role":       0,
			"status":     1,
			"created_at": now,
		})
	}
	for _, user := range users {
		if err := f.db.WithContext(ctx).Exec(`
INSERT INTO users (id, name, avatar, email, phone, password, role, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  avatar = VALUES(avatar),
  password = VALUES(password),
  role = VALUES(role),
  status = VALUES(status)
`, user["id"], user["name"], user["avatar"], user["email"], user["phone"], user["password"], user["role"], user["status"], user["created_at"]).Error; err != nil {
			return fmt.Errorf("pressure seed user %v failed: %w", user["id"], err)
		}
	}
	return nil
}
