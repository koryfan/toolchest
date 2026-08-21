// Package config 配置持久化：读写 ~/.config/toolchest/config.yaml。
// 通用容器：顶层按工具 ID 分节，每节的结构（State）由各工具包定义，
// 本包零感知——经泛型 Decode 取强类型，经 SetState 存强类型。
// 内容文本不入盘，只存设置类状态。
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 配置文件读写器（path 构造时定死；states 为工具 ID → State）
type Config struct {
	path   string
	states map[string]any
}

// New 默认配置：$XDG_CONFIG_HOME/toolchest/config.yaml
func New() *Config {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return &Config{path: filepath.Join(dir, "toolchest", "config.yaml"), states: map[string]any{}}
}

// NewAt 指定路径（测试用）
func NewAt(p string) *Config {
	return &Config{path: p, states: map[string]any{}}
}

// Load 读配置到内存容器。文件不存在/损坏返回空容器（首次启动/降级，不阻断）。
func (c *Config) Load() {
	c.states = map[string]any{}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}
	// 顶层泛型解码：得到 工具ID → map[string]any（嵌套结构）。
	// 强类型转换由各工具装配时的 Decode[T] 完成。
	_ = yaml.Unmarshal(data, &c.states)
}

// Decode 把指定工具节的原始数据转到强类型 State（经二次序列化，
// yaml 泛型解码产物 → T 结构体）。不存在/结构不符返回 ok=false，调用方用默认值。
func Decode[T any](c *Config, id string) (T, bool) {
	var zero T
	raw, ok := c.states[id]
	if !ok {
		return zero, false
	}
	b, err := yaml.Marshal(raw)
	if err != nil {
		return zero, false
	}
	var st T
	if err := yaml.Unmarshal(b, &st); err != nil {
		return zero, false
	}
	return st, true
}

// SetState 写入某工具的 State（运行期容器里始终存强类型，Save 输出稳定）
func (c *Config) SetState(id string, s any) {
	c.states[id] = s
}

// Save 全量写回（原子写：临时文件 + rename）
func (c *Config) Save() error {
	data, err := yaml.Marshal(c.states)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}
