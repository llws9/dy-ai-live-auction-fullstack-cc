package llm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
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
	logs := captureLogs(t)
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
	if got := logs.String(); !strings.Contains(got, "provider=doubao") || !strings.Contains(got, "event=request_start") || !strings.Contains(got, "event=request_success") {
		t.Fatalf("logs should contain provider/start/success, got %q", got)
	}
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})
	return &buf
}

func TestDoubao_Chat_Timeout(t *testing.T) {
	logs := captureLogs(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	p := NewDoubaoProvider(DoubaoOptions{
		BaseURL: srv.URL, APIKey: "k", Model: "m", Timeout: 50 * time.Millisecond,
	})

	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrUpstreamTimeout) {
		t.Fatalf("want ErrUpstreamTimeout, got %v", err)
	}
	if got := logs.String(); !strings.Contains(got, "provider=doubao") || !strings.Contains(got, "event=request_failed") || !strings.Contains(got, "category=timeout") {
		t.Fatalf("logs should contain provider/event/category, got %q", got)
	}
}

func TestDoubao_Chat_4xx(t *testing.T) {
	logs := captureLogs(t)
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"bad key"}`)
	})

	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrUpstreamClient) {
		t.Fatalf("want ErrUpstreamClient, got %v", err)
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err should contain status 401, got %v", err)
	}
	if got := logs.String(); !strings.Contains(got, "provider=doubao") || !strings.Contains(got, "event=response_error") || !strings.Contains(got, "status=401") {
		t.Fatalf("logs should contain provider/event/status, got %q", got)
	}
}

func TestDoubao_Chat_5xx(t *testing.T) {
	logs := captureLogs(t)
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrUpstreamServer) {
		t.Fatalf("want ErrUpstreamServer, got %v", err)
	}
	if got := logs.String(); !strings.Contains(got, "provider=doubao") || !strings.Contains(got, "event=response_error") || !strings.Contains(got, "status=502") {
		t.Fatalf("logs should contain provider/event/status, got %q", got)
	}
}

func TestDoubao_Chat_EmptyChoices(t *testing.T) {
	logs := captureLogs(t)
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[]}`)
	})

	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("want ErrInvalidResponse, got %v", err)
	}
	if got := logs.String(); !strings.Contains(got, "provider=doubao") || !strings.Contains(got, "event=invalid_response") || !strings.Contains(got, "reason=empty_choices") {
		t.Fatalf("logs should contain provider/event/reason, got %q", got)
	}
}
