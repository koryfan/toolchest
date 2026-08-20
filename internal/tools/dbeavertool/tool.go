package dbeavertool

import (
	"sync"

	"tool-dev/internal/domain"
)

// Tool DBeaver 凭据查看工具（应用层）：编排领域纯函数 OpenWorkspace，
// 维护目录记忆（最近 8 个 + 当前目录）——应用层便利状态，持久化在 State。
type Tool struct {
	mu       sync.Mutex
	state    State
	handlers map[string]domain.OpHandler
}

// New 创建（state 由装配层注入）
func New(st State) *Tool {
	t := &Tool{state: st}
	t.handlers = map[string]domain.OpHandler{
		"select":  t.opSelect,  // arg=目录：记忆 + 解密
		"refresh": t.opRefresh, // 重解密当前目录
	}
	return t
}

func (t *Tool) Meta() domain.Meta {
	return domain.Meta{ID: "dbeaver", Title: "DBeaver 凭据", Icon: "dbeaver", ContentType: "dbeaver"}
}

// Actions 工具级操作：选择目录（前端本地开 file picker）、目录下拉（缓存）、刷新
func (t *Tool) Actions() []domain.Action {
	t.mu.Lock()
	defer t.mu.Unlock()

	opts := make([]domain.RadioOption, 0, len(t.state.RecentDirs))
	for _, d := range t.state.RecentDirs {
		opts = append(opts, domain.RadioOption{Value: d, Label: d})
	}
	current := t.state.CurrentDir
	if current == "" {
		current = "选择 DBeaver 工作目录"
	}

	return []domain.Action{
		domain.NewButton("pick", "选择目录", "dbeaverPickDir"),
		domain.NewRadio("select", "目录", opts, current),
		domain.NewButton("refresh", "刷新", ""),
	}
}

// SetContent 无文本内容
func (t *Tool) SetContent(s string) {}

// State 持久化状态
func (t *Tool) State() any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// RunningInBackground 无后台任务
func (t *Tool) RunningInBackground() bool { return false }

// Invoke 操作分发
func (t *Tool) Invoke(op string, arg string) domain.Result {
	h, ok := t.handlers[op]
	if !ok {
		return domain.Result{IsError: true, Notice: "未知操作: " + op}
	}
	return h(arg)
}

// result 载荷：凭据列表 + 动作刷新（目录下拉更新）
func (t *Tool) result(entries []CredentialEntry, err error) domain.Result {
	r := domain.Result{Actions: t.Actions()}
	if err != nil {
		r.IsError = true
		r.Notice = err.Error()
		return r
	}
	r.Data = map[string]any{"entries": entries}
	return r
}

func (t *Tool) opSelect(arg string) domain.Result {
	entries, err := OpenWorkspace(arg)
	if err != nil {
		// 打开失败的目录不进下拉缓存
		return t.result(nil, err)
	}
	t.mu.Lock()
	t.state.pushRecent(arg)
	t.mu.Unlock()
	return t.result(entries, nil)
}

func (t *Tool) opRefresh(arg string) domain.Result {
	t.mu.Lock()
	dir := t.state.CurrentDir
	t.mu.Unlock()
	if dir == "" {
		return domain.Result{IsError: true, Notice: "尚未选择目录", Actions: t.Actions()}
	}
	entries, err := OpenWorkspace(dir)
	return t.result(entries, err)
}
