package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"product-service/dao"
	"product-service/model"
)

type AuctionRuleTemplateService struct {
	dao *dao.AuctionRuleTemplateDAO
}

func NewAuctionRuleTemplateService(dao *dao.AuctionRuleTemplateDAO) *AuctionRuleTemplateService {
	return &AuctionRuleTemplateService{dao: dao}
}

type CreateAuctionRuleTemplateRequest struct {
	Name               string `json:"name"`
	StartPrice         string `json:"start_price"`
	Increment          string `json:"increment"`
	CapPrice           string `json:"cap_price,omitempty"`
	Duration           int    `json:"duration"`
	DelayDuration      int    `json:"delay_duration,omitempty"`
	MaxDelayTime       int    `json:"max_delay_time,omitempty"`
	TriggerDelayBefore int    `json:"trigger_delay_before,omitempty"`
	IsDefault          bool   `json:"is_default"`
}

type UpdateAuctionRuleTemplateRequest = CreateAuctionRuleTemplateRequest

type AuctionRuleTemplateResponse struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	StartPrice         string `json:"start_price"`
	Increment          string `json:"increment"`
	CapPrice           string `json:"cap_price,omitempty"`
	Duration           int    `json:"duration"`
	DelayDuration      int    `json:"delay_duration"`
	MaxDelayTime       int    `json:"max_delay_time"`
	TriggerDelayBefore int    `json:"trigger_delay_before"`
	IsDefault          bool   `json:"is_default"`
}

func (s *AuctionRuleTemplateService) Create(ctx context.Context, ownerID int64, req CreateAuctionRuleTemplateRequest) (*AuctionRuleTemplateResponse, error) {
	item, err := buildAuctionRuleTemplate(ownerID, req)
	if err != nil {
		return nil, err
	}
	if err := s.dao.Create(ctx, item); err != nil {
		return nil, err
	}
	return toAuctionRuleTemplateResponse(item), nil
}

func (s *AuctionRuleTemplateService) List(ctx context.Context, ownerID int64, page, pageSize int) ([]AuctionRuleTemplateResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := s.dao.ListByOwner(ctx, ownerID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	res := make([]AuctionRuleTemplateResponse, 0, len(items))
	for i := range items {
		res = append(res, *toAuctionRuleTemplateResponse(&items[i]))
	}
	return res, total, nil
}

func (s *AuctionRuleTemplateService) Get(ctx context.Context, ownerID, id int64) (*AuctionRuleTemplateResponse, error) {
	item, err := s.dao.GetByIDAndOwner(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	return toAuctionRuleTemplateResponse(item), nil
}

func (s *AuctionRuleTemplateService) Update(ctx context.Context, ownerID, id int64, req UpdateAuctionRuleTemplateRequest) (*AuctionRuleTemplateResponse, error) {
	item, err := s.dao.GetByIDAndOwner(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	next, err := buildAuctionRuleTemplate(ownerID, CreateAuctionRuleTemplateRequest(req))
	if err != nil {
		return nil, err
	}
	next.ID = item.ID
	next.CreatedAt = item.CreatedAt
	if err := s.dao.Update(ctx, next); err != nil {
		return nil, err
	}
	return toAuctionRuleTemplateResponse(next), nil
}

func (s *AuctionRuleTemplateService) Delete(ctx context.Context, ownerID, id int64) error {
	return s.dao.DeleteByIDAndOwner(ctx, id, ownerID)
}

func buildAuctionRuleTemplate(ownerID int64, req CreateAuctionRuleTemplateRequest) (*model.AuctionRuleTemplate, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("模板名称不能为空")
	}
	if req.Duration <= 0 {
		return nil, errors.New("竞拍时长必须大于 0")
	}
	startPriceRaw := req.StartPrice
	if strings.TrimSpace(startPriceRaw) == "" {
		startPriceRaw = "0"
	}
	startPrice, err := parseMoney2(startPriceRaw, "起拍价")
	if err != nil {
		return nil, err
	}
	increment, err := parseMoney2(req.Increment, "加价幅度")
	if err != nil {
		return nil, err
	}
	if increment.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("加价幅度必须大于 0")
	}
	var capPrice *decimal.Decimal
	if strings.TrimSpace(req.CapPrice) != "" {
		cap, err := parseMoney2(req.CapPrice, "封顶价")
		if err != nil {
			return nil, err
		}
		capPrice = &cap
	}
	delayDuration := defaultPositive(req.DelayDuration, 30)
	maxDelayTime := defaultPositive(req.MaxDelayTime, 180)
	triggerDelayBefore := defaultPositive(req.TriggerDelayBefore, 30)
	return &model.AuctionRuleTemplate{
		OwnerID:            ownerID,
		Name:               name,
		StartPrice:         startPrice,
		Increment:          increment,
		CapPrice:           capPrice,
		Duration:           req.Duration,
		DelayDuration:      delayDuration,
		MaxDelayTime:       maxDelayTime,
		TriggerDelayBefore: triggerDelayBefore,
		IsDefault:          req.IsDefault,
	}, nil
}

func parseMoney2(raw string, field string) (decimal.Decimal, error) {
	v, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return decimal.Zero, fmt.Errorf("%s金额格式错误", field)
	}
	if !v.Equal(v.Round(2)) {
		return decimal.Zero, fmt.Errorf("%s金额最多支持两位小数", field)
	}
	if v.IsNegative() {
		return decimal.Zero, fmt.Errorf("%s金额不能为负数", field)
	}
	return v, nil
}

func defaultPositive(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func toAuctionRuleTemplateResponse(item *model.AuctionRuleTemplate) *AuctionRuleTemplateResponse {
	res := &AuctionRuleTemplateResponse{
		ID:                 item.ID,
		Name:               item.Name,
		StartPrice:         item.StartPrice.StringFixed(2),
		Increment:          item.Increment.StringFixed(2),
		Duration:           item.Duration,
		DelayDuration:      item.DelayDuration,
		MaxDelayTime:       item.MaxDelayTime,
		TriggerDelayBefore: item.TriggerDelayBefore,
		IsDefault:          item.IsDefault,
	}
	if item.CapPrice != nil {
		res.CapPrice = item.CapPrice.StringFixed(2)
	}
	return res
}
