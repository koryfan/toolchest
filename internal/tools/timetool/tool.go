// Package timetool 时间戳工具：输入（秒/毫秒时间戳或日期串）→ 多格式结果。
// 解析结果走 Result.Data（多字段），解析逻辑在 internal/transform。
package timetool

import (
	"strconv"
	"sync"
	"time"

	"tool-dev/internal/tools"
	"tool-dev/internal/transform"
)

// Tool 时间戳工具（内容镜像即输入行；无设置类状态）
type Tool struct {
	mu      sync.Mutex
	content string
}

// New 创建
func New() *Tool { return &Tool{} }

func (t *Tool) Meta() tools.Meta {
	return tools.Meta{ID: "time", Title: "时间戳转换", Icon: "time", ContentType: "time"}
}

// Actions 当前只有一个操作：填充当前时间戳
func (t *Tool) Actions() []tools.Action {
	return []tools.Action{
		tools.NewButton("now", "当前时间戳", ""),
	}
}

// SetContent 前端镜像同步
func (t *Tool) SetContent(s string) {
	t.mu.Lock()
	t.content = s
	t.mu.Unlock()
}

// Invoke 操作分发
func (t *Tool) Invoke(op string, arg string) tools.Result {
	switch op {
	case "now":
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		t.mu.Lock()
		t.content = ts
		t.mu.Unlock()
		return tools.Result{Content: ts}
	case "parse":
		t.mu.Lock()
		content := t.content
		t.mu.Unlock()
		if content == "" {
			return tools.Result{IsError: true, Notice: "请输入时间戳或日期"}
		}
		p := transform.NewTimeParser()
		p.SetInput(content)
		p.Parse()
		pr := p.Result()
		if pr.Error != "" {
			return tools.Result{IsError: true, Notice: pr.Error}
		}
		return tools.Result{Data: map[string]string{
			"localTime": pr.LocalTime, "utcTime": pr.UTCTime,
			"relativeTime": pr.RelativeTime, "weekday": pr.Weekday,
			"unixSecond": pr.UnixSecond, "unixMilli": pr.UnixMilli,
		}}
	default:
		return tools.Result{IsError: true, Notice: "未知操作: " + op}
	}
}

func (t *Tool) SaveState() map[string]string { return map[string]string{} }
func (t *Tool) LoadState(map[string]string)  {}
