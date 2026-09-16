package uuidtool

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestMeta(t *testing.T) {
	m := New(DefaultState()).Meta()
	if m.ID != "uuid" || m.Title != "UUID 生成" || m.ContentType != "uuid" {
		t.Fatalf("Meta 不符: %#v", m)
	}
}

func TestGenV4(t *testing.T) {
	tool := New(State{Version: "v4", Count: 1})
	r := tool.Invoke("gen", "")
	if r.IsError {
		t.Fatalf("生成报错: %s", r.Notice)
	}
	if !uuidPattern.MatchString(r.Content) {
		t.Fatalf("V4 格式不符: %q", r.Content)
	}
	if u, err := uuid.Parse(r.Content); err != nil || u.Version() != 4 {
		t.Fatalf("应为 V4: %q %v", r.Content, err)
	}
	if want := "已生成 1 个 UUID（V4）"; r.Notice != want {
		t.Fatalf("提示文案不符: %q, want %q", r.Notice, want)
	}
}

func TestGenV7(t *testing.T) {
	tool := New(State{Version: "v7", Count: 3})
	r := tool.Invoke("gen", "")
	if r.IsError {
		t.Fatalf("生成报错: %s", r.Notice)
	}
	lines := strings.Split(r.Content, "\n")
	if len(lines) != 3 {
		t.Fatalf("应生成 3 个: %q", r.Content)
	}
	for _, s := range lines {
		u, err := uuid.Parse(s)
		if err != nil || u.Version() != 7 {
			t.Fatalf("应为 V7: %q %v", s, err)
		}
	}
	// V7 时间有序：首末时间戳（毫秒）单调不减
	first, _ := uuid.Parse(lines[0])
	last, _ := uuid.Parse(lines[len(lines)-1])
	if first.Time() > last.Time() {
		t.Fatalf("V7 应时间有序: %q", lines)
	}
}

func TestSetVersionAndCount(t *testing.T) {
	tool := New(DefaultState())
	r := tool.Invoke("setVersion", "v4")
	if r.IsError {
		t.Fatalf("切 V4 报错: %s", r.Notice)
	}
	if got := tool.State().(State).Version; got != "v4" {
		t.Fatalf("版本未持久选中: %q", got)
	}
	r = tool.Invoke("setCount", "10")
	if r.IsError {
		t.Fatalf("切数量报错: %s", r.Notice)
	}
	if n := len(strings.Split(r.Content, "\n")); n != 10 {
		t.Fatalf("应生成 10 个，实际 %d", n)
	}
	// 结果带回刷新后的操作区配置（标签/选中跟随）
	if len(r.Actions) == 0 {
		t.Fatal("切换后应带回 Actions")
	}
}

func TestInvalidArgs(t *testing.T) {
	tool := New(DefaultState())
	if r := tool.Invoke("setVersion", "v6"); !r.IsError {
		t.Fatalf("非法版本应报错: %#v", r)
	}
	if r := tool.Invoke("setCount", "11"); !r.IsError {
		t.Fatalf("超上限应报错: %#v", r)
	}
	if r := tool.Invoke("setCount", "abc"); !r.IsError {
		t.Fatalf("非数字应报错: %#v", r)
	}
	if r := tool.Invoke("nope", ""); !r.IsError {
		t.Fatalf("未知操作应报错: %#v", r)
	}
}

func TestStateNormalize(t *testing.T) {
	// 配置损坏/越界：版本回 v7，数量夹回 1..10
	tool := New(State{Version: "v6", Count: 99})
	st := tool.State().(State)
	if st.Version != "v7" || st.Count != MaxCount {
		t.Fatalf("状态未校正: %#v", st)
	}
	tool2 := New(State{Version: "v7", Count: 0})
	if st2 := tool2.State().(State); st2.Count != 1 {
		t.Fatalf("数量下界未校正: %#v", st2)
	}
}
