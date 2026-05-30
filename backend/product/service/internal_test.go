package service

import (
	"context"
	"testing"

	"product-service/dao"
	"product-service/model"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InternalServiceTestSuite covers service methods used by other backends via
// the /internal/* endpoints (T2.1):
//   - ListProductsByCategory: backs `GET /internal/products?category_id=`
//   - GetProductsByIDs:       backs `POST /internal/products/batch`
type InternalServiceTestSuite struct {
	suite.Suite
	db      *gorm.DB
	service *ProductService
}

func (s *InternalServiceTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	s.NoError(err)
	s.NoError(db.AutoMigrate(&model.Product{}, &model.AuctionRule{}, &model.LiveStream{}))
	s.db = db
	s.service = NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
}

func (s *InternalServiceTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	sqlDB.Close()
}

func (s *InternalServiceTestSuite) SetupTest() {
	s.db.Exec("DELETE FROM products")
}

// helper: insert a product with the given category id (nil -> NULL)
func (s *InternalServiceTestSuite) seedProduct(name string, categoryID *int64) *model.Product {
	p := &model.Product{
		Name:       name,
		CategoryID: categoryID,
		Status:     model.ProductStatusPublished,
		Images:     model.JSONArray{name + "-img.jpg"},
	}
	s.NoError(s.db.Create(p).Error)
	return p
}

func ptrInt64(v int64) *int64 { return &v }

// --- ListProductsByCategory -------------------------------------------------

func (s *InternalServiceTestSuite) TestListProductsByCategory_FiltersByCategoryID() {
	ctx := context.Background()
	p1 := s.seedProduct("P1", ptrInt64(12))
	_ = s.seedProduct("P2", ptrInt64(99))
	p3 := s.seedProduct("P3", ptrInt64(12))

	items, total, err := s.service.ListProductsByCategory(ctx, 12, 1, 100)

	s.NoError(err)
	s.EqualValues(2, total)
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	s.ElementsMatch([]int64{p1.ID, p3.ID}, ids)
}

func (s *InternalServiceTestSuite) TestListProductsByCategory_EmptyWhenNoMatch() {
	ctx := context.Background()
	_ = s.seedProduct("P1", ptrInt64(12))

	items, total, err := s.service.ListProductsByCategory(ctx, 999, 1, 100)

	s.NoError(err)
	s.EqualValues(0, total)
	s.Empty(items)
}

func (s *InternalServiceTestSuite) TestListProductsByCategory_DefaultsAndPaging() {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		s.seedProduct("P", ptrInt64(7))
	}

	items, total, err := s.service.ListProductsByCategory(ctx, 7, 0, 0) // defaults
	s.NoError(err)
	s.EqualValues(5, total)
	s.Len(items, 5)

	items, total, err = s.service.ListProductsByCategory(ctx, 7, 1, 2)
	s.NoError(err)
	s.EqualValues(5, total)
	s.Len(items, 2)
}

// --- GetProductsByIDs -------------------------------------------------------

func (s *InternalServiceTestSuite) TestGetProductsByIDs_ReturnsMatching() {
	ctx := context.Background()
	p1 := s.seedProduct("P1", ptrInt64(1))
	p2 := s.seedProduct("P2", ptrInt64(2))
	_ = s.seedProduct("P3", ptrInt64(3))

	items, err := s.service.GetProductsByIDs(ctx, []int64{p1.ID, p2.ID, 99999})

	s.NoError(err)
	// Deleted/missing ids must be silently dropped (per spec §5.1.1).
	s.Len(items, 2)
	ids := []int64{items[0].ID, items[1].ID}
	s.ElementsMatch([]int64{p1.ID, p2.ID}, ids)
}

func (s *InternalServiceTestSuite) TestGetProductsByIDs_EmptyInputReturnsEmpty() {
	ctx := context.Background()
	items, err := s.service.GetProductsByIDs(ctx, []int64{})
	s.NoError(err)
	s.Empty(items)
}

func (s *InternalServiceTestSuite) TestGetProductsByIDs_NilInputReturnsEmpty() {
	ctx := context.Background()
	items, err := s.service.GetProductsByIDs(ctx, nil)
	s.NoError(err)
	s.Empty(items)
}

func (s *InternalServiceTestSuite) TestGetProductsByIDs_RejectsOversizedBatch() {
	ctx := context.Background()
	ids := make([]int64, 201)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	_, err := s.service.GetProductsByIDs(ctx, ids)
	s.Error(err)
}

func TestInternalServiceSuite(t *testing.T) {
	suite.Run(t, new(InternalServiceTestSuite))
}
