package dao

import (
	"context"
	"testing"

	"product-service/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProductDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Category{}, &model.Product{}))
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	require.NoError(t, db.Exec("DELETE FROM categories").Error)
	return db
}

func TestProductDAOListAdminScopedMerchantOnlyOwnProducts(t *testing.T) {
	db := setupProductDAOTestDB(t)
	dao := NewProductDAO(db)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	categoryID := int64(12)
	require.NoError(t, db.Create(&model.Category{ID: categoryID, Name: "艺术品", Code: "ART", Status: model.CategoryStatusActive}).Error)
	require.NoError(t, dao.Create(ctx, &model.Product{Name: "A", OwnerID: &ownerA, CategoryID: &categoryID, Status: model.ProductStatusDraft}))
	require.NoError(t, dao.Create(ctx, &model.Product{Name: "B", OwnerID: &ownerB, Status: model.ProductStatusDraft}))

	items, total, err := dao.ListAdminScoped(ctx, &ownerA, nil, 1, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Name)
	require.Equal(t, "艺术品", items[0].CategoryName)
}

func TestProductDAOGetByIDAndOwnerIDRejectsOtherOwner(t *testing.T) {
	db := setupProductDAOTestDB(t)
	dao := NewProductDAO(db)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	product := &model.Product{Name: "A", OwnerID: &ownerA, Status: model.ProductStatusDraft}
	require.NoError(t, dao.Create(ctx, product))

	got, err := dao.GetByIDAndOwnerID(ctx, product.ID, ownerB)

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Nil(t, got)
}
