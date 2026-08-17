// Package services 收纳暴露给前端（Wails 绑定）的服务：
// 每个工具一个 Service 结构体，方法直接调用 internal/transform 等纯逻辑层，
// 本层不做业务逻辑，只做 Go↔前端的桥接。
package services

import (
	"tool-dev/internal/transform"
)

// JSONService JSON 工具的后端服务（操作均返回 结果+错误，错误显示在前端提示区）
type JSONService struct{}

// OpResult 是所有操作的统一返回：成功 Content 有值，失败 Error 有值
type OpResult struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

// Pretty 格式化（indent 为缩进字符串，如 "  " 或 "\t"）
func (s *JSONService) Pretty(input, indent string) OpResult {
	tr := transform.NewJSONTransformer(input)
	tr.SetIndent(indent)
	if err := tr.DoPretty(); err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: tr.GetString()}
}

// Minify 压缩
func (s *JSONService) Minify(input string) OpResult {
	tr := transform.NewJSONTransformer(input)
	if err := tr.DoMinify(); err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: tr.GetString()}
}

// Escape 转义
func (s *JSONService) Escape(input string) OpResult {
	tr := transform.NewJSONTransformer(input)
	if err := tr.DoEscape(); err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: tr.GetString()}
}

// Unescape 反转义
func (s *JSONService) Unescape(input string) OpResult {
	tr := transform.NewJSONTransformer(input)
	if err := tr.DoUnescape(); err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: tr.GetString()}
}

// ChineseToUnicode 中文 → Unicode
func (s *JSONService) ChineseToUnicode(input string) OpResult {
	tr := transform.NewJSONTransformer(input)
	if err := tr.DoChineseToUnicode(); err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: tr.GetString()}
}

// UnicodeToChinese Unicode → 中文
func (s *JSONService) UnicodeToChinese(input string) OpResult {
	tr := transform.NewJSONTransformer(input)
	if err := tr.DoUnicodeToChinese(); err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: tr.GetString()}
}
