package tunneltool

import (
	"encoding/json"
	"fmt"
	"sync"

	"toolchest/internal/domain"
)

// Tool SSH 隧道工具：配置权威在 State（持久化），运行时在 Manager。
// 状态变化经 emit 回调发射 tunnel-update 事件（由装配层注入，
// 本包不依赖 Wails，保持纯逻辑可测）。
type Tool struct {
	mu       sync.Mutex // 保护 state.Tunnels（opSave/opDelete/resolveConfig 并发）
	state    State
	manager  *Manager
	handlers map[string]domain.OpHandler
}

// New 创建（state 由装配层经 config.Decode 注入；emit 为事件发射回调，测试可传 nil）
func New(st State, emit func()) *Tool {
	t := &Tool{state: st, manager: NewManager(emit)}
	t.handlers = map[string]domain.OpHandler{
		"list":   t.opList,
		"start":  t.opStart,
		"stop":   t.opStop,
		"test":   t.opTest,
		"save":   t.opSave,
		"delete": t.opDelete,
	}
	t.manager.ReloadConfigs(st.Tunnels)
	return t
}

func (t *Tool) Meta() domain.Meta {
	return domain.Meta{ID: "tunnel", Title: "SSH 隧道", Icon: "tunnel", ContentType: "tunnel"}
}

// Actions 工具级操作：新增隧道（前端本地打开弹窗）
func (t *Tool) Actions() []domain.Action {
	return []domain.Action{
		domain.NewButton("new", "新增隧道", "tunnelNew"),
	}
}

// SetContent 隧道工具无文本内容（列表页数据走事件/Invoke）
func (t *Tool) SetContent(s string) {}

// State 持久化状态
func (t *Tool) State() any { return t.state }

// RunningInBackground 有隧道在跑（侧栏提醒灯）
func (t *Tool) RunningInBackground() bool { return t.manager.AnyRunning() }

// Invoke 操作分发
func (t *Tool) Invoke(op string, arg string) domain.Result {
	h, ok := t.handlers[op]
	if !ok {
		return domain.Result{IsError: true, Notice: "未知操作: " + op}
	}
	return h(arg)
}

// CloseAll 停止全部隧道（应用退出时由装配层调用）
func (t *Tool) CloseAll() { t.manager.CloseAll() }

// payload tunnel-update 事件载荷（也是 list/start/stop/save/delete 的 Data）
type payload struct {
	Tunnels []TunnelView `json:"tunnels"`
	Running int          `json:"running"`
}

func (t *Tool) data() payload {
	return payload{Tunnels: t.manager.Views(), Running: t.manager.RunningCount()}
}

// Data 全量隧道视图 + 运行数（事件载荷/外部读取用）
func (t *Tool) Data() any { return t.data() }

// --- 操作回调 ---

func (t *Tool) opList(arg string) domain.Result {
	return domain.Result{Data: t.data()}
}

func (t *Tool) opStart(arg string) domain.Result {
	if err := t.manager.Start(arg); err != nil {
		return domain.Result{IsError: true, Notice: "开启失败：" + err.Error(), Data: t.data()}
	}
	return domain.Result{Notice: "隧道已开启：" + arg, Data: t.data()}
}

func (t *Tool) opStop(arg string) domain.Result {
	t.manager.Stop(arg)
	return domain.Result{Notice: "隧道已停止：" + arg, Data: t.data()}
}

// opTest 三步验证。arg = 隧道名（已保存的）或隧道配置 JSON（编辑中的表单值）
func (t *Tool) opTest(arg string) domain.Result {
	cfg, err := t.resolveConfig(arg)
	if err != nil {
		return domain.Result{IsError: true, Notice: err.Error()}
	}
	if msg := cfg.validate(); msg != "" {
		return domain.Result{IsError: true, Notice: msg}
	}
	if err := testTunnel(cfg.normalize()); err != nil {
		return domain.Result{IsError: true, Notice: "测试失败：" + err.Error()}
	}
	return domain.Result{Notice: "隧道配置可用（连接/认证/远程端口均通过）"}
}

// opSave 保存配置。arg = 表单 JSON；Name 已存在 = 编辑，否则新增
func (t *Tool) opSave(arg string) domain.Result {
	t.mu.Lock()
	defer t.mu.Unlock()
	var cfg TunnelConfig
	if err := json.Unmarshal([]byte(arg), &cfg); err != nil {
		return domain.Result{IsError: true, Notice: "表单数据解析失败: " + err.Error()}
	}
	if msg := cfg.validate(); msg != "" {
		return domain.Result{IsError: true, Notice: msg}
	}
	cfg = cfg.normalize()

	replaced := false
	for i, c := range t.state.Tunnels {
		if c.Name == cfg.Name {
			t.state.Tunnels[i] = cfg
			replaced = true
			break
		}
	}
	if !replaced {
		t.state.Tunnels = append(t.state.Tunnels, cfg)
	}
	t.manager.ReloadConfigs(t.state.Tunnels)
	action := "新增"
	if replaced {
		action = "更新"
	}
	return domain.Result{Notice: "隧道配置已" + action + "：" + cfg.Name, Data: t.data()}
}

func (t *Tool) opDelete(arg string) domain.Result {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, c := range t.state.Tunnels {
		if c.Name == arg {
			t.state.Tunnels = append(t.state.Tunnels[:i], t.state.Tunnels[i+1:]...)
			t.manager.Stop(arg) // running 先停再删
			t.manager.ReloadConfigs(t.state.Tunnels)
			return domain.Result{Notice: "隧道已删除：" + arg, Data: t.data()}
		}
	}
	return domain.Result{IsError: true, Notice: "隧道不存在: " + arg}
}

// resolveConfig arg 是隧道名或配置 JSON
func (t *Tool) resolveConfig(arg string) (TunnelConfig, error) {
	if len(arg) > 0 && arg[0] == '{' {
		var cfg TunnelConfig
		if err := json.Unmarshal([]byte(arg), &cfg); err != nil {
			return cfg, fmt.Errorf("表单数据解析失败: %w", err)
		}
		return cfg, nil
	}
	for _, c := range t.state.Tunnels {
		if c.Name == arg {
			return c, nil
		}
	}
	return TunnelConfig{}, fmt.Errorf("隧道不存在: %s", arg)
}
