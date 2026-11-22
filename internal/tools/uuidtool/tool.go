// Package uuidtool UUID 生成工具：版本（v4 全随机 / v7 时间有序）
// 与数量（1..10）为设置类状态，生成走 github.com/google/uuid，
// 结果每行一个写入 Content。
package uuidtool

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"

	"toolchest/internal/domain"
)

// MaxCount 单次生成数量上限
const MaxCount = 10

// Tool UUID 生成工具（生成结果即内容镜像；设置类状态持久化）
type Tool struct {
	mu       sync.Mutex
	state    State
	handlers map[string]domain.OpHandler
}

// New 创建（state 由装配层注入，非法值校正）
func New(st State) *Tool {
	st.normalize()
	t := &Tool{state: st}
	t.handlers = map[string]domain.OpHandler{
		"gen":        t.opGen,        // 生成（按钮）
		"setVersion": t.opSetVersion, // radio：arg=v4/v7
		"setCount":   t.opSetCount,   // radio：arg=数量
	}
	return t
}

func (t *Tool) Meta() domain.Meta {
	return domain.Meta{ID: "uuid", Title: "UUID 生成", Icon: "uuid", ContentType: "uuid"}
}

// Actions 工具级操作：版本/数量单选（标签带当前值）、生成、复制/清空（前端本地）
func (t *Tool) Actions() []domain.Action {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.actionsLocked()
}

// actionsLocked 构造操作区配置（调用方须持锁；Invoke 回调内复用免重入死锁）
func (t *Tool) actionsLocked() []domain.Action {

	// 数量只留常用档位（校验上限仍是 MaxCount）
	countOpts := make([]domain.RadioOption, 0, 4)
	for _, n := range []int{1, 2, 5, 10} {
		s := strconv.Itoa(n)
		countOpts = append(countOpts, domain.RadioOption{Value: s, Label: s})
	}

	return []domain.Action{
		domain.NewRadio("setVersion", "版本 "+strings.ToUpper(t.state.Version),
			[]domain.RadioOption{
				{Value: "v4", Label: "V4（随机）"},
				{Value: "v7", Label: "V7（时间有序）"},
			}, t.state.Version),
		domain.NewRadio("setCount", "数量 "+strconv.Itoa(t.state.Count),
			countOpts, strconv.Itoa(t.state.Count)),
		domain.NewButton("gen", "生成", ""),
		domain.NewButton("copy", "复制", "copy"),
		domain.NewButton("clear", "清空", "clear"),
	}
}

// SetContent 前端镜像同步（生成结果回写，无用户编辑）
func (t *Tool) SetContent(s string) {}

// State 持久化状态
func (t *Tool) State() any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// RunningInBackground 无后台任务
func (t *Tool) RunningInBackground() bool { return false }

// Invoke 操作分发：在锁内查找并执行回调
// Invoke 操作分发（回调含慢操作，自行加锁 → mu 传 nil）
func (t *Tool) Invoke(op string, arg string) domain.Result {
	return domain.Dispatch(nil, t.handlers, op, arg)
}

// --- 操作回调（锁内执行） ---

// opGen 按当前版本/数量生成，结果每行一个
func (t *Tool) opGen(arg string) domain.Result {
	out := make([]string, 0, t.state.Count)
	for range t.state.Count {
		var id uuid.UUID
		var err error
		if t.state.Version == "v7" {
			id, err = uuid.NewV7()
		} else {
			id = uuid.New()
		}
		if err != nil {
			return domain.Result{IsError: true, Notice: "生成失败: " + err.Error()}
		}
		out = append(out, id.String())
	}
	content := strings.Join(out, "\n")
	return domain.Result{
		Content: content,
		Notice:  fmt.Sprintf("已生成 %d 个 UUID（%s）", t.state.Count, strings.ToUpper(t.state.Version)),
		Actions: t.actionsLocked(),
	}
}

// opSetVersion 切换版本后立即生成
func (t *Tool) opSetVersion(arg string) domain.Result {
	if arg != "v4" && arg != "v7" {
		return domain.Result{IsError: true, Notice: "未知版本: " + arg}
	}
	t.state.Version = arg
	return t.opGen(arg)
}

// opSetCount 切换数量后立即生成
func (t *Tool) opSetCount(arg string) domain.Result {
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 || n > MaxCount {
		return domain.Result{IsError: true, Notice: "数量须在 1~" + strconv.Itoa(MaxCount) + " 之间"}
	}
	t.state.Count = n
	return t.opGen(arg)
}
