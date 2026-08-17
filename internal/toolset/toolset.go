// Package toolset 工具注册表：列出全部内置工具（对应 Fyne 期 pages.All 模式）。
// 契约在 internal/tools；这里只做装配。加新工具 = 建子包 + 这里加一行。
package toolset

import (
	"tool-dev/internal/config"
	"tool-dev/internal/domain"
	"tool-dev/internal/tools/jsontool"
	"tool-dev/internal/tools/timetool"
)

var all = []domain.Tool{
	jsontool.New(),
	timetool.New(),
}

// All 全部注册工具
func All() []domain.Tool { return append([]domain.Tool{}, all...) }

// Find 按工具 ID 查找
func Find(id string) domain.Tool {
	for _, t := range All() {
		if t.Meta().ID == id {
			return t
		}
	}
	return nil
}

// Init 启动时灌入持久化状态
func Init(states config.States) {
	for _, t := range All() {
		if s, ok := states[t.Meta().ID]; ok {
			t.LoadState(s)
		}
	}
}

// Snapshot 全量状态快照（持久化用）
func Snapshot() config.States {
	out := config.States{}
	for _, t := range All() {
		out[t.Meta().ID] = t.SaveState()
	}
	return out
}

// ToolDescs 全量工具描述（services.Tools() 直接下发）
func ToolDescs() []domain.ToolDesc {
	ts := All()
	out := make([]domain.ToolDesc, 0, len(ts))
	for _, t := range ts {
		out = append(out, domain.ToolDesc{Meta: t.Meta(), Actions: t.Actions()})
	}
	return out
}
