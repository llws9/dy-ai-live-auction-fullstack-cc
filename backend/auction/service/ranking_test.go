package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRankingThrottle(t *testing.T) {
	throttle := NewRankingThrottle()

	t.Run("should allow first request", func(t *testing.T) {
		result := throttle.ShouldSend(1)
		assert.True(t, result)
	})

	t.Run("should throttle second request within 200ms", func(t *testing.T) {
		throttle.ShouldSend(1) // 第一次
		result := throttle.ShouldSend(1) // 立即第二次
		assert.False(t, result) // 应该被节流
	})

	t.Run("should allow request after 200ms", func(t *testing.T) {
		throttle.ShouldSend(2) // 第一次
		time.Sleep(210 * time.Millisecond) // 等待200ms
		result := throttle.ShouldSend(2) // 第三次
		assert.True(t, result)
	})
}
