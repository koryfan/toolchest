package toolset

import (
	"testing"

	"tool-dev/internal/config"
	"tool-dev/internal/domain"
)

// 全部注册的工具：Actions 里声明的每个非 local 操作，Invoke 必须有实现
// （防字符串分发写错——删了实现忘删配置、或反之，测试直接红）
func TestAllToolsOpsImplemented(t *testing.T) {
	for _, tl := range All() {
		for _, a := range tl.Actions() {
			var opIDs []string
			switch a.Type {
			case domain.ActionButton:
				if a.Local == "" {
					opIDs = append(opIDs, a.ID)
				}
			case domain.ActionMenu:
				for _, it := range a.Items {
					opIDs = append(opIDs, it.ID)
				}
			case domain.ActionRadio:
				opIDs = append(opIDs, a.ID) // radio 选中即 Invoke(a.ID, value)
			}
			for _, op := range opIDs {
				// 用空内容试调用：只求「未知操作」路径不命中，不校验业务结果
				r := tl.Invoke(op, "")
				if r.IsError && r.Notice == "未知操作: "+op {
					t.Errorf("%s/%s: Actions 声明了但 Invoke 未实现", tl.Meta().ID, op)
				}
			}
		}
	}
}

func TestFindAndInit(t *testing.T) {
	if Find("json") == nil || Find("time") == nil {
		t.Fatalf("json/time 工具应已注册")
	}
	if Find("nope") != nil {
		t.Fatalf("未知 ID 应返回 nil")
	}
	Init(config.States{"json": {"indent": "\t"}})
	snap := Snapshot()
	if snap["json"] == nil || snap["time"] == nil {
		t.Fatalf("Snapshot 应覆盖全部注册工具: %#v", snap)
	}
}
