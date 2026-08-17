// Package jsontool JSON 工具：配置 + 状态 + 操作分发内聚一处。
// 删一个功能 = 删 Actions 里对应配置 + invoke 的一个 case（+ 对应测试行）。
package jsontool

import (
	"fmt"
	"sync"

	"tool-dev/internal/tools"
	"tool-dev/internal/transform"
)

// indentOptions 缩进选项（value 即 transform.SetIndent 的实参）
var indentOptions = []tools.RadioOption{
	{Value: "  ", Label: "2 空格"},
	{Value: "    ", Label: "4 空格"},
	{Value: "\t", Label: "Tab"},
}

// Tool JSON 工具（内容镜像 + 缩进设置状态）
type Tool struct {
	mu      sync.Mutex
	content string
	indent  string
}

// New 创建（默认 2 空格缩进）
func New() *Tool {
	return &Tool{indent: "  "}
}

func (t *Tool) Meta() tools.Meta {
	return tools.Meta{ID: "json", Title: "JSON 工具", Icon: "json", ContentType: "json"}
}

// Actions 操作区配置：清空/复制（前端自理）+ 格式化 + 设置（缩进，选中即生效）+ 更多
func (t *Tool) Actions() []tools.Action {
	t.mu.Lock()
	cur := t.indent
	t.mu.Unlock()
	return []tools.Action{
		tools.NewButton("clear", "清空", "clear"),
		tools.NewButton("copy", "复制", "copy"),
		tools.NewButton("pretty", "格式化", ""),
		tools.NewRadio("setIndent", "设置", indentOptions, cur),
		tools.NewMenu("more", "更多", []tools.MenuItem{
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
	t.content = s
	t.mu.Unlock()
}

// Invoke 操作分发。op 取自 Actions 配置的 ID；arg 仅 radio 类携带选中值。
func (t *Tool) Invoke(op string, arg string) tools.Result {
	t.mu.Lock()
	content := t.content
	switch op {
	case "setIndent":
		t.indent = arg
	case "pretty", "minify", "escape", "unescape", "c2u", "u2c":
		// 内容不变
	default:
		t.mu.Unlock()
		return tools.Result{IsError: true, Notice: "未知操作: " + op}
	}
	t.mu.Unlock()

	if op == "setIndent" {
		// 选中即生效：空内容只记录设置；有内容立即重新格式化
		if trimEmpty(content) {
			return tools.Result{Actions: t.Actions()}
		}
		out, err := do(content, arg, "pretty")
		if err != nil {
			return tools.Result{IsError: true, Notice: err.Error(), Actions: t.Actions()}
		}
		return tools.Result{Content: out, Notice: "格式化完成", Actions: t.Actions()}
	}

	out, err := do(content, t.indent, op)
	if err != nil {
		return tools.Result{IsError: true, Notice: err.Error()}
	}
	notice := map[string]string{
		"pretty": "格式化完成", "minify": "压缩完成", "escape": "转义完成",
		"unescape": "反转义完成", "c2u": "中文 → Unicode 完成", "u2c": "Unicode → 中文 完成",
	}[op]
	return tools.Result{Content: out, Notice: notice}
}

// do 走 transform 纯逻辑（本层不写算法）
func do(content, indent, op string) (string, error) {
	tr := transform.NewJSONTransformer(content)
	tr.SetIndent(indent)
	var err error
	switch op {
	case "pretty":
		err = tr.DoPretty()
	case "minify":
		err = tr.DoMinify()
	case "escape":
		err = tr.DoEscape()
	case "unescape":
		err = tr.DoUnescape()
	case "c2u":
		err = tr.DoChineseToUnicode()
	case "u2c":
		err = tr.DoUnicodeToChinese()
	default:
		return "", fmt.Errorf("未知操作: %s", op)
	}
	if err != nil {
		return "", err
	}
	return tr.GetString(), nil
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
	if v := state["indent"]; v != "" {
		t.mu.Lock()
		t.indent = v
		t.mu.Unlock()
	}
}
