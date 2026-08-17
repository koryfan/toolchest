// Package domain 工具域核心类型：契约 + 配置/结果结构体。
// 零业务逻辑、零外部依赖，供工具实现与装配层共同引用。
package domain

// Tool 工具契约：Meta 元信息，Actions 操作区配置（含选中态），
// Invoke 执行操作（内容权威在工具自身，直接用持有副本计算），
// SetContent 同步前端镜像内容，Save/LoadState 设置类状态持久化钩子。
type Tool interface {
	Meta() Meta
	Actions() []Action
	Invoke(op string, arg string) Result
	SetContent(s string)
	SaveState() map[string]string
	LoadState(state map[string]string)
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

// ActionType 操作区配置形态（判别字段，bindings 友好）
type ActionType string

const (
	ActionButton ActionType = "button" // 纯按钮（Local 非空 = 前端 handler 自理）
	ActionMenu   ActionType = "menu"   // 下拉菜单（Items 逐项 Invoke）
	ActionRadio  ActionType = "radio"  // 单选组（选中即 Invoke 自身 ID + value）
)

// Action 操作区配置
type Action struct {
	Type    ActionType    `json:"type"`
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Local   string        `json:"local,omitempty"`   // button：非空 = 前端 handler 名
	Items   []MenuItem    `json:"items,omitempty"`   // menu
	Options []RadioOption `json:"options,omitempty"` // radio
	Current string        `json:"current,omitempty"` // radio 当前选中值
}

// MenuItem 菜单项（点击即 Invoke(item.ID)）
type MenuItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RadioOption 单选项（Value 即 Invoke 的 arg）
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

// NewButton 按钮；local 非空表示前端 handler 自理（如清空/复制）
func NewButton(id, label, local string) Action {
	return Action{Type: ActionButton, ID: id, Label: label, Local: local}
}

// NewMenu 下拉菜单
func NewMenu(id, label string, items []MenuItem) Action {
	return Action{Type: ActionMenu, ID: id, Label: label, Items: items}
}

// NewRadio 单选组；current 为当前选中值
func NewRadio(id, label string, options []RadioOption, current string) Action {
	return Action{Type: ActionRadio, ID: id, Label: label, Options: options, Current: current}
}
