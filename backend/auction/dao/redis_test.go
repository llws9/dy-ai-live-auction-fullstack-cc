package dao

import "testing"

func TestRedisOptions_DefaultsBoundActiveConnections(t *testing.T) {
	opts := redisOptions(RedisConfig{Addr: "localhost:6379"})

	if opts.PoolSize != 128 {
		t.Fatalf("PoolSize: want 128, got %d", opts.PoolSize)
	}
	if opts.MaxActiveConns != opts.PoolSize {
		t.Fatalf("MaxActiveConns should default to PoolSize, got %d", opts.MaxActiveConns)
	}
	if opts.MaxConcurrentDials != 16 {
		t.Fatalf("MaxConcurrentDials: want 16, got %d", opts.MaxConcurrentDials)
	}
}

func TestRedisOptions_NormalizesPoolBounds(t *testing.T) {
	opts := redisOptions(RedisConfig{
		Addr:           "localhost:6379",
		PoolSize:       8,
		MinIdleConns:   32,
		MaxIdleConns:   64,
		MaxActiveConns: 16,
	})

	if opts.MinIdleConns != 8 {
		t.Fatalf("MinIdleConns should be capped by PoolSize, got %d", opts.MinIdleConns)
	}
	if opts.MaxIdleConns != 16 {
		t.Fatalf("MaxIdleConns should be capped by MaxActiveConns, got %d", opts.MaxIdleConns)
	}
}
