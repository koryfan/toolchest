package timetool

import (
	"strings"
	"testing"
	"time"
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
	data, ok := r.Data.(map[string]string)
	if !ok {
		t.Fatalf("parse 结果应为 map: %#v", r.Data)
	}
	if !strings.HasPrefix(data["localTime"], "2024-07-03") {
		t.Fatalf("localTime 不符: %q", data["localTime"])
	}
	if data["unixSecond"] != "1720000000" || data["unixMilli"] != "1720000000000" {
		t.Fatalf("时间戳往返不符: %#v", data)
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

func TestNowTime(t *testing.T) {
	tl := New()
	r := tl.Invoke("nowTime", "")
	if r.IsError {
		t.Fatalf("nowTime 报错: %s", r.Notice)
	}
	// 填充格式固定为 2006-01-02 15:04:05 -0700 MST
	if _, err := time.Parse("2006-01-02 15:04:05 -0700 MST", r.Content); err != nil {
		t.Fatalf("nowTime 格式不符: %q: %v", r.Content, err)
	}
	// 填充值必须能被解析（往返可用）
	tl.SetContent(r.Content)
	pr := tl.Invoke("parse", "")
	if pr.IsError {
		t.Fatalf("nowTime 结果应可解析: %s", pr.Notice)
	}
}

func TestStateless(t *testing.T) {
	// 无设置类状态：Save 空 map、Load 不炸即可
	tl := New()
	if tl.State() != nil {
		t.Fatalf("time 工具无持久化状态: %#v", tl.State())
	}
}
