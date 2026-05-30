package handler

import (
	"context"
	"errors"
	"regexp"

	"auction-service/model"
)

// 业务错误（用于 HTTP shell 区分 4xx 状态码）。
var (
	ErrAddressInvalid       = errors.New("address invalid input")
	ErrAddressLimitExceeded = errors.New("address limit exceeded")
	ErrAddressNotFound      = errors.New("address not found")
)

const maxAddressesPerUser = 20

var phoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// AddressView / AddressMutation 复用 model 包定义（dao 共用，避免 import cycle）。
type AddressView = model.AddressView
type AddressMutation = model.AddressMutation

// AddressInput 接收前端 JSON Body（spec A / F-A3 §字段约束）。
type AddressInput struct {
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	Detail        string `json:"detail"`
	IsDefault     bool   `json:"is_default"`
}

// AddressListResponse 列表响应 data 字段。
type AddressListResponse struct {
	Items []AddressView `json:"items"`
	Total int64         `json:"total"`
}

// AddressStore 抽象 DAO 操作，便于单测。
//
// 所有写入操作以 (id, userID) 为复合定位键 — 越权访问表现为 hit=false（spec A / F-A3 错误码 §404）。
// SetDefault 内部以事务保证同 user_id 的 is_default 互斥。
type AddressStore interface {
	List(ctx context.Context, userID int64) ([]AddressView, error)
	Count(ctx context.Context, userID int64) (int64, error)
	Get(ctx context.Context, id, userID int64) (*AddressView, bool, error)
	Create(ctx context.Context, m AddressMutation) (*AddressView, error)
	Update(ctx context.Context, id, userID int64, m AddressMutation) (hit bool, err error)
	Delete(ctx context.Context, id, userID int64) (hit bool, err error)
	SetDefault(ctx context.Context, id, userID int64) (hit bool, err error)
}

// ValidateAddressInput 锁定字段约束（spec A / F-A3 §字段约束）。
func ValidateAddressInput(in AddressInput) error {
	if l := len(in.RecipientName); l < 1 || l > 32 {
		return errors.New("recipient_name length 1-32 required")
	}
	if !phoneRegexp.MatchString(in.Phone) {
		return errors.New("phone must be valid 11-digit CN mobile")
	}
	if l := len(in.Province); l < 1 || l > 32 {
		return errors.New("province length 1-32 required")
	}
	if l := len(in.City); l < 1 || l > 32 {
		return errors.New("city length 1-32 required")
	}
	if l := len(in.District); l < 1 || l > 32 {
		return errors.New("district length 1-32 required")
	}
	if l := len(in.Detail); l < 1 || l > 128 {
		return errors.New("detail length 1-128 required")
	}
	return nil
}

// BuildListAddresses 编排 GET /users/me/addresses。
func BuildListAddresses(ctx context.Context, store AddressStore, userID int64) (*AddressListResponse, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	items, err := store.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []AddressView{}
	}
	return &AddressListResponse{Items: items, Total: int64(len(items))}, nil
}

// BuildCreateAddress 编排 POST /users/me/addresses。
//   - 校验失败 → ErrAddressInvalid
//   - 当前 >=20 → ErrAddressLimitExceeded
//   - 当 count==0 时强制 is_default=true（首条强默认，spec §字段约束）
func BuildCreateAddress(ctx context.Context, store AddressStore, userID int64, in AddressInput) (*AddressView, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	if err := ValidateAddressInput(in); err != nil {
		return nil, errors.Join(ErrAddressInvalid, err)
	}
	count, err := store.Count(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= maxAddressesPerUser {
		return nil, ErrAddressLimitExceeded
	}
	if count == 0 {
		in.IsDefault = true
	}
	return store.Create(ctx, AddressMutation{
		UserID:        userID,
		RecipientName: in.RecipientName,
		Phone:         in.Phone,
		Province:      in.Province,
		City:          in.City,
		District:      in.District,
		Detail:        in.Detail,
		IsDefault:     in.IsDefault,
	})
}

// BuildUpdateAddress 编排 PUT /users/me/addresses/:id。
//   - 校验失败 → ErrAddressInvalid（先于 store 调用）
//   - store hit=false → ErrAddressNotFound（覆盖越权）
func BuildUpdateAddress(ctx context.Context, store AddressStore, userID, id int64, in AddressInput) error {
	if userID <= 0 || id <= 0 {
		return errors.New("invalid id or user_id")
	}
	if err := ValidateAddressInput(in); err != nil {
		return errors.Join(ErrAddressInvalid, err)
	}
	hit, err := store.Update(ctx, id, userID, AddressMutation{
		ID:            id,
		UserID:        userID,
		RecipientName: in.RecipientName,
		Phone:         in.Phone,
		Province:      in.Province,
		City:          in.City,
		District:      in.District,
		Detail:        in.Detail,
		IsDefault:     in.IsDefault,
	})
	if err != nil {
		return err
	}
	if !hit {
		return ErrAddressNotFound
	}
	return nil
}

// BuildDeleteAddress 编排 DELETE /users/me/addresses/:id。
func BuildDeleteAddress(ctx context.Context, store AddressStore, userID, id int64) error {
	if userID <= 0 || id <= 0 {
		return errors.New("invalid id or user_id")
	}
	hit, err := store.Delete(ctx, id, userID)
	if err != nil {
		return err
	}
	if !hit {
		return ErrAddressNotFound
	}
	return nil
}

// BuildSetDefaultAddress 编排 POST /users/me/addresses/:id/default。
//   - store.SetDefault 内部事务保证同 user_id 的 is_default 互斥。
func BuildSetDefaultAddress(ctx context.Context, store AddressStore, userID, id int64) error {
	if userID <= 0 || id <= 0 {
		return errors.New("invalid id or user_id")
	}
	hit, err := store.SetDefault(ctx, id, userID)
	if err != nil {
		return err
	}
	if !hit {
		return ErrAddressNotFound
	}
	return nil
}
