// Package toolset 工具注册表：装配全部内置工具（config.Decode 注入各工具 State），
// 并承接 Invoke 的状态变化落盘。契约在 internal/domain；这里只做装配与编排。
package toolset

import (
	"bytes"
	"encoding/json"

	"toolchest/internal/config"
	"toolchest/internal/domain"
	"toolchest/internal/tools/dbeavertool"
	"toolchest/internal/tools/jsontool"
	"toolchest/internal/tools/timetool"
	"toolchest/internal/tools/tunneltool"
	"toolchest/internal/tools/uuidtool"
)

var (
	all []domain.Tool
	cfg *config.Config
)

// Init 装配全部工具（cfg 已 Load；各节经泛型 Decode 转强类型，缺省/损坏用默认值）
func Init(c *config.Config) {
	cfg = c
	all = []domain.Tool{
		jsontool.New(state(cfg, "json", jsontool.DefaultState)),
		timetool.New(),
		tunneltool.New(state(cfg, "tunnel", tunneltool.DefaultState), func() { emitToolUpdate("tunnel") }),
		dbeavertool.New(state(cfg, "dbeaver", dbeavertool.DefaultState)),
		uuidtool.New(state(cfg, "uuid", uuidtool.DefaultState)),
	}
}

// state 有状态工具装配两步收口：Decode（缺省/损坏用默认值）→
// 容器回写强类型 State（保证 Save 输出结构稳定）。
// SetState 收在 helper 内，杜绝装配处漏写导致该节保存静默丢失。
func state[T any](c *config.Config, section string, def func() T) T {
	st, ok := config.Decode[T](c, section)
	if !ok {
		st = def()
	}
	c.SetState(section, st)
	return st
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

// Invoke 操作转发 + 持久化状态变化时落盘（写失败不阻断操作）
func Invoke(toolID, op, arg string) domain.Result {
	t := Find(toolID)
	if t == nil {
		return domain.Result{IsError: true, Notice: "未知工具: " + toolID}
	}
	before := statesJSON()
	r := t.Invoke(op, arg)
	// spec：改变状态时同步写盘——快照对比，没变就不写；SetState 必须存
	// 强类型 State（容器里的类型决定 Save 输出结构）
	if after := statesJSON(); !bytes.Equal(before, after) {
		for id, st := range States() {
			cfg.SetState(id, st)
		}
		_ = cfg.Save()
	}
	return r
}

// statesJSON 状态快照（JSON 字节）。直接引用 States() 做对比
// 会共享 slice 底层数组：工具原地替换元素（如 Tunnels[i] = cfg）时
// before 快照跟着变，DeepEqual 恒等，落盘被跳过；序列化字节对比天然隔离。
func statesJSON() []byte {
	b, err := json.Marshal(States())
	if err != nil {
		return nil
	}
	return b
}

// States 全量持久化状态快照（落盘对比用）
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
		out = append(out, domain.ToolDesc{Meta: t.Meta(), Actions: t.Actions(), Running: t.RunningInBackground()})
	}
	return out
}
