package services

import (
	"toolchest/internal/domain"
	"toolchest/internal/toolset"
)

// AppService 应用级服务：Wails 边界适配——工具描述下发、操作转发、
// 内容镜像同步。装配与落盘在 internal/toolset；这里只做桥接。
type AppService struct{}

// NewAppService 构造
func NewAppService() *AppService {
	return &AppService{}
}

// Tools 全量工具描述（元信息 + 操作区配置 + 后台运行信号）
func (s *AppService) Tools() []domain.ToolDesc {
	return toolset.ToolDescs()
}

// Invoke 转发操作（落盘在 toolset.Invoke 内收口）
func (s *AppService) Invoke(toolID, op, arg string) domain.Result {
	return toolset.Invoke(toolID, op, arg)
}

// UpdateContent 前端 debounce 推送的内容镜像同步
func (s *AppService) UpdateContent(toolID, content string) {
	if t := toolset.Find(toolID); t != nil {
		t.SetContent(content)
	}
}
