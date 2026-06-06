package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

func setupRuleTemplateHandlerTest(t *testing.T) *AuctionRuleTemplateHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AuctionRuleTemplate{}, &model.Product{}, &model.AuctionRule{}))
	require.NoError(t, db.Exec("DELETE FROM auction_rule_templates").Error)
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	require.NoError(t, db.Exec("DELETE FROM auction_rules").Error)
	svc := service.NewAuctionRuleTemplateService(dao.NewAuctionRuleTemplateDAO(db), dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db))
	return NewAuctionRuleTemplateHandler(svc)
}

func mustDecimal(t *testing.T, raw string) decimal.Decimal {
	t.Helper()
	v, err := decimal.NewFromString(raw)
	require.NoError(t, err)
	return v
}

func TestAuctionRuleTemplateHandlerCreateRejectsAdmin(t *testing.T) {
	h := setupRuleTemplateHandlerTest(t)
	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/auction-rule-templates")
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"name":"平台模板","start_price":"10.00","increment":"1.00","duration":60}`)

	h.Create(context.Background(), c)

	require.Equal(t, http.StatusForbidden, c.Response.StatusCode())
}

func TestAuctionRuleTemplateHandlerCreateMerchant(t *testing.T) {
	h := setupRuleTemplateHandlerTest(t)
	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/auction-rule-templates")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"name":"默认模板","start_price":"10.00","increment":"1.00","duration":60}`)

	h.Create(context.Background(), c)

	require.Equal(t, http.StatusCreated, c.Response.StatusCode())
	require.Contains(t, string(c.Response.Body()), `"start_price":"10.00"`)
}

func TestAuctionRuleTemplateHandlerListMerchantOnlyOwnTemplates(t *testing.T) {
	h := setupRuleTemplateHandlerTest(t)
	for _, req := range []struct {
		owner string
		body  string
	}{
		{"1001", `{"name":"A","start_price":"10.00","increment":"1.00","duration":60}`},
		{"1002", `{"name":"B","start_price":"20.00","increment":"2.00","duration":60}`},
	} {
		c := app.NewContext(0)
		c.Request.SetRequestURI("/api/v1/admin/auction-rule-templates")
		c.Request.Header.Set("X-User-ID", req.owner)
		c.Request.Header.Set("X-User-Role", "merchant")
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.SetBodyString(req.body)
		h.Create(context.Background(), c)
		require.Equal(t, http.StatusCreated, c.Response.StatusCode())
	}

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/auction-rule-templates")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	h.List(context.Background(), c)

	body := string(c.Response.Body())
	require.Equal(t, http.StatusOK, c.Response.StatusCode())
	require.Contains(t, body, `"name":"A"`)
	require.False(t, strings.Contains(body, `"name":"B"`))
}

func TestAuctionRuleTemplateHandlerApplyToProduct(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AuctionRuleTemplate{}, &model.Product{}, &model.AuctionRule{}))
	ownerID := int64(1001)
	product := model.Product{OwnerID: &ownerID, Name: "青花瓷瓶", Status: model.ProductStatusDraft}
	require.NoError(t, db.Create(&product).Error)
	template := model.AuctionRuleTemplate{
		OwnerID:            ownerID,
		Name:               "默认模板",
		StartPrice:         mustDecimal(t, "100.00"),
		Increment:          mustDecimal(t, "10.00"),
		Duration:           3600,
		DelayDuration:      30,
		MaxDelayTime:       180,
		TriggerDelayBefore: 30,
	}
	require.NoError(t, db.Create(&template).Error)
	svc := service.NewAuctionRuleTemplateService(dao.NewAuctionRuleTemplateDAO(db), dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db))
	h := NewAuctionRuleTemplateHandler(svc)
	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/products/1/apply-rule-template")
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(product.ID, 10)})
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"template_id":` + strconv.FormatInt(template.ID, 10) + `}`)

	h.ApplyToProduct(context.Background(), c)

	require.Equal(t, http.StatusOK, c.Response.StatusCode())
	require.Contains(t, string(c.Response.Body()), `"product_id":`)
	require.Contains(t, string(c.Response.Body()), `"increment":10`)
}
