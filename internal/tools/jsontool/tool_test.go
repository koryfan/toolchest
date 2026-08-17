package jsontool

import (
	"strings"
	"testing"
)

const src = `{"name":"工具","age":3,"list":[1,2]}`

func newTool() *Tool { t := New(); t.SetContent(src); return t }

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestMeta(t *testing.T) {
	m := New().Meta()
	if m.ID != "json" || m.Title != "JSON 工具" || m.Icon != "json" || m.ContentType != "json" {
		t.Fatalf("Meta 不符: %#v", m)
	}
}

// 操作表：输入 → 期望内容（经 transform，算法本身有单测，这里只验分发与结果写回）
func TestInvokeOps(t *testing.T) {
	tests := []struct {
		name, op, arg, wantSub string // wantSub = 结果应包含的子串
	}{
		{"格式化默认缩进", "pretty", "", "\n  \"name\""},
		{"格式化Tab缩进", "setIndent", "\t", "\n\t\"name\""},
		{"压缩", "minify", "", "\"age\":3,\"list\""}, // minify 会按键排序
		{"转义", "escape", "", "\\\"name\\\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := newTool()
			r := tl.Invoke(tt.op, tt.arg)
			if r.IsError {
				t.Fatalf("%s 报错: %s", tt.op, r.Notice)
			}
			if !contains(r.Content, tt.wantSub) {
				t.Fatalf("%s 结果不含 %q: %q", tt.op, tt.wantSub, r.Content)
			}
		})
	}
}

// 反转义：输入须是已转义字面量。模拟前端流程——escape 结果经 SetContent 写回再 unescape 应还原
func TestUnescapeRoundtrip(t *testing.T) {
	tl := newTool()
	esc := tl.Invoke("escape", "")
	if esc.IsError {
		t.Fatalf("escape 报错: %s", esc.Notice)
	}
	tl.SetContent(esc.Content)
	r := tl.Invoke("unescape", "")
	if r.IsError || r.Content != src {
		t.Fatalf("escape→unescape 应还原原文: %q (err=%v)", r.Content, r.IsError)
	}
}

func TestSetIndentReturnsActions(t *testing.T) {
	tl := newTool()
	r := tl.Invoke("setIndent", "\t")
	if r.IsError || r.Actions == nil {
		t.Fatalf("setIndent 应带回新配置: %#v", r)
	}
	// 新配置里设置组的 Current 应是 \t
	for _, a := range r.Actions {
		if a.Type == "radio" && a.ID == "setIndent" && a.Current != "\t" {
			t.Fatalf("radio Current 应为 \\t: %#v", a)
		}
	}
	// 内容同时被重新格式化
	if !contains(r.Content, "\n\t\"name\"") {
		t.Fatalf("setIndent 应立即重新格式化: %q", r.Content)
	}
}

func TestSetContentThenInvoke(t *testing.T) {
	tl := New()
	tl.SetContent("  不合法  ")
	r := tl.Invoke("pretty", "")
	if !r.IsError {
		t.Fatalf("坏 JSON 应报错而非静默")
	}
}

func TestStateRoundtrip(t *testing.T) {
	tl := New()
	tl.Invoke("setIndent", "\t")
	st := tl.SaveState()
	tl2 := New()
	tl2.LoadState(st)
	if tl2.SaveState()["indent"] != "\t" {
		t.Fatalf("状态还原失败: %#v", tl2.SaveState())
	}
	// 还原后的缩进应作用于后续格式化
	tl2.SetContent(src)
	r := tl2.Invoke("pretty", "")
	if !contains(r.Content, "\n\t\"name\"") {
		t.Fatalf("还原后的缩进未生效: %q", r.Content)
	}
}

func TestUnknownOp(t *testing.T) {
	r := New().Invoke("nope", "")
	if !r.IsError || r.Notice != "未知操作: nope" {
		t.Fatalf("未知操作应有明确报错: %#v", r)
	}
}
