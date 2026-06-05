package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
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
	r.Register(pressure.New(pressureClientFactory{gatewayURL: cfg.Target.GatewayURL}))
	if db != nil {
		seedDAO := dao.NewSeedDAO(db)
		bizCli := auction.NewClient(cfg.Target.GatewayURL, 10*time.Second)
		bizCli.SetJWTSecret(cfg.Security.JWTSecret)
		r.Register(e2e.NewScenario(bizCli, seedDAO))
		internalCli := auction.NewClient(cfg.Target.AuctionURL, 10*time.Second)
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
}

func (f pressureClientFactory) NewClient() pressure.Bidder {
	// 5s 超时；测试模式下 user_id 走请求体，不需要 Authorization
	return pressure.NewClient(f.gatewayURL, "", 5*time.Second)
}
