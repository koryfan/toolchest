// Package strutil 提供通用的字符串处理工具函数。
// 这些函数无业务耦合、无状态，可被项目任意层级复用。
package strutil

import (
	"encoding/json"
	"strings"
)

// Escape 把任意字符串转成 JSON 字符串字面量（带外层双引号）
// 比如 hello\nworld → "hello\nworld"
// 标准库 json.Marshal(string) 会自动处理换行、引号、反斜杠等转义
func Escape(input string) string {
	out, err := json.Marshal(input)
	if err != nil {
		// json.Marshal 对任意 string 都不会失败，走到这里属于标准库异常
		return input
	}
	return string(out)
}

// Unescape 把 JSON 字符串字面量还原成原始字符串
// 输入需带外层双引号；如果不带会自动补
// 非法转义序列（如 \xyz）返回 error
func Unescape(input string) (string, error) {
	s := strings.TrimSpace(input)
	if !strings.HasPrefix(s, "\"") {
		s = "\"" + s + "\""
	}
	var result string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return "", err
	}
	return result, nil
}
