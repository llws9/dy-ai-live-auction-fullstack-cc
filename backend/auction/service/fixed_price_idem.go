package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const idemTTL = 10 * time.Minute

var uuidV4Re = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IdemStore 提供一口价购买的幂等存储。
// 存储的整数业务 ID 语义为「该用户在该 item 上的购买记录 ID（purchase ID）」。
type IdemStore struct {
	rdb *redis.Client
}

func NewIdemStore(rdb *redis.Client) *IdemStore {
	return &IdemStore{rdb: rdb}
}

// IsValidKey 校验幂等键是否为 UUID v4 格式。
func (s *IdemStore) IsValidKey(k string) bool {
	return uuidV4Re.MatchString(k)
}

func idemKey(userID, itemID int64, key string) string {
	return fmt.Sprintf("fp:idem:%d:%d:%s", userID, itemID, key)
}

// GetOrInit 命中返回 (storedID, true, nil)；未命中返回 (0, false, nil)。
// 最后一个 int64 参数当前未用。
func (s *IdemStore) GetOrInit(ctx context.Context, userID, itemID int64, key string, _ int64) (int64, bool, error) {
	val, err := s.rdb.Get(ctx, idemKey(userID, itemID, key)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

// Persist 写入购买记录 ID 并设置 10min TTL。
func (s *IdemStore) Persist(ctx context.Context, userID, itemID int64, key string, storedID int64) error {
	return s.rdb.Set(ctx, idemKey(userID, itemID, key), storedID, idemTTL).Err()
}
