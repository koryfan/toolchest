package toolset

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"toolchest/internal/domain"
)

// ToolUpdateEvent tool-update 事件名（前端 Events.On 订阅，载荷 {id, data}）
const ToolUpdateEvent = "tool-update"

// emitToolUpdate 工具后台状态变化 → 经 DataReporter 契约取 Data 推给前端。
// emit 可能来自后台 goroutine（掉线检测），application.EmitEvent 线程安全。
func emitToolUpdate(id string) {
	t := Find(id)
	if t == nil {
		return
	}
	r, ok := t.(domain.DataReporter)
	if !ok {
		return
	}
	app := application.Get()
	if app == nil {
		return // 非 GUI 环境（单测）：无事件循环，跳过
	}
	app.Event.Emit(ToolUpdateEvent, map[string]any{"id": id, "data": r.Data()})
}
