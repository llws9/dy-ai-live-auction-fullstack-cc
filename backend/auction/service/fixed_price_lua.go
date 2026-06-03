package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// StockResult 表示一口价库存抢占的结果码。
type StockResult int

const (
	StockResultSuccess       StockResult = 1
	StockResultSoldOut       StockResult = -1
	StockResultAlreadyBought StockResult = -2
	StockResultUninitialized StockResult = -3
)

// acquireScript 原子完成库存抢占。
// KEYS[1]=stock key, KEYS[2]=bought set key, ARGV[1]=userID
//   - stock key 不存在 → return -3（未初始化）
//   - userID 已在 bought set → return -2（重复购买）
//   - DECR stock，若 <0 则 INCR 回滚并 return -1（售罄）
//   - 否则 SADD userID 到 bought set，return 1（成功）
var acquireScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then
    return -3
end
if redis.call("SISMEMBER", KEYS[2], ARGV[1]) == 1 then
    return -2
end
local remaining = redis.call("DECR", KEYS[1])
if remaining < 0 then
    redis.call("INCR", KEYS[1])
    return -1
end
redis.call("SADD", KEYS[2], ARGV[1])
return 1
`)

// StockGuard 基于 Redis Lua 脚本实现一口价库存的原子抢占与补偿。
type StockGuard struct {
	rdb *redis.Client
}

// NewStockGuard 构造 StockGuard。
func NewStockGuard(rdb *redis.Client) *StockGuard {
	return &StockGuard{rdb: rdb}
}

func stockKey(itemID int64) string  { return fmt.Sprintf("fp:stock:%d", itemID) }
func boughtKey(itemID int64) string { return fmt.Sprintf("fp:bought:%d", itemID) }

// Init 初始化指定商品的库存为 total（无过期）。
func (g *StockGuard) Init(ctx context.Context, itemID int64, total int) error {
	return g.rdb.Set(ctx, stockKey(itemID), total, 0).Err()
}

// TryAcquire 尝试为 userID 抢占一件 itemID 库存，返回结果码。
func (g *StockGuard) TryAcquire(ctx context.Context, itemID, userID int64) (StockResult, error) {
	res, err := acquireScript.Run(ctx, g.rdb,
		[]string{stockKey(itemID), boughtKey(itemID)},
		userID,
	).Int64()
	if err != nil {
		return 0, err
	}
	return StockResult(res), nil
}

// Compensate 回补一件库存并移除 userID 的购买记录（用于下游失败补偿）。
func (g *StockGuard) Compensate(ctx context.Context, itemID, userID int64) error {
	pipe := g.rdb.TxPipeline()
	pipe.Incr(ctx, stockKey(itemID))
	pipe.SRem(ctx, boughtKey(itemID), userID)
	_, err := pipe.Exec(ctx)
	return err
}

// Cleanup 删除商品的库存与购买记录（下架/售罄后清理用）。
func (g *StockGuard) Cleanup(ctx context.Context, itemID int64) error {
	return g.rdb.Del(ctx, stockKey(itemID), boughtKey(itemID)).Err()
}

// Remaining 返回当前剩余库存。
func (g *StockGuard) Remaining(ctx context.Context, itemID int64) (int, error) {
	return g.rdb.Get(ctx, stockKey(itemID)).Int()
}
