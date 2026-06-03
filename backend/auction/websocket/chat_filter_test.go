package websocket

import "testing"

func TestChatFilter_LengthBoundary(t *testing.T) {
	f := NewChatFilter(50, []string{})

	cases := []struct {
		name    string
		text    string
		wantErr int
	}{
		{"empty", "", ChatErrCodeLengthExceeded},
		{"49 chars", repeatRune('a', 49), 0},
		{"50 chars", repeatRune('a', 50), 0},
		{"51 chars", repeatRune('a', 51), ChatErrCodeLengthExceeded},
		{"50 chinese", repeatRune('中', 50), 0},
		{"51 chinese", repeatRune('中', 51), ChatErrCodeLengthExceeded},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotCode := f.Validate(c.text)
			if gotCode != c.wantErr {
				t.Errorf("Validate(%q) code = %d, want %d", c.name, gotCode, c.wantErr)
			}
		})
	}
}

func TestChatFilter_BlockedWord(t *testing.T) {
	f := NewChatFilter(50, []string{"微信", "vx"})

	cases := []struct {
		text    string
		wantErr int
	}{
		{"加我微信", ChatErrCodeBlockedWord},
		{"加vx一下", ChatErrCodeBlockedWord},
		{"VX 大写", ChatErrCodeBlockedWord}, // 大小写不敏感
		{"正常聊天内容", 0},
	}

	for _, c := range cases {
		gotCode := f.Validate(c.text)
		if gotCode != c.wantErr {
			t.Errorf("Validate(%q) code = %d, want %d", c.text, gotCode, c.wantErr)
		}
	}
}

func repeatRune(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
