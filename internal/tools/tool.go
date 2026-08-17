// Package tools 工具域：Tool 契约 + Action/Result 等类型。
// 配置权威在后端：每个工具自己声明标题/图标/操作区配置并分发操作，
// 前端只是通用渲染器。注册表在 internal/toolset（避免实现包反向依赖本包成环）。
package tools


// Tool 工具契约：Meta 元信息，Actions 操作区配置（含选中态），
// Invoke 执行操作（内容权威在工具自身，直接用持有副本计算），
// SetContent 同步前端镜像内容，Save/LoadState 设置类状态持久化钩子。
type Tool interface {
	Meta() Meta
	Actions() []Action
	Invoke(op string, arg string) Result
	SetContent(s string)
	SaveState() map[string]any
	LoadState(state map[string]any)
}

// Meta 工具元信息（侧栏 + 工具栏渲染依据）
type Meta struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Icon        string `json:"icon"`        // 图标名（"json"/"time"），前端按名查 SVG
	ContentType string `json:"contentType"` // 内容形态，前端 contentRegistry 映射
}

// Result 操作结果：新内容 + 提示 + 变化后的操作区配置一次带回
type Result struct {
	Content string            `json:"content"`           // 新内容（空 = 不变）
	Notice  string            `json:"notice"`            // 提示文案（空 = 不提示）
	IsError bool              `json:"isError"`           // true = Notice 为错误
	Actions []Action          `json:"actions,omitempty"` // 变化后的配置（nil = 无变化）
	Data    map[string]string `json:"data,omitempty"`    // 多字段结果（时间戳解析等）
}

// Action 操作区配置（判别字段 struct，bindings 友好）
type Action struct {
	Type    string        `json:"type"` // button | menu | radio
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Local   string        `json:"local,omitempty"`   // button：非空 = 前端 handler 自理
	Items   []MenuItem    `json:"items,omitempty"`   // menu
	Options []RadioOption `json:"options,omitempty"` // radio
	Current string        `json:"current,omitempty"` // radio 当前选中值
}

type MenuItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type RadioOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ToolDesc Tools() 下发单元：元信息 + 操作区配置
type ToolDesc struct {
	Meta    Meta     `json:"meta"`
	Actions []Action `json:"actions"`
}

// ---- 构造函数：保持工具侧配置可读 ----

func NewButton(id, label, local string) Action {
	return Action{Type: "button", ID: id, Label: label, Local: local}
}

func NewMenu(id, label string, items []MenuItem) Action {
	return Action{Type: "menu", ID: id, Label: label, Items: items}
}

func NewRadio(id, label string, options []RadioOption, current string) Action {
	return Action{Type: "radio", ID: id, Label: label, Options: options, Current: current}
}
