package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuctionService_CreateAuctionValidatesProductLifecycle(t *testing.T) {
	merchantID := int64(1001)

	tests := []struct {
		name        string
		req         *CreateAuctionRequest
		seed        func(t *testing.T, db *gorm.DB)
		wantErr     string
		wantCreated bool
	}{
		{
			name: "merchant can create auction for own published product with bound rule",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77,
			},
			wantCreated: true,
		},
		{
			name:    "nil request rejected",
			req:     nil,
			wantErr: "创建竞拍请求不能为空",
		},
		{
			name: "nil creator rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: nil, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77,
			},
			wantErr: "创建者ID非法",
		},
		{
			name: "draft product rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 0, RuleBound: true, LiveStreamID: 77,
			},
			wantErr: "商品未进入竞拍池",
		},
		{
			name: "other merchant product rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 2002, ProductStatus: 1, RuleBound: true, LiveStreamID: 77,
			},
			wantErr: "商品不存在或不属于当前商家",
		},
		{
			name: "missing rule rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: false, LiveStreamID: 77,
			},
			wantErr: "规则模板不存在或不属于当前商家",
		},
		{
			name: "missing live stream rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 0,
			},
			wantErr: "直播间不可用",
		},
		{
			name: "active auction rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77,
			},
			seed: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&model.Auction{
					ProductID: 11,
					Status:    model.AuctionStatusOngoing,
					StartTime: time.Now().Add(-time.Minute),
					EndTime:   time.Now().Add(time.Hour),
				}).Error)
			},
			wantErr: "该商品已有待开始或进行中的竞拍场次",
		},
		{
			name: "latest sold auction rejected",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77,
			},
			seed: func(t *testing.T, db *gorm.DB) {
				winnerID := int64(3003)
				require.NoError(t, db.Create(&model.Auction{
					ProductID: 11,
					Status:    model.AuctionStatusEnded,
					WinnerID:  &winnerID,
					StartTime: time.Now().Add(-2 * time.Hour),
					EndTime:   time.Now().Add(-time.Hour),
				}).Error)
			},
			wantErr: "已成交商品不能再次创建竞拍",
		},
		{
			name: "latest unsold auction allows retry",
			req: &CreateAuctionRequest{
				ProductID: 11, CreatorID: &merchantID, Duration: 3600,
				ProductOwnerID: 1001, ProductStatus: 1, RuleBound: true, LiveStreamID: 77,
			},
			seed: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&model.Auction{
					ProductID: 11,
					Status:    model.AuctionStatusEnded,
					WinnerID:  nil,
					StartTime: time.Now().Add(-2 * time.Hour),
					EndTime:   time.Now().Add(-time.Hour),
				}).Error)
			},
			wantCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newAuctionCreateTestDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}

			svc := NewAuctionService(dao.NewAuctionDAO(db))
			got, err := svc.CreateAuction(context.Background(), tt.req)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, int64(11), got.ProductID)
			assert.Equal(t, &merchantID, got.CreatorID)
			assert.Equal(t, model.AuctionStatusPending, got.Status)
			assert.Equal(t, int64(77), *got.LiveStreamID)
			assert.True(t, got.CurrentPrice.Equal(decimal.Zero))
			assert.False(t, got.StartTime.IsZero())
			assert.WithinDuration(t, time.Now(), got.StartTime, 2*time.Second)
			assert.WithinDuration(t, got.StartTime.Add(time.Hour), got.EndTime, time.Second)
		})
	}
}

func TestAuctionService_CreateAuctionUsesRequestedFutureStartTime(t *testing.T) {
	db := newAuctionCreateTestDB(t)
	merchantID := int64(1001)
	requestedStart := time.Now().Add(30 * time.Minute).Truncate(time.Second)

	svc := NewAuctionService(dao.NewAuctionDAO(db))
	got, err := svc.CreateAuction(context.Background(), &CreateAuctionRequest{
		ProductID:      11,
		CreatorID:      &merchantID,
		Duration:       3600,
		ProductOwnerID: merchantID,
		ProductStatus:  1,
		RuleBound:      true,
		LiveStreamID:   77,
		StartTime:      requestedStart,
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.AuctionStatusPending, got.Status)
	assert.True(t, got.StartTime.Equal(requestedStart), "start_time should preserve the requested schedule")
	assert.True(t, got.EndTime.Equal(requestedStart.Add(time.Hour)), "end_time should be derived from scheduled start_time + duration")
}

func TestIsActiveAuctionUniqueConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "mysql duplicate on active product key",
			err:  errors.New("Error 1062 (23000): Duplicate entry '11' for key 'uk_active_product'"),
			want: true,
		},
		{
			name: "mysql duplicate on other key is not active auction conflict",
			err:  errors.New("Error 1062 (23000): Duplicate entry 'A-001' for key 'uk_other_unique_key'"),
			want: false,
		},
		{
			name: "mysql duplicate on active live stream key is not product conflict",
			err:  errors.New("Error 1062 (23000): Duplicate entry '77' for key 'uk_active_live_stream'"),
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("database connection lost"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isActiveAuctionUniqueConflict(tt.err))
		})
	}
}

func TestIsActiveLiveStreamUniqueConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "mysql duplicate on active live stream key",
			err:  errors.New("Error 1062 (23000): Duplicate entry '77' for key 'uk_active_live_stream'"),
			want: true,
		},
		{
			name: "mysql duplicate on active product key is not live stream conflict",
			err:  errors.New("Error 1062 (23000): Duplicate entry '11' for key 'uk_active_product'"),
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("database connection lost"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isActiveLiveStreamUniqueConflict(tt.err))
		})
	}
}

func newAuctionCreateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return db
}
