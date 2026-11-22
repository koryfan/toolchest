// Package jsontool JSON 工具：配置 + 状态 + 操作分发内聚一处。
// 操作以回调函数（OpHandler）直接挂在 handlers 表上，
// Invoke 只需查找调用——加功能 = 配置里加一项 + handlers 加一个回调。
package jsontool

import (
	"sync"

	"toolchest/internal/domain"
)

// indentOptions 缩进选项（value 即 Pretty 的缩进实参）
var indentOptions = []domain.RadioOption{
	{Value: "  ", Label: "2 空格"},
	{Value: "    ", Label: "4 空格"},
	{Value: "\t", Label: "Tab"},
}

// State 持久化状态（config.yaml 的 json 节）
type State struct {
	Indent string `yaml:"indent"`
}

// DefaultState 默认 2 空格缩进
func DefaultState() State { return State{Indent: "  "} }

// Tool JSON 工具（内容镜像 + 缩进设置状态 + 操作回调表）
type Tool struct {
	mu       sync.Mutex
	content  string
	state    State
	handlers map[string]domain.OpHandler
}

// New 创建（state 由装配层经 config.Decode 注入，零值降级为默认）
func New(st State) *Tool {
	if st.Indent == "" {
		st = DefaultState()
	}
	t := &Tool{state: st}
	t.handlers = map[string]domain.OpHandler{
		"setIndent": t.opSetIndent, // 选中即生效：改状态 + 立即重格式化
	}
	for op, d := range transformOps {
		t.handlers[op] = func(string) domain.Result { return t.runTransform(op, d) }
	}
	return t
}

func (t *Tool) Meta() domain.Meta {
	return domain.Meta{ID: "json", Title: "JSON 工具", Icon: "json", ContentType: "json"}
}

// Actions 操作区配置：清空/复制（前端自理）+ 格式化 + 设置（缩进）+ 更多
// 声明的每个 op ID 都必须能在 handlers 表找到（toolset_test 覆盖检查兜底）
func (t *Tool) Actions() []domain.Action {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.actions()
}

// actions 内部无锁版（调用方须持有 t.mu；Invoke 在锁内回调场景用）
func (t *Tool) actions() []domain.Action {
	cur := t.state.Indent
	return []domain.Action{
		domain.NewButton("clear", "清空", "clear"),
		domain.NewButton("copy", "复制", "copy"),
		domain.NewButton("pretty", "格式化", ""),
		domain.NewRadio("setIndent", "设置", indentOptions, cur),
		domain.NewMenu("more", "更多", []domain.MenuItem{
			{ID: "minify", Label: "压缩"},
			{ID: "escape", Label: "转义"},
			{ID: "unescape", Label: "反转义"},
			{ID: "c2u", Label: "中文 → Unicode"},
			{ID: "u2c", Label: "Unicode → 中文"},
		}),
	}
}

// SetContent 前端镜像同步（debounce 推送）
func (t *Tool) SetContent(s string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.content = s
}

// Invoke 操作分发：在锁内查找并执行回调。op 取自 Actions 配置的 ID。
// Invoke 操作分发（基座统一实现：锁内查表执行回调）
func (t *Tool) Invoke(op string, arg string) domain.Result {
	return domain.Dispatch(&t.mu, t.handlers, op, arg)
}

// --- 操作回调（锁内执行，可直接读写 t.content/t.indent） ---

func (t *Tool) opSetIndent(arg string) domain.Result {
	t.state.Indent = arg
	if trimEmpty(t.content) {
		return domain.Result{Actions: t.actions()}
	}
	out, err := Pretty(t.content, arg)
	if err != nil {
		return domain.Result{IsError: true, Notice: err.Error(), Actions: t.actions()}
	}
	return domain.Result{Content: out, Notice: "格式化完成", Actions: t.actions()}
}

// transformOp 内容转换单元：纯函数 + 完成提示
type transformOp struct {
	fn     func(content, indent string) (string, error)
	notice string
}

// transformOps op ID → 转换（单一事实源：handlers 表由此派生，
// Actions 菜单项 ID 与这里的键一一对应）
var transformOps = map[string]transformOp{
	"pretty":   {func(c, i string) (string, error) { return Pretty(c, i) }, "格式化完成"},
	"minify":   {func(c, _ string) (string, error) { return Minify(c) }, "压缩完成"},
	"escape":   {func(c, _ string) (string, error) { return Escape(c) }, "转义完成"},
	"unescape": {func(c, _ string) (string, error) { return Unescape(c) }, "反转义完成"},
	"c2u":      {func(c, _ string) (string, error) { return ChineseToUnicode(c) }, "中文 → Unicode 完成"},
	"u2c":      {func(c, _ string) (string, error) { return UnicodeToChinese(c) }, "Unicode → 中文 完成"},
}

// runTransform 走 op 对应的转换纯函数（锁内回调，直接读 t.content）。
// 空白内容与 opSetIndent 同语义：静默返回原内容，不报解析错误
func (t *Tool) runTransform(op string, d transformOp) domain.Result {
	if trimEmpty(t.content) {
		return domain.Result{}
	}
	out, err := d.fn(t.content, t.state.Indent)
	if err != nil {
		return domain.Result{IsError: true, Notice: err.Error()}
	}
	return domain.Result{Content: out, Notice: d.notice}
}

func trimEmpty(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// State 持久化状态快照（内容不入盘）
func (t *Tool) State() any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// RunningInBackground JSON 工具无后台任务
func (t *Tool) RunningInBackground() bool { return false }
