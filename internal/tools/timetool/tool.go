// Package timetool 时间戳工具：输入（秒/毫秒时间戳或日期串）→ 多格式结果。
// 解析结果走 Result.Data（多字段），解析纯函数在本包 parse.go。
package timetool

import (
	"strconv"
	"sync"
	"time"

	"toolchest/internal/domain"
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
		"nowTime": t.opNowTime,
		"now":     t.opNow,
		"parse":   t.opParse, // 前端输入触发（不作为按钮/菜单声明在 Actions）
	}
	return t
}

func (t *Tool) Meta() domain.Meta {
	return domain.Meta{ID: "time", Title: "时间戳转换", Icon: "time", ContentType: "time"}
}

// Actions 填充当前时间 / 当前时间戳两个按钮
func (t *Tool) Actions() []domain.Action {
	return []domain.Action{
		domain.NewButton("nowTime", "当前时间", ""),
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
// Invoke 操作分发（基座统一实现：锁内查表执行回调）
func (t *Tool) Invoke(op string, arg string) domain.Result {
	return domain.Dispatch(&t.mu, t.handlers, op, arg)
}

// --- 操作回调（锁内执行） ---

// opNowTime 填充当前本地时间（与解析器 LocalTime 同款格式，填充后可直接解析）
func (t *Tool) opNowTime(arg string) domain.Result {
	ts := time.Now().Format("2006-01-02 15:04:05 -0700 MST")
	t.content = ts
	return domain.Result{Content: ts}
}

// opNow 填充当前秒级时间戳
func (t *Tool) opNow(arg string) domain.Result {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	t.content = ts
	return domain.Result{Content: ts}
}

// opParse 解析当前输入为多格式结果
// opParse 解析当前输入为多格式结果
func (t *Tool) opParse(arg string) domain.Result {
	if t.content == "" {
		return domain.Result{IsError: true, Notice: "请输入时间戳或日期"}
	}
	pr := ParseTime(t.content, nil)
	if pr.Error != "" {
		return domain.Result{IsError: true, Notice: pr.Error}
	}
	return domain.Result{Data: map[string]string{
		"localTime": pr.LocalTime, "utcTime": pr.UTCTime,
		"relativeTime": pr.RelativeTime, "weekday": pr.Weekday,
		"unixSecond": pr.UnixSecond, "unixMilli": pr.UnixMilli,
	}}
}

// State 无持久化状态
func (t *Tool) State() any { return nil }

// RunningInBackground 无后台任务
func (t *Tool) RunningInBackground() bool { return false }
