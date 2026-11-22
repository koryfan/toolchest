// Package domain 工具域核心类型：契约 + 配置/结果结构体。
// 零业务逻辑、零外部依赖，供工具实现与装配层共同引用。
package domain

// Tool 工具契约：Meta 元信息，Actions 操作区配置（含选中态），
// Invoke 执行操作（内容权威在工具自身，直接用持有副本计算），
// SetContent 同步前端镜像内容，State 返回持久化状态结构体
// （类型由工具包定义，config 经泛型取用/装配注入；nil = 无状态），
// RunningInBackground 标识是否有后台任务在跑（侧栏提醒灯）。
type Tool interface {
	Meta() Meta
	Actions() []Action
	Invoke(op string, arg string) Result
	SetContent(s string)
	State() any
	RunningInBackground() bool
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
	Content string   `json:"content"`           // 新内容（空 = 不变）
	Notice  string   `json:"notice"`            // 提示文案（空 = 不提示）
	IsError bool     `json:"isError"`           // true = Notice 为错误
	Actions []Action `json:"actions,omitempty"` // 变化后的配置（nil = 无变化）
	Data    any      `json:"data,omitempty"`    // 附加数据（时间戳解析多字段、隧道列表等，类型由工具定）
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

// OpHandler 工具操作的执行回调：arg 仅 radio 类携带选中值。
// 实现直接访问工具自身状态并在构造的 Result 里给出
// 新内容/提示/变化后的操作区配置（是否持锁由 Dispatch 的 mu 策略决定）。
type OpHandler func(arg string) Result

// ToolDesc Tools() 下发单元：元信息 + 操作区配置 + 后台运行信号
type ToolDesc struct {
	Meta    Meta     `json:"meta"`
	Actions []Action `json:"actions"`
	Running bool     `json:"running"` // 有后台任务在跑（侧栏提醒灯）
}

// DataReporter 有后台状态数据的工具实现：Data 返回 tool-update
// 事件载荷（装配层经此取数据，避免类型断言回挖具体工具类型）
type DataReporter interface {
	Data() any
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
