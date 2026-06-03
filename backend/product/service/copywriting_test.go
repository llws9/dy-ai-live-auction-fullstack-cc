package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	sharedllm "shared/llm"
)

type fakeProvider struct {
	respContent string
	err         error
	gotReq      *sharedllm.ChatRequest
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Chat(ctx context.Context, req *sharedllm.ChatRequest) (*sharedllm.ChatResponse, error) {
	f.gotReq = req
	if f.err != nil {
		return nil, f.err
	}
	return &sharedllm.ChatResponse{Content: f.respContent, InputTokens: 1, OutputTokens: 1}, nil
}

type fakeCategoryResolver struct{ names map[int64]string }

func (f *fakeCategoryResolver) GetNameByID(ctx context.Context, id int64) (string, bool, error) {
	name, ok := f.names[id]
	return name, ok, nil
}

func newRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestCopywriting_Generate_Success(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"二手iPhone 12","description":"九成新自用 无暗病 原装电池","selling_points":["九成新","原装电池","无暗病"],"suggested_start_price":"1999"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{names: map[int64]string{1: "手机数码"}}, newRedis(t), "doubao-1.5-vision-pro")

	resp, err := svc.Generate(context.Background(), 100, &CopywritingRequest{
		Images:     []string{"https://cdn.example.com/a.jpg"},
		CategoryID: int64Ptr(1),
		Keywords:   "九成新",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Name != "二手iPhone 12" {
		t.Fatalf("Name mapping failed: %q", resp.Name)
	}
	if resp.SuggestedStartPrice != "1999" {
		t.Fatalf("price mapping failed: %q", resp.SuggestedStartPrice)
	}
	if len(fp.gotReq.Messages) < 2 {
		t.Fatalf("expect system + user messages, got %d", len(fp.gotReq.Messages))
	}
	if fp.gotReq.ResponseFormat == nil || fp.gotReq.ResponseFormat.Type != "json_object" {
		t.Fatalf("response_format must be json_object")
	}
	var joined string
	for _, part := range fp.gotReq.Messages[1].Content {
		joined += part.Text
	}
	if !strings.Contains(joined, "手机数码") {
		t.Fatalf("category name should be in prompt, got %q", joined)
	}
}

func TestCopywriting_Generate_EmptyImages_400(t *testing.T) {
	fp := &fakeProvider{}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: nil})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func TestCopywriting_Generate_TooManyImages_400(t *testing.T) {
	fp := &fakeProvider{}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	imgs := make([]string, 7)
	for i := range imgs {
		imgs[i] = "https://cdn.example.com/x.jpg"
	}
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: imgs})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func TestCopywriting_Generate_RateLimited_429(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"x","description":"y","selling_points":["a","b","c"],"suggested_start_price":"1"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	for i := 0; i < 5; i++ {
		_, err := svc.Generate(context.Background(), 100, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
		if err != nil {
			t.Fatalf("call %d unexpected err: %v", i, err)
		}
	}
	_, err := svc.Generate(context.Background(), 100, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
}

func TestCopywriting_Generate_NilRedis_FailOpen(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"x","description":"y","selling_points":["a"],"suggested_start_price":"1"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, nil, "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if err != nil {
		t.Fatalf("nil redis should fail-open, got %v", err)
	}
}

func TestCopywriting_Generate_UpstreamFail_502(t *testing.T) {
	fp := &fakeProvider{err: sharedllm.ErrUpstreamServer}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrUpstreamFailed) {
		t.Fatalf("want ErrUpstreamFailed, got %v", err)
	}
}

func TestCopywriting_Generate_UpstreamTimeout(t *testing.T) {
	fp := &fakeProvider{err: sharedllm.ErrUpstreamTimeout}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrUpstreamTimeout) {
		t.Fatalf("want ErrUpstreamTimeout, got %v", err)
	}
}

func TestCopywriting_Generate_BadJSON_502InvalidOutput(t *testing.T) {
	fp := &fakeProvider{respContent: "not a json"}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("want ErrInvalidOutput, got %v", err)
	}
}

func TestCopywriting_Generate_PriceNotNumber_502(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"x","description":"y","selling_points":["a"],"suggested_start_price":"abc"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("want ErrInvalidOutput, got %v", err)
	}
}

func TestCopywriting_Generate_CategoryNotExists_400(t *testing.T) {
	fp := &fakeProvider{respContent: `{}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{names: map[int64]string{}}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{
		Images:     []string{"https://cdn.example.com/a.jpg"},
		CategoryID: int64Ptr(99),
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func int64Ptr(v int64) *int64 { return &v }
