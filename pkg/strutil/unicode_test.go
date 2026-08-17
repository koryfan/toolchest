package strutil

import (
	"strings"
	"testing"
)

func TestChineseToUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"纯中文", `你好`, `\u4f60\u597d`},
		{"中英混合", `hello 世界`, `hello \u4e16\u754c`},
		{"纯 ASCII", `hello`, `hello`},
		{"带标点", `价格￥100`, `\u4ef7\u683c\uffe5100`},
		{"空字符串", ``, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChineseToUnicode(tt.input); got != tt.want {
				t.Errorf("ChineseToUnicode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnicodeToChinese(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"纯 Unicode 转义", `\u4f60\u597d`, `你好`},
		{"混合文本", `hello \u4e16\u754c`, `hello 世界`},
		{"无转义原样返回", `hello`, `hello`},
		{"单个转义", `\u4e2d`, `中`},
		{"非法转义原样保留", `\uzzzz`, `\uzzzz`},
		{"空字符串", ``, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnicodeToChinese(tt.input); got != tt.want {
				t.Errorf("UnicodeToChinese() = %q, want %q", got, tt.want)
			}
		})
	}
}

// 互转一致性：中文 → Unicode → 中文 应该还原
func TestChineseUnicodeRoundTrip(t *testing.T) {
	original := `你好，世界！hello 123`
	encoded := ChineseToUnicode(original)
	if strings.Contains(encoded, "你") {
		t.Error("编码后不应包含原始中文字符")
	}
	if decoded := UnicodeToChinese(encoded); decoded != original {
		t.Errorf("往返不一致: got %q, want %q", decoded, original)
	}
}
