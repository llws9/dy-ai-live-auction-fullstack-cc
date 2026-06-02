package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	sharedllm "shared/llm"
)

// 业务错误（handler 据此映射 HTTP 状态码）。
var (
	ErrInvalidRequest  = errors.New("copywriting: invalid request")
	ErrRateLimited     = errors.New("copywriting: rate limited")
	ErrUpstreamFailed  = errors.New("copywriting: upstream failed")
	ErrUpstreamTimeout = errors.New("copywriting: upstream timeout")
	ErrInvalidOutput   = errors.New("copywriting: invalid llm output")
)

const (
	maxImages       = 6
	maxKeywordsLen  = 100
	rateLimitPerMin = 5
	rateWindowSec   = 120
)

// CategoryNameResolver 抽象"按 ID 取类目名 + 存在性"查询，便于测试。
type CategoryNameResolver interface {
	GetNameByID(ctx context.Context, id int64) (name string, ok bool, err error)
}

// CopywritingRequest API 入参。
type CopywritingRequest struct {
	Images     []string `json:"images" binding:"required,min=1,max=6"`
	CategoryID *int64   `json:"category_id,omitempty"`
	Keywords   string   `json:"keywords,omitempty"`
}

// CopywritingResponse API 响应。
type CopywritingResponse struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	SellingPoints       []string `json:"selling_points"`
	SuggestedStartPrice string   `json:"suggested_start_price"`
}

// CopywritingService 编排 AI 文案生成流程。
type CopywritingService struct {
	provider     sharedllm.Provider
	categoryRes  CategoryNameResolver
	redis        *redis.Client
	defaultModel string
	now          func() time.Time
}

// NewCopywritingService 构造 service；redis 允许为 nil，此时限流 fail-open。
func NewCopywritingService(p sharedllm.Provider, c CategoryNameResolver, r *redis.Client, defaultModel string) *CopywritingService {
	return &CopywritingService{
		provider:     p,
		categoryRes:  c,
		redis:        r,
		defaultModel: defaultModel,
		now:          time.Now,
	}
}

// Generate 生成商品文案草稿。
func (s *CopywritingService) Generate(ctx context.Context, userID int64, req *CopywritingRequest) (*CopywritingResponse, error) {
	categoryName, err := s.validate(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.checkRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	resp, err := s.provider.Chat(ctx, s.buildChatRequest(req, categoryName))
	if err != nil {
		if errors.Is(err, sharedllm.ErrUpstreamTimeout) {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
	}

	parsed, err := parseCopywritingOutput(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return parsed, nil
}

func (s *CopywritingService) validate(ctx context.Context, req *CopywritingRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("%w: request nil", ErrInvalidRequest)
	}
	if len(req.Images) == 0 {
		return "", fmt.Errorf("%w: images empty", ErrInvalidRequest)
	}
	if len(req.Images) > maxImages {
		return "", fmt.Errorf("%w: too many images (>%d)", ErrInvalidRequest, maxImages)
	}
	for _, u := range req.Images {
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return "", fmt.Errorf("%w: invalid image url %q", ErrInvalidRequest, u)
		}
	}
	if len([]rune(req.Keywords)) > maxKeywordsLen {
		return "", fmt.Errorf("%w: keywords too long", ErrInvalidRequest)
	}

	if req.CategoryID == nil {
		return "", nil
	}
	if s.categoryRes == nil {
		return "", fmt.Errorf("%w: category resolver missing", ErrInvalidRequest)
	}
	name, ok, err := s.categoryRes.GetNameByID(ctx, *req.CategoryID)
	if err != nil {
		return "", fmt.Errorf("%w: category lookup: %v", ErrInvalidRequest, err)
	}
	if !ok {
		return "", fmt.Errorf("%w: category %d not exists", ErrInvalidRequest, *req.CategoryID)
	}
	return name, nil
}

func (s *CopywritingService) checkRateLimit(ctx context.Context, userID int64) error {
	if s.redis == nil {
		return nil
	}
	key := fmt.Sprintf("ai:copywriting:%d:%s", userID, s.now().UTC().Format("200601021504"))
	n, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = s.redis.Expire(ctx, key, rateWindowSec*time.Second).Err()
	}
	if n > int64(rateLimitPerMin) {
		return fmt.Errorf("%w: %d/min exceeded", ErrRateLimited, rateLimitPerMin)
	}
	return nil
}

const systemPrompt = `你是直播竞拍平台的商品文案专家。请根据图片和卖家提供的关键词，生成商品的：
1. name: ≤30字标题，含品类与关键卖点
2. description: 80-150字描述，分点列卖点
3. selling_points: 3-5个短语，每个≤12字
4. suggested_start_price: 起拍价建议（人民币元，纯数字字符串，参考二手市场行情，保守偏低 30%-50%）

严格输出 JSON，schema：
{"name":"","description":"","selling_points":[],"suggested_start_price":""}
不要任何额外解释、不要 markdown 代码块。`

func (s *CopywritingService) buildChatRequest(req *CopywritingRequest, categoryName string) *sharedllm.ChatRequest {
	parts := make([]sharedllm.ContentPart, 0, len(req.Images)+1)
	for _, u := range req.Images {
		parts = append(parts, sharedllm.ContentPart{
			Type:     "image_url",
			ImageURL: &sharedllm.ImageURL{URL: u},
		})
	}

	var info []string
	if categoryName != "" {
		info = append(info, "类目: "+categoryName)
	}
	if strings.TrimSpace(req.Keywords) != "" {
		info = append(info, "关键词: "+req.Keywords)
	}
	if len(info) == 0 {
		info = append(info, "请基于图片生成商品文案。")
	}
	parts = append(parts, sharedllm.ContentPart{Type: "text", Text: strings.Join(info, "\n")})

	return &sharedllm.ChatRequest{
		Model: s.defaultModel,
		Messages: []sharedllm.ChatMessage{
			{Role: "system", Content: []sharedllm.ContentPart{{Type: "text", Text: systemPrompt}}},
			{Role: "user", Content: parts},
		},
		Temperature:    0.6,
		MaxTokens:      600,
		ResponseFormat: &sharedllm.ResponseFormat{Type: "json_object"},
	}
}

func parseCopywritingOutput(raw string) (*CopywritingResponse, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var out CopywritingResponse
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if out.Name == "" || out.Description == "" || len(out.SellingPoints) == 0 || out.SuggestedStartPrice == "" {
		return nil, fmt.Errorf("missing required field")
	}
	if _, err := decimal.NewFromString(out.SuggestedStartPrice); err != nil {
		return nil, fmt.Errorf("price not a number: %w", err)
	}
	return &out, nil
}
