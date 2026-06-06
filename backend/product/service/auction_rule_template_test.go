package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
)

func setupRuleTemplateServiceTest(t *testing.T) (*AuctionRuleTemplateService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AuctionRuleTemplate{}, &model.Product{}, &model.AuctionRule{}))
	require.NoError(t, db.Exec("DELETE FROM auction_rule_templates").Error)
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	require.NoError(t, db.Exec("DELETE FROM auction_rules").Error)
	return NewAuctionRuleTemplateService(dao.NewAuctionRuleTemplateDAO(db), dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db)), db
}

func TestAuctionRuleTemplateServiceRejectsAmountWithMoreThanTwoDecimals(t *testing.T) {
	svc, _ := setupRuleTemplateServiceTest(t)

	_, err := svc.Create(context.Background(), 1001, CreateAuctionRuleTemplateRequest{
		Name:       "默认模板",
		StartPrice: "10.001",
		Increment:  "1.00",
		Duration:   60,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "金额最多支持两位小数")
}

func TestAuctionRuleTemplateServiceListOnlyOwnerTemplates(t *testing.T) {
	svc, _ := setupRuleTemplateServiceTest(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, 1001, CreateAuctionRuleTemplateRequest{Name: "A", StartPrice: "10.00", Increment: "1.00", Duration: 60})
	require.NoError(t, err)
	_, err = svc.Create(ctx, 1002, CreateAuctionRuleTemplateRequest{Name: "B", StartPrice: "20.00", Increment: "2.00", Duration: 60})
	require.NoError(t, err)

	items, total, err := svc.List(ctx, 1001, 1, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Name)
	require.Equal(t, "10.00", items[0].StartPrice)
}

func TestAuctionRuleTemplateServiceApplyToProductWritesAuctionRule(t *testing.T) {
	svc, db := setupRuleTemplateServiceTest(t)
	ctx := context.Background()
	ownerID := int64(1001)
	otherOwnerID := int64(1002)
	product := model.Product{OwnerID: &ownerID, Name: "青花瓷瓶", Status: model.ProductStatusDraft}
	require.NoError(t, db.Create(&product).Error)
	otherProduct := model.Product{OwnerID: &otherOwnerID, Name: "他人商品", Status: model.ProductStatusDraft}
	require.NoError(t, db.Create(&otherProduct).Error)
	template, err := svc.Create(ctx, ownerID, CreateAuctionRuleTemplateRequest{
		Name:               "默认模板",
		StartPrice:         "100.00",
		Increment:          "10.00",
		CapPrice:           "1000.00",
		Duration:           3600,
		DelayDuration:      30,
		MaxDelayTime:       180,
		TriggerDelayBefore: 30,
	})
	require.NoError(t, err)

	rule, err := svc.ApplyToProduct(ctx, ownerID, product.ID, template.ID)

	require.NoError(t, err)
	require.Equal(t, product.ID, rule.ProductID)
	require.Equal(t, 100.0, rule.StartPrice)
	require.Equal(t, 10.0, rule.Increment)
	require.NotNil(t, rule.CapPrice)
	require.Equal(t, 1000.0, *rule.CapPrice)
	require.Equal(t, 3600, rule.Duration)

	_, err = svc.ApplyToProduct(ctx, ownerID, otherProduct.ID, template.ID)
	require.Error(t, err)
}

func TestAuctionRuleTemplateServiceApplyToProductClearsNullableAndZeroFields(t *testing.T) {
	svc, db := setupRuleTemplateServiceTest(t)
	ctx := context.Background()
	ownerID := int64(1001)
	product := model.Product{OwnerID: &ownerID, Name: "青花瓷瓶", Status: model.ProductStatusDraft}
	require.NoError(t, db.Create(&product).Error)
	oldCapPrice := 1000.0
	require.NoError(t, db.Create(&model.AuctionRule{
		ProductID:          product.ID,
		StartPrice:         100,
		Increment:          10,
		CapPrice:           &oldCapPrice,
		Duration:           3600,
		DelayDuration:      30,
		MaxDelayTime:       180,
		TriggerDelayBefore: 30,
	}).Error)
	template, err := svc.Create(ctx, ownerID, CreateAuctionRuleTemplateRequest{
		Name:               "无封顶零元起拍",
		StartPrice:         "0.00",
		Increment:          "1.00",
		CapPrice:           "",
		Duration:           120,
		DelayDuration:      15,
		MaxDelayTime:       45,
		TriggerDelayBefore: 10,
	})
	require.NoError(t, err)

	rule, err := svc.ApplyToProduct(ctx, ownerID, product.ID, template.ID)

	require.NoError(t, err)
	require.Zero(t, rule.StartPrice)
	require.Nil(t, rule.CapPrice)
	require.Equal(t, 120, rule.Duration)

	var persisted model.AuctionRule
	require.NoError(t, db.Where("product_id = ?", product.ID).First(&persisted).Error)
	require.Zero(t, persisted.StartPrice)
	require.Nil(t, persisted.CapPrice)
	require.Equal(t, 120, persisted.Duration)
}
