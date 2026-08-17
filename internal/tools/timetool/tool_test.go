package timetool

import (
	"strings"
	"testing"
)

func TestMeta(t *testing.T) {
	m := New().Meta()
	if m.ID != "time" || m.Title != "时间戳转换" || m.ContentType != "time" {
		t.Fatalf("Meta 不符: %#v", m)
	}
}

func TestParseTimestamp(t *testing.T) {
	tl := New()
	tl.SetContent("1720000000")
	r := tl.Invoke("parse", "")
	if r.IsError {
		t.Fatalf("解析报错: %s", r.Notice)
	}
	if !strings.HasPrefix(r.Data["localTime"], "2024-07-03") {
		t.Fatalf("localTime 不符: %q", r.Data["localTime"])
	}
	if r.Data["unixSecond"] != "1720000000" || r.Data["unixMilli"] != "1720000000000" {
		t.Fatalf("时间戳往返不符: %#v", r.Data)
	}
}

func TestParseInvalid(t *testing.T) {
	tl := New()
	tl.SetContent("abc")
	r := tl.Invoke("parse", "")
	if !r.IsError || r.Notice == "" {
		t.Fatalf("坏输入应报错: %#v", r)
	}
}

func TestNow(t *testing.T) {
	r := New().Invoke("now", "")
	if r.IsError || len(r.Content) != 10 {
		t.Fatalf("now 应返回 10 位秒级时间戳: %#v", r)
	}
}

func TestStateless(t *testing.T) {
	// 无设置类状态：Save 空 map、Load 不炸即可
	tl := New()
	if len(tl.SaveState()) != 0 {
		t.Fatalf("time 工具无持久化状态: %#v", tl.SaveState())
	}
	tl.LoadState(map[string]any{"x": 1})
}
