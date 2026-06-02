package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		return nil, fmt.Errorf("build endpoint: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		var ne interface{ Timeout() bool }
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrUpstreamServer, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUpstreamServer, resp.StatusCode, snippet(respBody))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUpstreamClient, resp.StatusCode, snippet(respBody))
	}

	var parsed doubaoResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("%w: choices empty", ErrInvalidResponse)
	}
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
