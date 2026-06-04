package dao

import (
	"context"

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

// ListAdminOrders 返回全量订单（不按 winner_id 过滤），可选按 status / user_id 筛选。
// userID 在 admin 语义里等价于 winner_id 过滤——admin 想查某用户的订单。
func (d *OrderAdminDAO) ListAdminOrders(ctx context.Context, status *model.OrderStatus, userID *int64, page, pageSize int) ([]OrderAdminRow, int64, error) {
	return d.ListAdminOrdersScoped(ctx, status, userID, nil, page, pageSize)
}

func (d *OrderAdminDAO) ListAdminOrdersScoped(ctx context.Context, status *model.OrderStatus, userID *int64, sellerID *int64, page, pageSize int) ([]OrderAdminRow, int64, error) {
	var rows []OrderAdminRow
	var total int64

	countQ := d.db.WithContext(ctx).Model(&model.Order{})
	listQ := d.adminBaseQuery(ctx)
	if status != nil {
		countQ = countQ.Where("status = ?", *status)
		listQ = listQ.Where("orders.status = ?", *status)
	}
	if userID != nil {
		countQ = countQ.Where("winner_id = ?", *userID)
		listQ = listQ.Where("orders.winner_id = ?", *userID)
	}
	if sellerID != nil {
		countQ = countQ.Where("seller_id = ?", *sellerID)
		listQ = listQ.Where("orders.seller_id = ?", *sellerID)
	}

	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := listQ.
		Order("orders.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
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
