package dao

import (
	"context"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"product-service/model"
)

// OrderAdminRow 是 admin 列表/详情查询结果，内嵌 model.Order 并补齐 product 信息。
// product 信息通过本地 LEFT JOIN products 表得到，避免跨库 JOIN（users 表不在 product 服务库）。
type OrderAdminRow struct {
	model.Order
	ProductName       string `json:"product_name" gorm:"column:product_name"`
	ProductImagesJSON string `json:"-" gorm:"column:product_images"`
}

type OrderAdminSummary struct {
	PendingPaymentCount int64 `json:"pending_payment_count"`
	PaidCount           int64 `json:"paid_count"`
	ShippedCount        int64 `json:"shipped_count"`
	CompletedCount      int64 `json:"completed_count"`
}

// OrderAdminDAO 提供 admin 端订单查询能力，与用户视角 OrderDAO 区分，避免误用。
type OrderAdminDAO struct {
	db *gorm.DB
}

// NewOrderAdminDAO 创建 admin 订单 DAO。
func NewOrderAdminDAO(db *gorm.DB) *OrderAdminDAO {
	return &OrderAdminDAO{db: db}
}

// adminBaseQuery 构造统一的 SELECT + JOIN 语句。
// 注意：products.images 在 mysql 上是 JSON 列，sqlite 上是 TEXT；统一以字符串读出，由 service 层解析首图。
func (d *OrderAdminDAO) adminBaseQuery(ctx context.Context) *gorm.DB {
	return d.db.WithContext(ctx).
		Table("orders").
		Select("orders.*, p.name AS product_name, p.images AS product_images").
		Joins("LEFT JOIN products p ON p.id = orders.product_id")
}

func (d *OrderAdminDAO) adminCountQuery(ctx context.Context) *gorm.DB {
	return d.db.WithContext(ctx).
		Table("orders").
		Joins("LEFT JOIN products p ON p.id = orders.product_id")
}

func applyAdminOrderFilters(query *gorm.DB, status *model.OrderStatus, userID *int64, sellerID *int64, search string, includeStatus bool) *gorm.DB {
	if includeStatus && status != nil {
		query = query.Where("orders.status = ?", *status)
	}
	if userID != nil {
		query = query.Where("orders.winner_id = ?", *userID)
	}
	if sellerID != nil {
		query = query.Where("orders.seller_id = ?", *sellerID)
	}
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}
	like := "%" + search + "%"
	if id, err := strconv.ParseInt(search, 10, 64); err == nil && id > 0 {
		return query.Where("(orders.id = ? OR orders.winner_id = ? OR p.name LIKE ?)", id, id, like)
	}
	return query.Where("p.name LIKE ?", like)
}

// ListAdminOrders 返回全量订单（不按 winner_id 过滤），可选按 status / user_id 筛选。
// userID 在 admin 语义里等价于 winner_id 过滤——admin 想查某用户的订单。
func (d *OrderAdminDAO) ListAdminOrders(ctx context.Context, status *model.OrderStatus, userID *int64, page, pageSize int) ([]OrderAdminRow, int64, error) {
	rows, total, _, err := d.ListAdminOrdersScoped(ctx, status, userID, nil, "", page, pageSize)
	return rows, total, err
}

func (d *OrderAdminDAO) ListAdminOrdersScoped(ctx context.Context, status *model.OrderStatus, userID *int64, sellerID *int64, search string, page, pageSize int) ([]OrderAdminRow, int64, OrderAdminSummary, error) {
	var rows []OrderAdminRow
	var total int64
	var summary OrderAdminSummary

	countQ := applyAdminOrderFilters(d.adminCountQuery(ctx), status, userID, sellerID, search, true)
	listQ := applyAdminOrderFilters(d.adminBaseQuery(ctx), status, userID, sellerID, search, true)

	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, summary, err
	}
	if err := d.countAdminOrderSummary(ctx, userID, sellerID, search, &summary); err != nil {
		return nil, 0, summary, err
	}

	offset := (page - 1) * pageSize
	if err := listQ.
		Order("orders.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, summary, err
	}
	return rows, total, summary, nil
}

func (d *OrderAdminDAO) countAdminOrderSummary(ctx context.Context, userID *int64, sellerID *int64, search string, summary *OrderAdminSummary) error {
	countStatus := func(status model.OrderStatus) (int64, error) {
		var count int64
		err := applyAdminOrderFilters(d.adminCountQuery(ctx), &status, userID, sellerID, search, true).Count(&count).Error
		return count, err
	}

	var err error
	if summary.PendingPaymentCount, err = countStatus(model.OrderStatusPending); err != nil {
		return err
	}
	if summary.PaidCount, err = countStatus(model.OrderStatusPaid); err != nil {
		return err
	}
	if summary.ShippedCount, err = countStatus(model.OrderStatusShipped); err != nil {
		return err
	}
	if summary.CompletedCount, err = countStatus(model.OrderStatusCompleted); err != nil {
		return err
	}
	return nil
}

// GetAdminOrder 根据 id 返回单条 admin 视图订单。未命中返回 gorm.ErrRecordNotFound。
func (d *OrderAdminDAO) GetAdminOrder(ctx context.Context, id int64) (*OrderAdminRow, error) {
	return d.GetAdminOrderScoped(ctx, id, nil)
}

func (d *OrderAdminDAO) GetAdminOrderScoped(ctx context.Context, id int64, sellerID *int64) (*OrderAdminRow, error) {
	var row OrderAdminRow
	query := d.adminBaseQuery(ctx).Where("orders.id = ?", id)
	if sellerID != nil {
		query = query.Where("orders.seller_id = ?", *sellerID)
	}
	err := query.Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}
