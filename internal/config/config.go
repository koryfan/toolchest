// Package config 配置持久化：读写 ~/.config/tool-dev/config.json。
// 结构按工具 ID 分节：{"json": {...}, "time": {...}}。
// 内容文本不入盘，只存设置类状态。
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var (
	mu   sync.Mutex
	path string
)

// SetPath 覆盖配置文件路径（测试用）
func SetPath(p string) { mu.Lock(); path = p; mu.Unlock() }

// Path 返回配置文件路径：$XDG_CONFIG_HOME/tool-dev/config.json
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	if path != "" {
		return path
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "tool-dev", "config.json")
}

// Load 读配置；文件不存在返回空 map（首次启动）
func Load() (map[string]map[string]any, error) {
	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		return map[string]map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		// 配置损坏不阻断启动，当空配置处理
		return map[string]map[string]any{}, nil
	}
	return out, nil
}

// Save 全量写回（原子写：先写临时文件再 rename）
func Save(states map[string]map[string]any) error {
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
