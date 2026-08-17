// Package jsontool JSON 工具：配置 + 状态 + 操作分发内聚一处。
// 操作以回调函数（OpHandler）直接挂在 handlers 表上，
// Invoke 只需查找调用——加功能 = 配置里加一项 + handlers 加一个回调。
package jsontool

import (
	"sync"

	"tool-dev/internal/domain"
)

// indentOptions 缩进选项（value 即 Pretty 的缩进实参）
var indentOptions = []domain.RadioOption{
	{Value: "  ", Label: "2 空格"},
	{Value: "    ", Label: "4 空格"},
	{Value: "\t", Label: "Tab"},
}

// Tool JSON 工具（内容镜像 + 缩进设置状态 + 操作回调表）
type Tool struct {
	mu       sync.Mutex
	content  string
	indent   string
	handlers map[string]domain.OpHandler
}

// New 创建（默认 2 空格缩进）
func New() *Tool {
	t := &Tool{indent: "  "}
	t.handlers = map[string]domain.OpHandler{
		"setIndent": t.opSetIndent, // 选中即生效：改状态 + 立即重格式化
		"pretty":    t.opPretty,
		"minify":    t.opMinify,
		"escape":    t.opEscape,
		"unescape":  t.opUnescape,
		"c2u":       t.opChineseToUnicode,
		"u2c":       t.opUnicodeToChinese,
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
	cur := t.indent
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
func (t *Tool) Invoke(op string, arg string) domain.Result {
	t.mu.Lock()
	defer t.mu.Unlock()
	h, ok := t.handlers[op]
	if !ok {
		return domain.Result{IsError: true, Notice: "未知操作: " + op}
	}
	return h(arg)
}

// --- 操作回调（锁内执行，可直接读写 t.content/t.indent） ---

func (t *Tool) opSetIndent(arg string) domain.Result {
	t.indent = arg
	if trimEmpty(t.content) {
		return domain.Result{Actions: t.actions()}
	}
	out, err := Pretty(t.content, arg)
	if err != nil {
		return domain.Result{IsError: true, Notice: err.Error(), Actions: t.actions()}
	}
	return domain.Result{Content: out, Notice: "格式化完成", Actions: t.actions()}
}

func (t *Tool) opPretty(arg string) domain.Result {
	return runTransform(t.content, t.indent, "pretty")
}

func (t *Tool) opMinify(arg string) domain.Result {
	return runTransform(t.content, t.indent, "minify")
}

func (t *Tool) opEscape(arg string) domain.Result {
	return runTransform(t.content, t.indent, "escape")
}

func (t *Tool) opUnescape(arg string) domain.Result {
	return runTransform(t.content, t.indent, "unescape")
}

func (t *Tool) opChineseToUnicode(arg string) domain.Result {
	return runTransform(t.content, t.indent, "c2u")
}

func (t *Tool) opUnicodeToChinese(arg string) domain.Result {
	return runTransform(t.content, t.indent, "u2c")
}

// runTransform 按操作名走本包的转换纯函数，结果统一包成 Result
func runTransform(content, indent, op string) domain.Result {
	var out string
	var err error
	switch op {
	case "pretty":
		out, err = Pretty(content, indent)
	case "minify":
		out, err = Minify(content)
	case "escape":
		out, err = Escape(content)
	case "unescape":
		out, err = Unescape(content)
	case "c2u":
		out, err = ChineseToUnicode(content)
	case "u2c":
		out, err = UnicodeToChinese(content)
	default:
		return domain.Result{IsError: true, Notice: "未知操作: " + op}
	}
	if err != nil {
		return domain.Result{IsError: true, Notice: err.Error()}
	}
	notice := map[string]string{
		"pretty": "格式化完成", "minify": "压缩完成", "escape": "转义完成",
		"unescape": "反转义完成", "c2u": "中文 → Unicode 完成", "u2c": "Unicode → 中文 完成",
	}[op]
	return domain.Result{Content: out, Notice: notice}
}

func trimEmpty(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// SaveState/LoadState 设置类状态持久化（内容不入盘）
func (t *Tool) SaveState() map[string]string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return map[string]string{"indent": t.indent}
}

func (t *Tool) LoadState(state map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if v := state["indent"]; v != "" {
		t.indent = v
	}
}
