// Package services 收纳暴露给前端（Wails 绑定）的服务：
// 每个工具一个 Service 结构体，方法走工具域（toolset/jsontool 等），
// 本层不做业务逻辑，只做 Go↔前端的桥接。
package services

import (
	"tool-dev/internal/tools/jsontool"
)

// JSONService JSON 工具的旧版服务（前端迁移到 AppService 后删除）
type JSONService struct{}

// OpResult 是所有操作的统一返回：成功 Content 有值，失败 Error 有值
type OpResult struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

// Pretty 格式化（indent 为缩进字符串，如 "  " 或 "\t"）
func (s *JSONService) Pretty(input, indent string) OpResult {
	content, err := jsontool.Pretty(input, indent)
	if err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: content}
}

// Minify 压缩
func (s *JSONService) Minify(input string) OpResult {
	content, err := jsontool.Minify(input)
	if err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: content}
}

// Escape 转义
func (s *JSONService) Escape(input string) OpResult {
	content, err := jsontool.Escape(input)
	if err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: content}
}

// Unescape 反转义
func (s *JSONService) Unescape(input string) OpResult {
	content, err := jsontool.Unescape(input)
	if err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: content}
}

// ChineseToUnicode 中文 → Unicode
func (s *JSONService) ChineseToUnicode(input string) OpResult {
	content, err := jsontool.ChineseToUnicode(input)
	if err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: content}
}

// UnicodeToChinese Unicode → 中文
func (s *JSONService) UnicodeToChinese(input string) OpResult {
	content, err := jsontool.UnicodeToChinese(input)
	if err != nil {
		return OpResult{Error: err.Error()}
	}
	return OpResult{Content: content}
}