package config

import "testing"

func TestSaveLoadRoundtrip(t *testing.T) {
	SetPath(t.TempDir() + "/cfg.json")

	in := map[string]map[string]any{
		"json": {"indent": "\t", "level": float64(5)},
		"time": {},
	}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["json"]["indent"] != "\t" || got["json"]["level"] != float64(5) {
		t.Fatalf("roundtrip 丢失: %#v", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	SetPath(t.TempDir() + "/nope.json")
	got, err := Load()
	if err != nil {
		t.Fatalf("文件不存在应返回空配置而非报错: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("期望空配置, 得到 %#v", got)
	}
}
