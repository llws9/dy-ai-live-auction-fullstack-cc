// Package partner Mock Partner Server：模拟外部平台对接服务。
//
// 用于场景 H（回调可靠投递）测试：
//   - POST /partner/orders                         接收回调，校验 HMAC，落 inbox
//   - GET  /partner/orders/by-idempotency-key/:key 探测查询
//   - POST /partner/_admin/config                  管理：配置故障注入
//   - GET  /partner/_admin/inbox                   管理：查看 inbox
//   - POST /partner/_admin/reset                   管理：清空 inbox + 重置配置
package partner

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"time"
)

// Config Mock 行为配置（管理 API 可改）
type Config struct {
	HMACSecret    string `json:"hmac_secret"`     // HMAC 密钥（默认 "test-secret-key"）
	DelayMs       int    `json:"delay_ms"`        // 处理延迟（毫秒），用于模拟超时
	FailRate      float64 `json:"fail_rate"`      // 0~1 失败率
	ForceFailNext int    `json:"force_fail_next"` // 强制下 N 次返回 5xx
	RejectAll     bool   `json:"reject_all"`      // 全部拒绝（500）
	ConsecutiveFailUntilSuccess int `json:"consecutive_fail_until_success"` // 前 N 次失败，第 N+1 次成功
}

// InboxEntry inbox 中的一条记录
type InboxEntry struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Body           json.RawMessage `json:"body"`
	Signature      string          `json:"signature"`
	ReceivedAt     time.Time       `json:"received_at"`
	HTTPStatus     int             `json:"http_status"`
}

// Server Mock Partner 服务
type Server struct {
	mu     sync.Mutex
	cfg    Config
	inbox  map[string]InboxEntry // idempotency_key -> entry（命中即幂等）
	failCnt int                  // ForceFailNext 计数
	totalCalls int               // 总请求计数
}

// NewServer 构造（默认 secret = "test-secret-key"）
func NewServer() *Server {
	return &Server{
		cfg:   Config{HMACSecret: "test-secret-key"},
		inbox: make(map[string]InboxEntry),
	}
}

// Mux 返回 http.Handler，方便挂在主 server 上或 httptest 启动
func (s *Server) Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/partner/orders", s.handleOrders)
	mux.HandleFunc("/partner/orders/by-idempotency-key/", s.handleProbe)
	mux.HandleFunc("/partner/_admin/config", s.handleAdminConfig)
	mux.HandleFunc("/partner/_admin/inbox", s.handleAdminInbox)
	mux.HandleFunc("/partner/_admin/reset", s.handleAdminReset)
	return mux
}

// Start 启动监听（blocking）
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.Mux())
}

// StartTest 启动 httptest server（测试用）
func (s *Server) StartTest() *httptest.Server { return httptest.NewServer(s.Mux()) }

// ---------- handlers ----------

// handleOrders POST /partner/orders
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	idem := r.Header.Get("X-Idempotency-Key")
	sig := r.Header.Get("X-Signature")

	s.mu.Lock()
	cfg := s.cfg
	delayMs := cfg.DelayMs
	s.totalCalls++
	calls := s.totalCalls
	s.mu.Unlock()

	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// 1) HMAC 校验：按 raw body + secret 计算 sha256
	expected := signHMAC(body, cfg.HMACSecret)
	if sig == "" || !hmac.Equal([]byte(expected), []byte(sig)) {
		s.recordInbox(idem, body, sig, http.StatusUnauthorized)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// 2) 幂等：同 key 已存在直接返回 200
	s.mu.Lock()
	if e, ok := s.inbox[idem]; ok && e.HTTPStatus < 300 {
		s.mu.Unlock()
		_, _ = w.Write([]byte(`{"ok":true,"duplicate":true}`))
		return
	}
	s.mu.Unlock()

	// 3) 故障注入
	if cfg.RejectAll {
		s.recordInbox(idem, body, sig, http.StatusInternalServerError)
		http.Error(w, "rejected", http.StatusInternalServerError)
		return
	}
	if cfg.ForceFailNext > 0 {
		s.mu.Lock()
		s.failCnt++
		shouldFail := s.failCnt <= cfg.ForceFailNext
		s.mu.Unlock()
		if shouldFail {
			s.recordInbox(idem, body, sig, http.StatusInternalServerError)
			http.Error(w, "forced fail", http.StatusInternalServerError)
			return
		}
	}
	if cfg.ConsecutiveFailUntilSuccess > 0 && calls <= cfg.ConsecutiveFailUntilSuccess {
		s.recordInbox(idem, body, sig, http.StatusInternalServerError)
		http.Error(w, "transient fail", http.StatusInternalServerError)
		return
	}

	// 4) 成功
	s.recordInbox(idem, body, sig, http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// handleProbe GET /partner/orders/by-idempotency-key/:key
func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	prefix := "/partner/orders/by-idempotency-key/"
	key := r.URL.Path[len(prefix):]
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	e, ok := s.inbox[key]
	s.mu.Unlock()
	if !ok || e.HTTPStatus >= 300 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"idempotency_key": key,
		"received_at":     e.ReceivedAt,
		"body":            json.RawMessage(e.Body),
	})
}

// handleAdminConfig POST /partner/_admin/config
func (s *Server) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var c Config
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "decode: "+err.Error(), http.StatusBadRequest)
		return
	}
	if c.HMACSecret == "" {
		c.HMACSecret = "test-secret-key"
	}
	s.mu.Lock()
	s.cfg = c
	s.failCnt = 0
	s.totalCalls = 0
	s.mu.Unlock()
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// handleAdminInbox GET /partner/_admin/inbox
func (s *Server) handleAdminInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	out := make([]InboxEntry, 0, len(s.inbox))
	for _, e := range s.inbox {
		out = append(out, e)
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out, "total": strconv.Itoa(len(out))})
}

// handleAdminReset POST /partner/_admin/reset
func (s *Server) handleAdminReset(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.inbox = make(map[string]InboxEntry)
	s.cfg = Config{HMACSecret: "test-secret-key"}
	s.failCnt = 0
	s.totalCalls = 0
	s.mu.Unlock()
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// ---------- 辅助 ----------

func (s *Server) recordInbox(key string, body []byte, sig string, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return // 没有 key 不记录
	}
	// 仅在成功时覆盖；失败状态不应替换已有的成功幂等记录
	if existing, ok := s.inbox[key]; ok && existing.HTTPStatus < 300 && status >= 300 {
		return
	}
	cp := make(json.RawMessage, len(body))
	copy(cp, body)
	s.inbox[key] = InboxEntry{
		IdempotencyKey: key,
		Body:           cp,
		Signature:      sig,
		ReceivedAt:     time.Now(),
		HTTPStatus:     status,
	}
}

// signHMAC 计算 sha256-HMAC（hex）
func signHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// SignBody 暴露给场景层使用：用相同算法签名
func SignBody(body []byte, secret string) string { return signHMAC(body, secret) }

// 帮助调用方组装签名 body 的工具：避免使用方依赖 partner 内部实现
var _ = bytes.NewReader // keep imports stable
