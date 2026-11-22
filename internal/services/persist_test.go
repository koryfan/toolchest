package services

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"toolchest/internal/config"
	"toolchest/internal/toolset"
)

// TestInvokeEditPersists 编辑已有隧道（原地替换 slice 元素）必须落盘。
// 回归：浅拷贝快照共享 Tunnels 底层数组，opSave 原地改元素时 before
// 快照跟着变，DeepEqual 恒等导致落盘被跳过（编辑任何字段都不生效，
// 重启后回旧值）。
func TestInvokeEditPersists(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	seed := "tunnel:\n    tunnels:\n        - name: n1\n          host: h\n          port: 2222\n          username: u\n          authMethod: publickey\n          keyPath: ~/.ssh/id_rsa\n          localPort: 5401\n          remotePort: 5432\n"
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.NewAt(path)
	cfg.Load()
	toolset.Init(cfg)
	svc := NewAppService()

	form := `{"name":"n1","host":"h","port":2222,"username":"u","authMethod":"publickey","keyPath":"~/.ssh/id_rsa","localHost":"127.0.0.1","localPort":9999,"remoteHost":"localhost","remotePort":5432,"forwardMode":"exec"}`
	r := svc.Invoke("tunnel", "save", form)
	if r.IsError {
		t.Fatalf("save 失败: %s", r.Notice)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "localPort: 9999") {
		t.Error("编辑未落盘：localPort 仍是旧值")
	}
	if !strings.Contains(got, "forwardMode: exec") {
		t.Error("编辑未落盘：forwardMode 缺失")
	}
	if strings.Contains(got, "Tunnels:") || strings.Contains(got, "Indent:") {
		t.Error("落盘 key 未走 yaml tag（大写字段名），重启后 Decode 匹配不上会丢数据")
	}

	// 重启往返：重新 Load + Decode 必须还原隧道
	cfg2 := config.NewAt(path)
	cfg2.Load()
	toolset.Init(cfg2)
	list := toolset.Find("tunnel").Invoke("list", "")
	if list.IsError {
		t.Fatalf("重启后 list 失败: %s", list.Notice)
	}
	if !strings.Contains(fmt.Sprint(list.Data), "ktv-audit-haier") && !strings.Contains(fmt.Sprint(list.Data), "n1") {
		t.Fatal("重启后隧道列表为空")
	}
}
