package tunneltool

import (
	"fmt"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Status 隧道运行状态
type Status string

const (
	StatusStopped Status = "stopped" // 未开启
	StatusRunning Status = "running" // 已开启
	StatusError   Status = "error"   // 已断开（掉线/错误）
)

// TunnelView 下发前端的隧道视图（配置 + 运行状态 + 错误信息）
type TunnelView struct {
	TunnelConfig
	Status Status `json:"status"`
	ErrMsg string `json:"errMsg"`
}

// runtime 单条隧道的运行时资源
type runtime struct {
	cfg      TunnelConfig
	status   Status
	errMsg   string
	client   *ssh.Client
	listener net.Listener
	done     chan struct{} // 转发循环退出信号（Stop 时经 listener 关闭间接触发）
}

// Manager 隧道运行时管理：配置镜像 + 运行时资源 + 状态变化通知。
// 配置的权威在 Tool.State（持久化），Manager 持有的是运行副本；
// 保存/删除配置时由 Tool 同步调用 ReloadConfigs。
type Manager struct {
	mu       sync.Mutex
	tunnels  []TunnelConfig
	runtimes map[string]*runtime
	emit     func() // 状态变化通知回调（tool 注入，发射 tunnel-update 事件）
}

// NewManager 构造。emit 在每次状态变化后被调用（可能来自后台 goroutine）。
func NewManager(emit func()) *Manager {
	return &Manager{
		runtimes: map[string]*runtime{},
		emit:     emit,
	}
}

// ReloadConfigs 配置变化时同步运行副本（按 Name 对齐；
// ReloadConfigs 配置变化时同步运行副本（按 Name 对齐；
// 被删的 running 隧道自动停止）。
func (m *Manager) ReloadConfigs(tunnels []TunnelConfig) {
	m.mu.Lock()
	for _, cfg := range tunnels {
		if rt, ok := m.runtimes[cfg.Name]; ok {
			rt.cfg = cfg // 运行中的配置热更新（转发参数下次 Start 生效）
			if rt.status != StatusRunning {
				// 出错/停止的隧道配置已变，旧错误信息失去意义
				rt.status = StatusStopped
				rt.errMsg = ""
			}
		}
	}
	// 已删除的隧道停止
	var names []string
	for name := range m.runtimes {
		found := false
		for _, cfg := range tunnels {
			if cfg.Name == name {
				found = true
				break
			}
		}
		if !found {
			names = append(names, name)
		}
	}
	m.tunnels = append([]TunnelConfig{}, tunnels...)
	m.mu.Unlock()
	for _, name := range names {
		m.Stop(name)
	}
}

// Start 开启隧道：dial → 本地监听 → 转发循环
func (m *Manager) Start(name string) error {
	m.mu.Lock()
	var cfg TunnelConfig
	for _, c := range m.tunnels {
		if c.Name == name {
			cfg = c
			break
		}
	}
	if cfg.Name == "" {
		m.mu.Unlock()
		return fmt.Errorf("隧道不存在: %s", name)
	}
	if rt, ok := m.runtimes[name]; ok && rt.status == StatusRunning {
		m.mu.Unlock()
		return fmt.Errorf("隧道已在运行中")
	}
	m.mu.Unlock()

	// 网络操作在锁外（10s 超时，不阻塞其它操作）
	client, err := dial(cfg)
	if err != nil {
		m.mu.Lock()
		m.runtimes[name] = &runtime{cfg: cfg, status: StatusError, errMsg: err.Error()}
		m.mu.Unlock()
		m.notify()
		return err
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort))
	if err != nil {
		_ = client.Close()
		err = fmt.Errorf("本地端口 %d 监听失败（可能被占用）", cfg.LocalPort)
		m.mu.Lock()
		m.runtimes[name] = &runtime{cfg: cfg, status: StatusError, errMsg: err.Error()}
		m.mu.Unlock()
		m.notify()
		return err
	}

	rt := &runtime{cfg: cfg, status: StatusRunning, client: client, listener: ln, done: make(chan struct{})}
	m.mu.Lock()
	m.runtimes[name] = rt
	m.mu.Unlock()

	go func() {
		// 转发循环退出 = 掉线或停止；根据 client 状态判断是否异常断开
		_ = forward(client, cfg, ln)
		close(rt.done)
		m.mu.Lock()
		if rt.status == StatusRunning { // 非 Stop 触发 = 异常断开
			rt.status = StatusError
			rt.errMsg = "连接已断开（网络故障或服务端关闭）"
			_ = client.Close()
		}
		m.mu.Unlock()
		m.notify()
	}()

	go func() {
		// 连接级掉线检测：空闲隧道（无活动连接）感知不到 SSH 断开，
		// accept 循环也不会退出——经 client.Wait() 兜底
		_ = client.Wait()
		m.mu.Lock()
		if rt.status == StatusRunning {
			rt.status = StatusError
			rt.errMsg = "连接已断开（网络故障或服务端关闭）"
			_ = ln.Close() // 让转发循环一并收尾
		}
		m.mu.Unlock()
		m.notify()
	}()
	m.notify()
	return nil
}

// Stop 停止隧道（不存在/未运行返回 nil，幂等）
func (m *Manager) Stop(name string) {
	m.mu.Lock()
	rt, ok := m.runtimes[name]
	if !ok {
		m.mu.Unlock()
		return
	}
	wasRunning := rt.status == StatusRunning
	rt.status = StatusStopped
	rt.errMsg = ""
	m.mu.Unlock()

	if rt.listener != nil {
		_ = rt.listener.Close() // 触发 accept 循环退出
	}
	if rt.client != nil {
		_ = rt.client.Close()
	}
	if wasRunning && rt.done != nil {
		<-rt.done // 等转发 goroutine 收尾，避免事件乱序
	}
	m.notify()
}

// CloseAll 停止全部隧道（应用退出时调用）
func (m *Manager) CloseAll() {
	m.mu.Lock()
	names := make([]string, 0, len(m.runtimes))
	for name := range m.runtimes {
		names = append(names, name)
	}
	m.mu.Unlock()
	for _, name := range names {
		m.Stop(name)
	}
}

// Views 全量隧道视图（配置 + 状态）
func (m *Manager) Views() []TunnelView {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]TunnelView, 0, len(m.tunnels))
	for _, cfg := range m.tunnels {
		v := TunnelView{TunnelConfig: cfg, Status: StatusStopped}
		if rt, ok := m.runtimes[cfg.Name]; ok {
			v.Status = rt.status
			v.ErrMsg = rt.errMsg
		}
		out = append(out, v)
	}
	return out
}

// AnyRunning 是否有隧道在运行（退出确认/侧栏提醒灯）
func (m *Manager) AnyRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rt := range m.runtimes {
		if rt.status == StatusRunning {
			return true
		}
	}
	return false
}

// RunningCount 运行中隧道数
func (m *Manager) RunningCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, rt := range m.runtimes {
		if rt.status == StatusRunning {
			n++
		}
	}
	return n
}

// notify 状态变化通知（后台 goroutine 也可能调用，emit 需自行保证线程安全）
func (m *Manager) notify() {
	if m.emit != nil {
		m.emit()
	}
}
