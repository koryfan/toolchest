// Package config 配置持久化：读写 ~/.config/tool-dev/config.yaml。
// 结构按工具 ID 分节，每节是该工具的设置项。内容文本不入盘，只存设置类状态。
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// States 全量持久化状态：工具 ID → 该工具的设置项
type States map[string]ToolState

// ToolState 单个工具的设置项（键值都是字符串——配置文件天然字符串友好）
type ToolState map[string]string

// Config 配置文件读写器（path 由构造时定死，不再有全局可变状态）
type Config struct {
	path string
}

// New 默认配置：$XDG_CONFIG_HOME/tool-dev/config.yaml
func New() *Config {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return &Config{path: filepath.Join(dir, "tool-dev", "config.yaml")}
}

// NewAt 指定路径（测试用）
func NewAt(p string) *Config { return &Config{path: p} }

// Load 读配置；文件不存在返回空配置（首次启动）
func (c *Config) Load() (States, error) {
	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return States{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := States{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		// 配置损坏不阻断启动，当空配置处理
		return States{}, nil
	}
	return out, nil
}

// Save 全量写回（原子写：先写临时文件再 rename）
func (c *Config) Save(states States) error {
	data, err := yaml.Marshal(states)
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
