package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"product-service/dao"
	"product-service/model"
)

// ErrAdminDAOMissing 当 admin DAO 未注入而调用 admin 单条查询时返回，便于上层映射成 5xx。
var ErrAdminDAOMissing = errors.New("admin order DAO is not configured")

// OrderAdminVO 是 admin 端订单返回视图。
//   - user_id 与 winner_id 同值，仅为前端 OrderList 的 user_name fallback 兼容字段；
//   - product_image 取 products.images 数组首图（mysql 上是 JSON，sqlite 上是字符串，统一按字符串解析）；
//   - user_name 留空，前端通过 fallback 渲染 `用户 #${user_id}`，不在此处跨库 JOIN auction 服务的 users 表。
type OrderAdminVO struct {
	ID           int64             `json:"id"`
	AuctionID    int64             `json:"auction_id"`
	ProductID    int64             `json:"product_id"`
	ProductName  string            `json:"product_name"`
	ProductImage string            `json:"product_image"`
	WinnerID     int64             `json:"winner_id"`
	UserID       int64             `json:"user_id"`
	FinalPrice   decimal.Decimal   `json:"final_price"`
	Status       model.OrderStatus `json:"status"`
	PaidAt       *time.Time        `json:"paid_at,omitempty"`
	ShippedAt    *time.Time        `json:"shipped_at,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

func toAdminVO(row dao.OrderAdminRow) OrderAdminVO {
	return OrderAdminVO{
		ID:           row.ID,
		AuctionID:    row.AuctionID,
		ProductID:    row.ProductID,
		ProductName:  row.ProductName,
		ProductImage: firstProductImage(row.ProductImagesJSON),
		WinnerID:     row.WinnerID,
		UserID:       row.WinnerID,
		FinalPrice:   row.FinalPrice,
		Status:       row.Status,
		PaidAt:       row.PaidAt,
		ShippedAt:    row.ShippedAt,
		CompletedAt:  row.CompletedAt,
		CreatedAt:    row.CreatedAt,
	}
}

// firstProductImage 解析 products.images JSON 数组的首图，空/非法 JSON 一律返回 ""。
func firstProductImage(raw string) string {
	if raw == "" {
		return ""
	}
	var imgs []string
	if err := json.Unmarshal([]byte(raw), &imgs); err != nil {
		return ""
	}
	if len(imgs) == 0 {
		return ""
	}
	return imgs[0]
}

// ListAdminOrders admin 端订单列表，不按 winner_id 强过滤。
//   - 入参 status 与 userID 均为可选过滤条件；userID 等价于 winner_id 过滤（admin 想查某用户）；
//   - adminDAO 未注入时返回错误，避免 admin 接口静默降级成空列表。
func (s *OrderService) ListAdminOrders(ctx context.Context, status *model.OrderStatus, userID *int64, page, pageSize int) ([]OrderAdminVO, int64, error) {
	if s.adminDAO == nil {
		return nil, 0, ErrAdminDAOMissing
	}
	rows, total, err := s.adminDAO.ListAdminOrders(ctx, status, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	vos := make([]OrderAdminVO, 0, len(rows))
	for _, r := range rows {
		vos = append(vos, toAdminVO(r))
	}
	return vos, total, nil
}

// GetAdminOrder admin 端订单详情。
func (s *OrderService) GetAdminOrder(ctx context.Context, id int64) (*OrderAdminVO, error) {
	if s.adminDAO == nil {
		return nil, ErrAdminDAOMissing
	}
	row, err := s.adminDAO.GetAdminOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	vo := toAdminVO(*row)
	return &vo, nil
}
