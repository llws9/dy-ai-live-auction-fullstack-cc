package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newDoubaoTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *DoubaoProvider) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p := NewDoubaoProvider(DoubaoOptions{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "doubao-1.5-vision-pro",
		Timeout: 2 * time.Second,
	})
	return srv, p
}

func TestDoubao_Chat_Success(t *testing.T) {
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization want=Bearer test-key got=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type want=application/json got=%q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"doubao-1.5-vision-pro"`) {
			t.Fatalf("body missing model: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"choices":[{"message":{"content":"hello"}}],
			"usage":{"prompt_tokens":10,"completion_tokens":5}
		}`)
	})

	resp, err := p.Chat(context.Background(), &ChatRequest{
		Model:    "doubao-1.5-vision-pro",
		Messages: []ChatMessage{{Role: "user", Content: []ContentPart{{Type: "text", Text: "hi"}}}},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("Content want=hello got=%q", resp.Content)
	}
	if resp.InputTokens != 10 || resp.OutputTokens != 5 {
		t.Fatalf("tokens want=10/5 got=%d/%d", resp.InputTokens, resp.OutputTokens)
	}
	_ = errors.Is // Reserved for Task 4 error-path tests.
}
