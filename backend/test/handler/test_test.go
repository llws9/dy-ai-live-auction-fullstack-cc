package handler

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"test-service/dao"
	"test-service/model"
	"test-service/runner"
)

// 端到端：起 Hertz server，跑通 dummy → status → history
func TestHTTP_DummyEndpoints(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.TestResult{}, &model.TestSeedData{}))

	resultDAO := dao.NewResultDAO(db)
	r := runner.New(resultDAO)
	r.Register(runner.NewDummyScenario(50 * time.Millisecond))
	th := NewTestHandler(r, resultDAO)

	port := freePort(t)
	h := server.New(server.WithHostPorts("127.0.0.1:" + port))
	apiTest := h.Group("/api/test")
	apiTest.POST("/dummy", th.PostDummy)
	apiTest.GET("/status/:id", th.GetStatus)
	apiTest.GET("/history", th.GetHistory)
	apiTest.POST("/cancel/:id", th.PostCancel)

	go h.Spin()
	defer func() { _ = h.Shutdown(context.Background()) }()
	waitForPort(t, port)

	base := "http://127.0.0.1:" + port

	// POST /dummy
	resp, err := http.Post(base+"/api/test/dummy", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode, string(body))
	var out struct {
		TestID string `json:"test_id"`
	}
	require.NoError(t, json.Unmarshal(body, &out))
	require.NotEmpty(t, out.TestID)

	// GET /status/:id 直到 completed
	require.Eventually(t, func() bool {
		r2, e := http.Get(base + "/api/test/status/" + out.TestID)
		if e != nil {
			return false
		}
		defer r2.Body.Close()
		b, _ := io.ReadAll(r2.Body)
		// gorm 序列化字段名首字母大写
		var s struct {
			Status string `json:"Status"`
		}
		_ = json.Unmarshal(b, &s)
		return s.Status == model.StatusCompleted
	}, 3*time.Second, 50*time.Millisecond)

	// GET /history
	r3, err := http.Get(base + "/api/test/history?test_type=dummy&page=1&page_size=10")
	require.NoError(t, err)
	defer r3.Body.Close()
	hbody, _ := io.ReadAll(r3.Body)
	require.Equal(t, 200, r3.StatusCode, string(hbody))
	var hist struct {
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(hbody, &hist))
	assert.GreaterOrEqual(t, hist.Total, int64(1))
}

// freePort 返回一个随机可用端口
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	addr := l.Addr().String()
	idx := strings.LastIndex(addr, ":")
	return addr[idx+1:]
}

func waitForPort(t *testing.T, port string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 100*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("port %s never opened", port)
}
