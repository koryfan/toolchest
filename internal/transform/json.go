// Package transform 提供各工具的业务转换逻辑。
// JSON 工具通过 JSONTransformer 结构体持有输入状态，
// 各 Do* 方法原地修改状态；通用的字符串操作委托给 pkg/strutil。
package transform

import (
	"encoding/json"
	"fmt"

	"tool-dev/pkg/strutil"
)

// JSONTransformer 是 JSON 工具的业务封装。
// 持有当前输入内容，Do* 方法原地修改它，GetString() 取结果。
type JSONTransformer struct {
	Indent string // 缩进字符，默认两空格（MakePretty 用）
	input  []byte // 当前内容
}

// NewJSONTransformer 用输入字符串构造转换器，缩进默认两空格
func NewJSONTransformer(input string) *JSONTransformer {
	return &JSONTransformer{
		Indent: "  ",
		input:  []byte(input),
	}
}

// SetIndent 设置缩进字符（如 "  " 或 "\t"）
func (t *JSONTransformer) SetIndent(indent string) {
	t.Indent = indent
}

// GetString 返回当前内容
func (t *JSONTransformer) GetString() string {
	return string(t.input)
}

// DoPretty 把当前内容格式化成带缩进的 JSON（用 t.Indent）
func (t *JSONTransformer) DoPretty() error {
	var data any
	if err := json.Unmarshal(t.input, &data); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}
	out, err := json.MarshalIndent(data, "", t.Indent)
	if err != nil {
		return fmt.Errorf("格式化失败: %w", err)
	}
	t.input = out
	return nil
}

// DoMinify 去除 JSON 中的空白，压缩成最短形式
func (t *JSONTransformer) DoMinify() error {
	var data any
	if err := json.Unmarshal(t.input, &data); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}
	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("压缩失败: %w", err)
	}
	t.input = out
	return nil
}

// DoEscape 把当前内容转成 JSON 字符串字面量（带外层双引号）
func (t *JSONTransformer) DoEscape() error {
	t.input = []byte(strutil.Escape(string(t.input)))
	return nil
}

// DoUnescape 把 JSON 字符串字面量还原成原始字符串
// 输入不带外层引号会自动补；非法转义序列返回 error
func (t *JSONTransformer) DoUnescape() error {
	out, err := strutil.Unescape(string(t.input))
	if err != nil {
		return fmt.Errorf("反转义失败: %w", err)
	}
	t.input = []byte(out)
	return nil
}

// DoChineseToUnicode 把非 ASCII 字符（含中文）转成 \uXXXX
func (t *JSONTransformer) DoChineseToUnicode() error {
	t.input = []byte(strutil.ChineseToUnicode(string(t.input)))
	return nil
}

// DoUnicodeToChinese 把 \uXXXX 转回对应字符
func (t *JSONTransformer) DoUnicodeToChinese() error {
	t.input = []byte(strutil.UnicodeToChinese(string(t.input)))
	return nil
}
