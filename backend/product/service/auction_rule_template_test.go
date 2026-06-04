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

func setupRuleTemplateServiceTest(t *testing.T) *AuctionRuleTemplateService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AuctionRuleTemplate{}))
	require.NoError(t, db.Exec("DELETE FROM auction_rule_templates").Error)
	return NewAuctionRuleTemplateService(dao.NewAuctionRuleTemplateDAO(db))
}

func TestAuctionRuleTemplateServiceRejectsAmountWithMoreThanTwoDecimals(t *testing.T) {
	svc := setupRuleTemplateServiceTest(t)

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
	svc := setupRuleTemplateServiceTest(t)
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
