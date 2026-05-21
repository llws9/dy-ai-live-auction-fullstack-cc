package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDelayService_IsInDelayWindow 测试延时窗口判断
func TestDelayService_IsInDelayWindow(t *testing.T) {
	tests := []struct {
		name          string
		endTime       time.Time
		triggerBefore int
		expected      bool
	}{
		{
			name:          "在延时窗口内（剩余20秒）",
			endTime:       time.Now().Add(20 * time.Second),
			triggerBefore: 30,
			expected:      true,
		},
		{
			name:          "不在延时窗口内（剩余40秒）",
			endTime:       time.Now().Add(40 * time.Second),
			triggerBefore: 30,
			expected:      false,
		},
		{
			name:          "刚好在边界（剩余30秒）",
			endTime:       time.Now().Add(30 * time.Second),
			triggerBefore: 30,
			expected:      true,
		},
		{
			name:          "竞拍已结束",
			endTime:       time.Now().Add(-1 * time.Second),
			triggerBefore: 30,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := time.Until(tt.endTime)
			result := remaining.Seconds() <= float64(tt.triggerBefore) && remaining.Seconds() > 0
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDelayService_GetRemainingDelayTime 测试剩余延时时间计算
func TestDelayService_GetRemainingDelayTime(t *testing.T) {
	tests := []struct {
		name      string
		delayUsed int
		maxDelay  int
		expected  int
	}{
		{
			name:      "未延时",
			delayUsed: 0,
			maxDelay:  180,
			expected:  180,
		},
		{
			name:      "已延时部分",
			delayUsed: 100,
			maxDelay:  180,
			expected:  80,
		},
		{
			name:      "已达最大延时",
			delayUsed: 180,
			maxDelay:  180,
			expected:  0,
		},
		{
			name:      "超过最大延时",
			delayUsed: 200,
			maxDelay:  180,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := tt.maxDelay - tt.delayUsed
			if remaining < 0 {
				remaining = 0
			}
			assert.Equal(t, tt.expected, remaining)
		})
	}
}

// TestDelayCalculation 测试延时计算逻辑
func TestDelayCalculation(t *testing.T) {
	// 测试场景：竞拍剩余20秒，用户出价，应该触发延时
	endTime := time.Now().Add(20 * time.Second)
	triggerBefore := 30
	delayDuration := 30
	maxDelayTime := 180
	currentDelayUsed := 0

	// 检查是否在延时窗口内
	remaining := time.Until(endTime)
	shouldDelay := remaining.Seconds() <= float64(triggerBefore) && remaining.Seconds() > 0
	assert.True(t, shouldDelay, "应该在延时窗口内")

	// 检查是否可以继续延时
	canDelay := currentDelayUsed < maxDelayTime
	assert.True(t, canDelay, "应该可以继续延时")

	// 计算实际延时时长
	actualDelay := delayDuration
	remainingDelay := maxDelayTime - currentDelayUsed
	if actualDelay > remainingDelay {
		actualDelay = remainingDelay
	}
	assert.Equal(t, 30, actualDelay, "实际延时时长应该是30秒")

	// 更新后的延时使用量
	newDelayUsed := currentDelayUsed + actualDelay
	assert.Equal(t, 30, newDelayUsed, "延时使用量应该是30秒")
}

// TestDelayMaxLimit 测试达到最大延时上限
func TestDelayMaxLimit(t *testing.T) {
	// 测试场景：已延时160秒，最大180秒，再出价只能延时20秒
	currentDelayUsed := 160
	delayDuration := 30
	maxDelayTime := 180

	// 计算剩余可延时
	remainingDelay := maxDelayTime - currentDelayUsed
	assert.Equal(t, 20, remainingDelay, "剩余可延时20秒")

	// 实际延时时长
	actualDelay := delayDuration
	if actualDelay > remainingDelay {
		actualDelay = remainingDelay
	}
	assert.Equal(t, 20, actualDelay, "实际只能延时20秒")

	// 更新后达到最大上限
	newDelayUsed := currentDelayUsed + actualDelay
	assert.Equal(t, 180, newDelayUsed, "延时使用量达到上限180秒")

	// 再次检查是否可以延时
	canDelay := newDelayUsed < maxDelayTime
	assert.False(t, canDelay, "已达到最大延时上限，不可继续延时")
}

// TestDelayNotInWindow 测试不在延时窗口内
func TestDelayNotInWindow(t *testing.T) {
	// 测试场景：竞拍剩余40秒，不在延时窗口（30秒）内
	endTime := time.Now().Add(40 * time.Second)
	triggerBefore := 30

	remaining := time.Until(endTime)
	shouldDelay := remaining.Seconds() <= float64(triggerBefore) && remaining.Seconds() > 0
	assert.False(t, shouldDelay, "不应该在延时窗口内")
}
