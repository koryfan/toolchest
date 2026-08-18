package services

import (
	"reflect"

	"tool-dev/internal/config"
	"tool-dev/internal/domain"
	"tool-dev/internal/toolset"
)

// AppService 应用级服务：工具配置下发 + 操作转发 + 内容镜像同步。
// 旧 JSONService/TimeService 在前端迁移完成后删除。
type AppService struct {
	cfg *config.Config
}

// NewAppService 构造（cfg 由 main 注入，唯一的落盘通道）
func NewAppService(cfg *config.Config) *AppService {
	return &AppService{cfg: cfg}
}

// Tools 全量工具描述（元信息 + 操作区配置）
func (s *AppService) Tools() []domain.ToolDesc {
	return toolset.ToolDescs()
}

// Invoke 转发操作；持久化状态变化时落盘（写失败不阻断操作）
func (s *AppService) Invoke(toolID, op, arg string) domain.Result {
	t := toolset.Find(toolID)
	if t == nil {
		return domain.Result{IsError: true, Notice: "未知工具: " + toolID}
	}
	before := toolset.States()
	r := t.Invoke(op, arg)
	// spec：改变状态时同步写盘——快照对比，没变就不写
	if after := toolset.States(); !reflect.DeepEqual(before, after) {
		for id, st := range after {
			s.cfg.SetState(id, st)
		}
		_ = s.cfg.Save() // 写失败不阻断操作
	}
	return r
}

// UpdateContent 前端 debounce 推送的内容镜像同步
func (s *AppService) UpdateContent(toolID, content string) {
	if t := toolset.Find(toolID); t != nil {
		t.SetContent(content)
	}
}
