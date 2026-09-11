package services

import (
	"bytes"
	"encoding/json"

	"toolchest/internal/config"
	"toolchest/internal/domain"
	"toolchest/internal/toolset"
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
	before := statesJSON()
	r := t.Invoke(op, arg)
	// spec：改变状态时同步写盘——快照对比，没变就不写；SetState 必须存
	// 强类型 State（容器里的类型决定 Save 输出结构）
	if after := statesJSON(); !bytes.Equal(before, after) {
		for id, st := range toolset.States() {
			s.cfg.SetState(id, st)
		}
		_ = s.cfg.Save() // 写失败不阻断操作
	}
	return r
}

// statesJSON 状态快照（JSON 字节）。直接引用 toolset.States() 做对比
// 会共享 slice 底层数组：工具原地替换元素（如 Tunnels[i] = cfg）时
// before 快照跟着变，DeepEqual 恒等，落盘被跳过；序列化字节对比天然隔离。
func statesJSON() []byte {
	b, err := json.Marshal(toolset.States())
	if err != nil {
		return nil
	}
	return b
}

// UpdateContent 前端 debounce 推送的内容镜像同步
func (s *AppService) UpdateContent(toolID, content string) {
	if t := toolset.Find(toolID); t != nil {
		t.SetContent(content)
	}
}
