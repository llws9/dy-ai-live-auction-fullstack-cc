package websocket

import (
	"strings"
	"unicode/utf8"
)

// ChatFilter 弹幕内容校验器
type ChatFilter struct {
	maxLen  int
	blocked []string // 已转小写
}

// NewChatFilter 创建弹幕过滤器
func NewChatFilter(maxLen int, blockedWords []string) *ChatFilter {
	lowered := make([]string, len(blockedWords))
	for i, w := range blockedWords {
		lowered[i] = strings.ToLower(w)
	}
	return &ChatFilter{
		maxLen:  maxLen,
		blocked: lowered,
	}
}

// Validate 校验弹幕文本
// 返回 0 表示通过，非 0 为错误码（ChatErrCode*）
func (f *ChatFilter) Validate(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 || n > f.maxLen {
		return ChatErrCodeLengthExceeded
	}
	lower := strings.ToLower(text)
	for _, w := range f.blocked {
		if strings.Contains(lower, w) {
			return ChatErrCodeBlockedWord
		}
	}
	return 0
}
