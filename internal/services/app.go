package services

import (
	"tool-dev/internal/config"
	"tool-dev/internal/tools"
	"tool-dev/internal/toolset"
)

// AppService 应用级服务：工具配置下发 + 操作转发 + 内容镜像同步。
// 旧 JSONService/TimeService 在前端迁移完成后删除。
type AppService struct{}

// Tools 全量工具描述（元信息 + 操作区配置）
func (s *AppService) Tools() []tools.ToolDesc {
	return toolset.ToolDescs()
}

// Invoke 转发操作；状态变化时全量落盘（写失败不阻断操作）
func (s *AppService) Invoke(toolID, op, arg string) tools.Result {
	t := toolset.Find(toolID)
	if t == nil {
		return tools.Result{IsError: true, Notice: "未知工具: " + toolID}
	}
	r := t.Invoke(op, arg)
	_ = config.Save(toolset.Snapshot())
	return r
}

// UpdateContent 前端 debounce 推送的内容镜像同步
func (s *AppService) UpdateContent(toolID, content string) {
	if t := toolset.Find(toolID); t != nil {
		t.SetContent(content)
	}
}
