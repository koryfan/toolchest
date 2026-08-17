// Package timetool 时间戳工具：输入（秒/毫秒时间戳或日期串）→ 多格式结果。
// 解析结果走 Result.Data（多字段），解析逻辑在 internal/transform。
package timetool

import (
	"strconv"
	"sync"
	"time"

	"tool-dev/internal/domain"
	"tool-dev/internal/transform"
)

// Tool 时间戳工具（内容镜像即输入行；无设置类状态）
type Tool struct {
	mu       sync.Mutex
	content  string
	handlers map[string]domain.OpHandler
}

// New 创建
func New() *Tool {
	t := &Tool{}
	t.handlers = map[string]domain.OpHandler{
		"now":   t.opNow,
		"parse": t.opParse, // 前端输入触发（不作为按钮/菜单声明在 Actions）
	}
	return t
}

func (t *Tool) Meta() domain.Meta {
	return domain.Meta{ID: "time", Title: "时间戳转换", Icon: "time", ContentType: "time"}
}

// Actions 当前只有一个操作：填充当前时间戳
func (t *Tool) Actions() []domain.Action {
	return []domain.Action{
		domain.NewButton("now", "当前时间戳", ""),
	}
}

// SetContent 前端镜像同步
func (t *Tool) SetContent(s string) {
	t.mu.Lock()
	t.content = s
	t.mu.Unlock()
}

// Invoke 操作分发：在锁内查找并执行回调
func (t *Tool) Invoke(op string, arg string) domain.Result {
	t.mu.Lock()
	defer t.mu.Unlock()
	h, ok := t.handlers[op]
	if !ok {
		return domain.Result{IsError: true, Notice: "未知操作: " + op}
	}
	return h(arg)
}

// --- 操作回调（锁内执行） ---

// opNow 填充当前秒级时间戳
func (t *Tool) opNow(arg string) domain.Result {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	t.content = ts
	return domain.Result{Content: ts}
}

// opParse 解析当前输入为多格式结果
func (t *Tool) opParse(arg string) domain.Result {
	if t.content == "" {
		return domain.Result{IsError: true, Notice: "请输入时间戳或日期"}
	}
	p := transform.NewTimeParser()
	p.SetInput(t.content)
	p.Parse()
	pr := p.Result()
	if pr.Error != "" {
		return domain.Result{IsError: true, Notice: pr.Error}
	}
	return domain.Result{Data: map[string]string{
		"localTime": pr.LocalTime, "utcTime": pr.UTCTime,
		"relativeTime": pr.RelativeTime, "weekday": pr.Weekday,
		"unixSecond": pr.UnixSecond, "unixMilli": pr.UnixMilli,
	}}
}

func (t *Tool) SaveState() map[string]string { return map[string]string{} }
func (t *Tool) LoadState(map[string]string)  {}