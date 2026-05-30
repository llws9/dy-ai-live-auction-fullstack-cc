// Package chaos 进程内故障注入器（M6 MVP 版）。
//
// 设计：
//   - 单例 Broker：管理一组活跃 FaultProfile；
//   - HTTPMiddleware（http.RoundTripper 装饰器）：发起 HTTP 请求时根据当前 profile 注入
//     固定/随机延迟、随机失败（502/503）、强制断连；
//   - 完全在 test-service 进程内生效，不影响 auction-service / gateway-service。
//
// 与 toxiproxy 方案的取舍：
//   - 真实性较低（仅注入 client 侧）；
//   - 但部署零侵入，演示足够直观，符合用户 MVP 选择。
package chaos

import (
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// FaultType 故障类型
type FaultType string

const (
	FaultLatency      FaultType = "latency"      // 注入固定延迟
	FaultJitter       FaultType = "jitter"       // 注入抖动延迟（[0,Max)）
	FaultErrorRate    FaultType = "error_rate"   // 按概率返回 503
	FaultDisconnect   FaultType = "disconnect"   // 直接拒绝（模拟断连）
	FaultRedisFlap    FaultType = "redis_flap"   // 业务语义：模拟 Redis 闪断
	FaultMQPause      FaultType = "mq_pause"     // 业务语义：模拟 MQ 暂停消费
)

// Profile 一条故障注入配置
type Profile struct {
	ID         string        `json:"id"`
	Type       FaultType     `json:"type"`
	LatencyMs  int           `json:"latency_ms,omitempty"`
	JitterMs   int           `json:"jitter_ms,omitempty"`
	ErrorRate  float64       `json:"error_rate,omitempty"` // [0,1]
	StartedAt  time.Time     `json:"started_at"`
	Duration   time.Duration `json:"duration"`
	expiresAt  time.Time
}

// Broker 故障注入控制中心（进程内单例）
type Broker struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	rng      *rand.Rand
}

// global 单例
var global = &Broker{
	profiles: make(map[string]*Profile),
	rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
}

// Default 取全局 broker
func Default() *Broker { return global }

// Inject 注入一条 profile；duration<=0 表示永久（直到 Recover）
func (b *Broker) Inject(p Profile) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if p.ID == "" {
		p.ID = string(p.Type) + "-" + time.Now().Format("150405.000")
	}
	p.StartedAt = time.Now()
	if p.Duration > 0 {
		p.expiresAt = p.StartedAt.Add(p.Duration)
	}
	cp := p
	b.profiles[p.ID] = &cp
}

// Recover 移除一条 profile
func (b *Broker) Recover(id string) {
	b.mu.Lock()
	delete(b.profiles, id)
	b.mu.Unlock()
}

// RecoverAll 全部清空
func (b *Broker) RecoverAll() {
	b.mu.Lock()
	b.profiles = make(map[string]*Profile)
	b.mu.Unlock()
}

// List 当前活跃的 profiles 快照（已过期的会顺手清理）
func (b *Broker) List() []Profile {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	out := make([]Profile, 0, len(b.profiles))
	for id, p := range b.profiles {
		if !p.expiresAt.IsZero() && now.After(p.expiresAt) {
			delete(b.profiles, id)
			continue
		}
		out = append(out, *p)
	}
	return out
}

// applyTo 在一次 HTTP 调用前对每个活跃 profile 顺序应用故障
// 返回（应延迟时间，是否拒绝请求）
func (b *Broker) applyTo() (time.Duration, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	now := time.Now()
	var sleep time.Duration
	deny := false
	for id, p := range b.profiles {
		if !p.expiresAt.IsZero() && now.After(p.expiresAt) {
			// 过期的不应用，留给后台 List 清理
			_ = id
			continue
		}
		switch p.Type {
		case FaultLatency:
			sleep += time.Duration(p.LatencyMs) * time.Millisecond
		case FaultJitter:
			if p.JitterMs > 0 {
				sleep += time.Duration(b.rng.Intn(p.JitterMs)) * time.Millisecond
			}
		case FaultErrorRate:
			if b.rng.Float64() < p.ErrorRate {
				deny = true
			}
		case FaultDisconnect, FaultRedisFlap, FaultMQPause:
			// 直接拒绝（演示用：把 Redis/MQ 故障近似为业务调用失败）
			deny = true
		}
	}
	return sleep, deny
}

// ErrInjected 故障注入导致的人工失败
var ErrInjected = errors.New("chaos: injected failure")

// ChaosTransport http.RoundTripper 装饰器
type ChaosTransport struct {
	Underlying http.RoundTripper
}

// NewTransport 包装现有 transport
func NewTransport(under http.RoundTripper) *ChaosTransport {
	if under == nil {
		under = http.DefaultTransport
	}
	return &ChaosTransport{Underlying: under}
}

// RoundTrip 实现接口
func (t *ChaosTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	d, deny := global.applyTo()
	if d > 0 {
		select {
		case <-time.After(d):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	if deny {
		return nil, ErrInjected
	}
	return t.Underlying.RoundTrip(req)
}
