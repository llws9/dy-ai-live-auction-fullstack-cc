package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"

	"product-service/service"
)

type stubCopySvc struct {
	resp      *service.CopywritingResponse
	err       error
	gotUserID int64
}

func (s *stubCopySvc) Generate(ctx context.Context, userID int64, req *service.CopywritingRequest) (*service.CopywritingResponse, error) {
	s.gotUserID = userID
	return s.resp, s.err
}

func setupCopyRouter(t *testing.T, svc CopywritingServiceAPI) *server.Hertz {
	t.Helper()
	h := server.New(server.WithExitWaitTime(0))
	hh := NewCopywritingHandler(svc)
	h.POST("/api/v1/products/ai/copywriting", func(c context.Context, ctx *app.RequestContext) {
		hh.Generate(c, ctx)
	})
	return h
}

func mustBody(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestCopyHandler_Success_200(t *testing.T) {
	svc := &stubCopySvc{resp: &service.CopywritingResponse{Name: "x", Description: "y", SellingPoints: []string{"a"}, SuggestedStartPrice: "1"}}
	h := setupCopyRouter(t, svc)

	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "streamer"},
	)
	if w.Result().StatusCode() != http.StatusOK {
		t.Fatalf("status want=200 got=%d body=%s", w.Result().StatusCode(), w.Result().Body())
	}
	if svc.gotUserID != 100 {
		t.Fatalf("service userID want=100 got=%d", svc.gotUserID)
	}
}

func TestCopyHandler_ForbiddenRole_403(t *testing.T) {
	svc := &stubCopySvc{}
	h := setupCopyRouter(t, svc)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "user"},
	)
	if w.Result().StatusCode() != http.StatusForbidden {
		t.Fatalf("status want=403 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_BadRequest_400(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrInvalidRequest}
	h := setupCopyRouter(t, svc)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "streamer"},
	)
	if w.Result().StatusCode() != http.StatusBadRequest {
		t.Fatalf("status want=400 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_RateLimited_429(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrRateLimited}
	h := setupCopyRouter(t, svc)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "merchant"},
	)
	if w.Result().StatusCode() != http.StatusTooManyRequests {
		t.Fatalf("status want=429 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_Upstream_502(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrUpstreamFailed}
	h := setupCopyRouter(t, svc)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "admin"},
	)
	if w.Result().StatusCode() != http.StatusBadGateway {
		t.Fatalf("status want=502 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_NotConfigured_503(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrNotConfigured}
	h := setupCopyRouter(t, svc)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "merchant"},
	)
	if w.Result().StatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("status want=503 got=%d body=%s", w.Result().StatusCode(), w.Result().Body())
	}
}

func TestCopyHandler_Timeout_504(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrUpstreamTimeout}
	h := setupCopyRouter(t, svc)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "streamer"},
	)
	if w.Result().StatusCode() != http.StatusGatewayTimeout {
		t.Fatalf("status want=504 got=%d", w.Result().StatusCode())
	}
}
