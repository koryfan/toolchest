package config

import (
	"os"
	"path/filepath"
	"testing"
)

// 测试用 State 类型（模拟工具包定义的强类型状态）
type jsonState struct {
	Indent string `yaml:"indent"`
}

type tunnelState struct {
	Tunnels []struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	} `yaml:"tunnels"`
}

func TestSaveLoadRoundtrip(t *testing.T) {
	cfg := NewAt(filepath.Join(t.TempDir(), "cfg.yaml"))

	// SetState 存强类型 → Save → Load → Decode 取回
	cfg.SetState("json", jsonState{Indent: "\t"})
	cfg.SetState("tunnel", tunnelState{Tunnels: []struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	}{{Name: "我的隧道", Port: 22}}})
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg2 := NewAt(cfg.path)
	cfg2.Load()
	js, ok := Decode[jsonState](cfg2, "json")
	if !ok || js.Indent != "\t" {
		t.Errorf("json 节往返失败: %#v ok=%v", js, ok)
	}
	tun, ok := Decode[tunnelState](cfg2, "tunnel")
	if !ok || len(tun.Tunnels) != 1 || tun.Tunnels[0].Name != "我的隧道" || tun.Tunnels[0].Port != 22 {
		t.Errorf("tunnel 节往返失败: %#v ok=%v", tun, ok)
	}
}

func TestDecodeMissing(t *testing.T) {
	cfg := NewAt(filepath.Join(t.TempDir(), "cfg.yaml"))
	cfg.Load() // 文件不存在

	if _, ok := Decode[jsonState](cfg, "json"); ok {
		t.Error("缺省节应返回 ok=false")
	}
}

func TestLoadCorrupted(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte("{{{bad yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := NewAt(p)
	cfg.Load() // 不应 panic/报错，按空容器处理
	if _, ok := Decode[jsonState](cfg, "json"); ok {
		t.Error("损坏文件应降级为空容器")
	}
	// 降级后仍可正常 Save
	cfg.SetState("json", jsonState{Indent: "  "})
	if err := cfg.Save(); err != nil {
		t.Fatalf("降级后 Save: %v", err)
	}
}
