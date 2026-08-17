package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	cfg := NewAt(filepath.Join(t.TempDir(), "cfg.yaml"))

	in := States{
		"json": {"indent": "\t", "level": "5"},
		"time": {},
	}
	if err := cfg.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := cfg.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["json"]["indent"] != "\t" || got["json"]["level"] != "5" {
		t.Fatalf("roundtrip 丢失: %#v", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg := NewAt(filepath.Join(t.TempDir(), "nope.yaml"))
	got, err := cfg.Load()
	if err != nil {
		t.Fatalf("文件不存在应返回空配置而非报错: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("期望空配置, 得到 %#v", got)
	}
}

// 文件格式应是可读 YAML（缩进值含空格/Tab 也不能丢）
func TestFileIsReadableYAML(t *testing.T) {
	dir := t.TempDir()
	cfg := NewAt(filepath.Join(dir, "cfg.yaml"))
	if err := cfg.Save(States{"json": {"indent": "  "}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "indent: '  '") {
		t.Fatalf("落盘内容应是人类可读 YAML: %q", raw)
	}
}

// 配置损坏（非法 YAML）不阻断启动
func TestLoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	os.WriteFile(p, []byte(":\n  - [invalid"), 0o644)
	got, err := NewAt(p).Load()
	if err != nil || len(got) != 0 {
		t.Fatalf("损坏配置应返回空配置: %#v, %v", got, err)
	}
}
