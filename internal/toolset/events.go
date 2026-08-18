package toolset

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"tool-dev/internal/tools/tunneltool"
)

// TunnelUpdateEvent tunnel-update 事件名（前端 Events.On 订阅）
const TunnelUpdateEvent = "tunnel-update"

// emitTunnelUpdate 隧道状态变化 → 推全量列表给前端。
// emit 可能来自后台 goroutine（掉线检测），application.EmitEvent 线程安全。
func emitTunnelUpdate() {
	t := Find("tunnel")
	if t == nil {
		return
	}
	tun := t.(*tunneltool.Tool)
	application.Get().Event.Emit(TunnelUpdateEvent, tun.Data())
}
