package partner

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// 工具：组装 body + 计算签名
func mkPost(t *testing.T, url string, body []byte, idem, secret string, tamper bool) *http.Response {
	t.Helper()
	sig := SignBody(body, secret)
	if tamper {
		sig = strings.Repeat("0", len(sig))
	}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("X-Idempotency-Key", idem)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post err: %v", err)
	}
	return resp
}

// 1) 正常投递：HMAC 正确 → 200 + inbox 命中
func TestServer_NormalDelivery(t *testing.T) {
	s := NewServer()
	ts := s.StartTest()
	defer ts.Close()

	body := []byte(`{"order_id":1001,"price":199.0}`)
	resp := mkPost(t, ts.URL+"/partner/orders", body, "key-normal-1", "test-secret-key", false)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}
	resp.Body.Close()

	// inbox 应有 1 条
	r2, _ := http.Get(ts.URL + "/partner/_admin/inbox")
	defer r2.Body.Close()
	var out struct {
		Items []InboxEntry `json:"items"`
	}
	_ = json.NewDecoder(r2.Body).Decode(&out)
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 inbox entry, got %d", len(out.Items))
	}
}

// 2) 探测：成功投递后 GET by-idempotency-key 命中
func TestServer_ProbeHitAfterDelivery(t *testing.T) {
	s := NewServer()
	ts := s.StartTest()
	defer ts.Close()

	body := []byte(`{"order_id":1002}`)
	resp := mkPost(t, ts.URL+"/partner/orders", body, "key-probe", "test-secret-key", false)
	resp.Body.Close()

	r, _ := http.Get(ts.URL + "/partner/orders/by-idempotency-key/key-probe")
	if r.StatusCode != 200 {
		t.Fatalf("expected 200 probe, got %d", r.StatusCode)
	}
	r.Body.Close()

	r2, _ := http.Get(ts.URL + "/partner/orders/by-idempotency-key/key-not-exists")
	if r2.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", r2.StatusCode)
	}
	r2.Body.Close()
}

// 3) 重复回调幂等：5 次相同 key → inbox 仍只 1 条
func TestServer_DuplicateIdempotent(t *testing.T) {
	s := NewServer()
	ts := s.StartTest()
	defer ts.Close()

	body := []byte(`{"order_id":1003}`)
	for i := 0; i < 5; i++ {
		r := mkPost(t, ts.URL+"/partner/orders", body, "key-dup", "test-secret-key", false)
		r.Body.Close()
	}
	r, _ := http.Get(ts.URL + "/partner/_admin/inbox")
	defer r.Body.Close()
	var out struct {
		Items []InboxEntry `json:"items"`
	}
	_ = json.NewDecoder(r.Body).Decode(&out)
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 entry under same idem key, got %d", len(out.Items))
	}
}

// 4) HMAC 篡改：返回 401
func TestServer_TamperedSignatureRejected(t *testing.T) {
	s := NewServer()
	ts := s.StartTest()
	defer ts.Close()

	body := []byte(`{"order_id":1004}`)
	resp := mkPost(t, ts.URL+"/partner/orders", body, "key-bad-sig", "test-secret-key", true)
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// 5) 强制失败 N 次后成功（DLQ / 最终一致路径）
func TestServer_ForceFailUntilSuccess(t *testing.T) {
	s := NewServer()
	ts := s.StartTest()
	defer ts.Close()

	cfgBody := []byte(`{"hmac_secret":"test-secret-key","consecutive_fail_until_success":3}`)
	rc, _ := http.Post(ts.URL+"/partner/_admin/config", "application/json", bytes.NewReader(cfgBody))
	rc.Body.Close()

	body := []byte(`{"order_id":1005}`)
	statuses := []int{}
	for i := 0; i < 4; i++ {
		r := mkPost(t, ts.URL+"/partner/orders", body, "key-flaky", "test-secret-key", false)
		statuses = append(statuses, r.StatusCode)
		r.Body.Close()
	}
	// 前 3 次应失败，第 4 次成功
	if statuses[0] != 500 || statuses[1] != 500 || statuses[2] != 500 || statuses[3] != 200 {
		t.Fatalf("expected [500,500,500,200], got %v", statuses)
	}
}

// 6) admin reset 清空 inbox 和配置
func TestServer_AdminReset(t *testing.T) {
	s := NewServer()
	ts := s.StartTest()
	defer ts.Close()

	body := []byte(`{"order_id":1006}`)
	r := mkPost(t, ts.URL+"/partner/orders", body, "key-reset", "test-secret-key", false)
	r.Body.Close()

	rr, _ := http.Post(ts.URL+"/partner/_admin/reset", "application/json", nil)
	rr.Body.Close()

	r2, _ := http.Get(ts.URL + "/partner/_admin/inbox")
	defer r2.Body.Close()
	var out struct {
		Items []InboxEntry `json:"items"`
	}
	_ = json.NewDecoder(r2.Body).Decode(&out)
	if len(out.Items) != 0 {
		t.Fatalf("expected reset clears inbox, got %d", len(out.Items))
	}
}
