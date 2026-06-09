package llm

import (
	"context"
	"errors"
)

// Error categories used by business services to map upstream failures.
var (
	ErrUpstreamTimeout    = errors.New("llm upstream timeout")
	ErrUpstreamClient     = errors.New("llm upstream client error")
	ErrUpstreamServer     = errors.New("llm upstream server error")
	ErrImageUnavailable   = errors.New("llm image unavailable")
	ErrInvalidResponse    = errors.New("llm invalid response")
	ErrMissingCredentials = errors.New("llm missing credentials")
)

// ChatMessage is an OpenAI-compatible multimodal chat message.
type ChatMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// ContentPart is a multimodal content fragment, either text or image.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type ResponseFormat struct {
	Type string `json:"type"` // "json_object"
}

type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Temperature    float32         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
}

// Provider abstracts an LLM provider for multiple implementations and tests.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
