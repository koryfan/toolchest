// Package tunneltool SSH 隧道工具：本地端口转发（ssh -L 等价）的
// 配置管理 + 运行时管理 + 事件推送。认证三式（密码/代理/公钥），
// 纯 Go 实现（golang.org/x/crypto/ssh），不依赖系统 ssh 二进制。
package tunneltool

// AuthMethod 认证方式
type AuthMethod string

const (
	AuthPassword  AuthMethod = "password"  // 密码
	AuthAgent     AuthMethod = "agent"     // 代理（ssh-copy-id 后系统默认身份）
	AuthPublicKey AuthMethod = "publickey" // 公钥（指定 IdentityFile）
)

// TunnelConfig 一条隧道的完整配置
// （yaml 用于 config.yaml 持久化，json 用于 Wails bindings 下发前端）
type TunnelConfig struct {
	Name       string     `yaml:"name" json:"name"`
	Host       string     `yaml:"host" json:"host"`
	Port       int        `yaml:"port" json:"port"` // 默认 22
	Username   string     `yaml:"username" json:"username"`
	AuthMethod AuthMethod `yaml:"authMethod" json:"authMethod"`
	Password   string     `yaml:"password,omitempty" json:"password,omitempty"`
	KeyPath    string     `yaml:"keyPath,omitempty" json:"keyPath,omitempty"` // 支持 ~ 展开
	LocalHost  string     `yaml:"localHost" json:"localHost"`                 // 默认 127.0.0.1
	LocalPort  int        `yaml:"localPort" json:"localPort"`
	RemoteHost string     `yaml:"remoteHost" json:"remoteHost"` // 默认 localhost
	RemotePort int        `yaml:"remotePort" json:"remotePort"`
}

// State 持久化状态（config.yaml 的 tunnel 节）
type State struct {
	Tunnels []TunnelConfig `yaml:"tunnels"`
}

// DefaultState 空配置
func DefaultState() State { return State{} }

// normalize 规范化默认值（保存前调用）
func (c TunnelConfig) normalize() TunnelConfig {
	if c.Port == 0 {
		c.Port = 22
	}
	if c.LocalHost == "" {
		c.LocalHost = "127.0.0.1"
	}
	if c.RemoteHost == "" {
		c.RemoteHost = "localhost"
	}
	return c
}

// validate 校验必填项（返回首个缺失项的中文说明，空 = 通过）
func (c TunnelConfig) validate() string {
	switch {
	case c.Name == "":
		return "隧道名称不能为空"
	case c.Host == "":
		return "主机不能为空"
	case c.Username == "":
		return "用户名不能为空"
	case c.LocalPort == 0:
		return "本地端口不能为空"
	case c.RemotePort == 0:
		return "远程端口不能为空"
	case c.AuthMethod == AuthPassword && c.Password == "":
		return "密码认证需要填写密码"
	case c.AuthMethod == AuthPublicKey && c.KeyPath == "":
		return "公钥认证需要填写密钥路径"
	}
	return ""
}
