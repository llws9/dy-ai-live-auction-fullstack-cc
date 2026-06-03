package service

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupTestRedis 启动一个进程内 miniredis 并返回连接它的 go-redis 客户端。
// miniredis 支持 EVAL/EVALSHA，可用于验证一口价库存 Lua 脚本的原子语义。
func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}
