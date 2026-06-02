package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultDoubaoTimeout = 8 * time.Second

// DoubaoOptions configures a Doubao OpenAI-compatible provider.
type DoubaoOptions struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// DoubaoProvider calls Volcengine Ark through the OpenAI-compatible API.
type DoubaoProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewDoubaoProvider constructs a Doubao provider.
func NewDoubaoProvider(opts DoubaoOptions) *DoubaoProvider {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultDoubaoTimeout
	}
	return &DoubaoProvider{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		apiKey:  opts.APIKey,
		model:   opts.Model,
		client:  &http.Client{Timeout: timeout},
	}
}

// Name returns the provider name.
func (p *DoubaoProvider) Name() string { return "doubao" }

type doubaoResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Chat calls /chat/completions using the OpenAI-compatible request shape.
func (p *DoubaoProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.model
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint, err := url.JoinPath(p.baseURL, "chat/completions")
	if err != nil {
		log.Printf("provider=doubao event=build_endpoint_failed base_url=%q err=%v", p.baseURL, err)
		return nil, fmt.Errorf("build endpoint: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("provider=doubao event=new_request_failed endpoint=%q err=%v", endpoint, err)
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	start := time.Now()
	log.Printf("provider=doubao event=request_start method=%s endpoint=%q model=%q messages=%d max_tokens=%d", http.MethodPost, endpoint, req.Model, len(req.Messages), req.MaxTokens)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		elapsed := time.Since(start)
		var ne interface{ Timeout() bool }
		if errors.As(err, &ne) && ne.Timeout() {
			log.Printf("provider=doubao event=request_failed category=timeout endpoint=%q elapsed_ms=%d err=%v", endpoint, elapsed.Milliseconds(), err)
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Printf("provider=doubao event=request_failed category=timeout endpoint=%q elapsed_ms=%d err=%v", endpoint, elapsed.Milliseconds(), err)
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		log.Printf("provider=doubao event=request_failed category=network endpoint=%q elapsed_ms=%d err=%v", endpoint, elapsed.Milliseconds(), err)
		return nil, fmt.Errorf("%w: %v", ErrUpstreamServer, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	elapsed := time.Since(start)
	log.Printf("provider=doubao event=response_received status=%d elapsed_ms=%d bytes=%d", resp.StatusCode, elapsed.Milliseconds(), len(respBody))
	if resp.StatusCode >= 500 {
		log.Printf("provider=doubao event=response_error category=server status=%d elapsed_ms=%d body=%q", resp.StatusCode, elapsed.Milliseconds(), snippet(respBody))
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUpstreamServer, resp.StatusCode, snippet(respBody))
	}
	if resp.StatusCode >= 400 {
		log.Printf("provider=doubao event=response_error category=client status=%d elapsed_ms=%d body=%q", resp.StatusCode, elapsed.Milliseconds(), snippet(respBody))
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUpstreamClient, resp.StatusCode, snippet(respBody))
	}

	var parsed doubaoResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		log.Printf("provider=doubao event=invalid_response reason=json_unmarshal status=%d elapsed_ms=%d err=%v body=%q", resp.StatusCode, elapsed.Milliseconds(), err, snippet(respBody))
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(parsed.Choices) == 0 {
		log.Printf("provider=doubao event=invalid_response reason=empty_choices status=%d elapsed_ms=%d body=%q", resp.StatusCode, elapsed.Milliseconds(), snippet(respBody))
		return nil, fmt.Errorf("%w: choices empty", ErrInvalidResponse)
	}
	log.Printf("provider=doubao event=request_success status=%d elapsed_ms=%d input_tokens=%d output_tokens=%d", resp.StatusCode, elapsed.Milliseconds(), parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens)
	return &ChatResponse{
		Content:      parsed.Choices[0].Message.Content,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}

func snippet(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
