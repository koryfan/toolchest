// JSON 转换纯函数：无状态，输入输出都是字符串。
// 逻辑从 internal/transform 并入（jsontool 自己已持有 content+indent，
// 无需再经 JSONTransformer 中转一层）；通用字符串操作委托 pkg/strutil。
package jsontool

import (
	"encoding/json"
	"fmt"

	"toolchest/pkg/strutil"
)

// Pretty 格式化 JSON（indent 为缩进字符，如 "  " 或 "\t"）
func Pretty(input, indent string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return "", fmt.Errorf("JSON 解析失败: %w", err)
	}
	out, err := json.MarshalIndent(data, "", indent)
	if err != nil {
		return "", fmt.Errorf("格式化失败: %w", err)
	}
	return string(out), nil
}

// Minify 去掉空白压缩成最短形式
func Minify(input string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return "", fmt.Errorf("JSON 解析失败: %w", err)
	}
	out, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("压缩失败: %w", err)
	}
	return string(out), nil
}

// Escape 把内容转成 JSON 字符串字面量（带外层双引号）
func Escape(input string) (string, error) {
	return strutil.Escape(input), nil
}

// Unescape 把 JSON 字符串字面量还原成原始字符串；非法转义序列返回 error
func Unescape(input string) (string, error) {
	out, err := strutil.Unescape(input)
	if err != nil {
		return "", fmt.Errorf("反转义失败: %w", err)
	}
	return out, nil
}

// ChineseToUnicode 把非 ASCII 字符（含中文）转成 \uXXXX
func ChineseToUnicode(input string) (string, error) {
	return strutil.ChineseToUnicode(input), nil
}

// UnicodeToChinese 把 \uXXXX 转回对应字符
func UnicodeToChinese(input string) (string, error) {
	return strutil.UnicodeToChinese(input), nil
}
