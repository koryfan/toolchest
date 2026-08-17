package strutil

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ChineseToUnicode 把字符串里的非 ASCII 字符（含中文）转成 \uXXXX 形式
// ASCII 字符原样保留
func ChineseToUnicode(input string) string {
	var b strings.Builder
	for _, r := range input {
		if r < utf8.RuneSelf {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "\\u%04x", r)
		}
	}
	return b.String()
}

// UnicodeToChinese 把字符串里的 \uXXXX 转回对应字符
// 遇到非法的 \u 序列（如 \uzzzz）原样保留那 6 个字符
func UnicodeToChinese(input string) string {
	var b strings.Builder
	for i := 0; i < len(input); {
		// 检测 \uXXXX 模式（\u 后跟 4 位十六进制）
		if i+6 <= len(input) && input[i] == '\\' && input[i+1] == 'u' {
			hex := input[i+2 : i+6]
			if code, err := strconv.ParseUint(hex, 16, 32); err == nil {
				b.WriteRune(rune(code))
				i += 6
				continue
			}
		}
		b.WriteByte(input[i])
		i++
	}
	return b.String()
}
