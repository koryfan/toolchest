// Package toolset 工具注册表：装配全部内置工具（config.Decode 注入各工具 State）。
// 契约在 internal/domain；这里只做装配。加新工具 = 建子包 + 这里加一行。
package toolset

import (
	"toolchest/internal/config"
	"toolchest/internal/domain"
	"toolchest/internal/tools/dbeavertool"
	"toolchest/internal/tools/jsontool"
	"toolchest/internal/tools/timetool"
	"toolchest/internal/tools/tunneltool"
	"toolchest/internal/tools/uuidtool"
)

var all []domain.Tool

// Init 装配全部工具（cfg 已 Load；各节经泛型 Decode 转强类型，缺省/损坏用默认值）
func Init(cfg *config.Config) {
	jsState, ok := config.Decode[jsontool.State](cfg, "json")
	if !ok {
		jsState = jsontool.DefaultState()
	}
	tunState, ok := config.Decode[tunneltool.State](cfg, "tunnel")
	if !ok {
		tunState = tunneltool.DefaultState()
	}
	dbState, ok := config.Decode[dbeavertool.State](cfg, "dbeaver")
	if !ok {
		dbState = dbeavertool.DefaultState()
	}
	uuidState, ok := config.Decode[uuidtool.State](cfg, "uuid")
	if !ok {
		uuidState = uuidtool.DefaultState()
	}

	all = []domain.Tool{
		jsontool.New(jsState),
		timetool.New(),
		tunneltool.New(tunState, emitTunnelUpdate),
		dbeavertool.New(dbState),
		uuidtool.New(uuidState),
	}

	// 容器里始终存强类型 State（保证 Save 输出结构稳定）
	cfg.SetState("json", jsState)
	cfg.SetState("tunnel", tunState)
	cfg.SetState("dbeaver", dbState)
	cfg.SetState("uuid", uuidState)
}

// All 全部注册工具（未 Init 时为空）
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

// States 全量持久化状态快照（AppService 落盘对比用）
func States() map[string]any {
	out := map[string]any{}
	for _, t := range All() {
		if st := t.State(); st != nil {
			out[t.Meta().ID] = st
		}
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
