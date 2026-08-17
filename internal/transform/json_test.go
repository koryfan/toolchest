package transform

import (
	"strings"
	"testing"
)

// doOK 断言操作成功且内容等于 want
func doOK(t *testing.T, name string, input string, want string, op func(*JSONTransformer) error) {
	t.Helper()
	tr := NewJSONTransformer(input)
	if err := op(tr); err != nil {
		t.Fatalf("%s: 操作返回 error = %v", name, err)
	}
	if got := tr.GetString(); got != want {
		t.Errorf("%s: got %q, want %q", name, got, want)
	}
}

func TestDoPretty(t *testing.T) {
	doOK(t, "简单对象", `{"a":1}`, "{\n  \"a\": 1\n}",
		func(tr *JSONTransformer) error { return tr.DoPretty() })
	doOK(t, "嵌套对象", `{"a":{"b":2}}`, "{\n  \"a\": {\n    \"b\": 2\n  }\n}",
		func(tr *JSONTransformer) error { return tr.DoPretty() })
	doOK(t, "数组", `[1,2,3]`, "[\n  1,\n  2,\n  3\n]",
		func(tr *JSONTransformer) error { return tr.DoPretty() })
	doOK(t, "中文不转义", `{"name":"张三"}`, "{\n  \"name\": \"张三\"\n}",
		func(tr *JSONTransformer) error { return tr.DoPretty() })

	// 失败路径
	tr := NewJSONTransformer(`{bad}`)
	if err := tr.DoPretty(); err == nil {
		t.Error("非法 JSON 期望返回 error")
	}
}

func TestDoMinify(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"带缩进对象", "{\n  \"a\": 1\n}", "{\"a\":1}"},
		{"带空格数组", "[ 1 , 2 , 3 ]", "[1,2,3]"},
		{"已压缩", `{"a":1}`, "{\"a\":1}"},
		{"中文不转义", `{"name":"张三"}`, `{"name":"张三"}`},
	}
	for _, c := range cases {
		doOK(t, c.name, c.input, c.want,
			func(tr *JSONTransformer) error { return tr.DoMinify() })
	}

	tr := NewJSONTransformer(`{bad}`)
	if err := tr.DoMinify(); err == nil {
		t.Error("非法 JSON 期望返回 error")
	}
}

func TestDoEscape(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"普通字符串", `hello`, `"hello"`},
		{"带换行", "hello\nworld", `"hello\nworld"`},
		{"带中文", `你好`, `"你好"`},
		{"空字符串", ``, `""`},
	}
	for _, c := range cases {
		doOK(t, c.name, c.input, c.want,
			func(tr *JSONTransformer) error { return tr.DoEscape() })
	}
}

func TestDoUnescape(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"带引号普通", `"hello"`, `hello`},
		{"不带引号自动补", `hello`, `hello`},
		{"带转义引号", `"he said \"hi\""`, `he said "hi"`},
		{"空字符串", ``, ``},
	}
	for _, c := range cases {
		doOK(t, c.name, c.input, c.want,
			func(tr *JSONTransformer) error { return tr.DoUnescape() })
	}

	// 失败路径
	tr := NewJSONTransformer(`"\xyz"`)
	if err := tr.DoUnescape(); err == nil {
		t.Error("非法转义序列期望返回 error")
	}
}

func TestDoChineseToUnicode(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"纯中文", `你好`, `\u4f60\u597d`},
		{"中英混合", `hello 世界`, `hello \u4e16\u754c`},
		{"纯 ASCII", `hello`, `hello`},
		{"空字符串", ``, ``},
	}
	for _, c := range cases {
		doOK(t, c.name, c.input, c.want,
			func(tr *JSONTransformer) error { return tr.DoChineseToUnicode() })
	}
}

func TestDoUnicodeToChinese(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"纯 Unicode 转义", `\u4f60\u597d`, `你好`},
		{"混合文本", `hello \u4e16\u754c`, `hello 世界`},
		{"无转义原样返回", `hello`, `hello`},
		{"非法转义原样保留", `\uzzzz`, `\uzzzz`},
	}
	for _, c := range cases {
		doOK(t, c.name, c.input, c.want,
			func(tr *JSONTransformer) error { return tr.DoUnicodeToChinese() })
	}
}

// 链式操作：格式化 → 转 Unicode → 转回中文，验证多次 Do 调用可行
func TestChainOps(t *testing.T) {
	tr := NewJSONTransformer(`{"name":"张三"}`)
	if err := tr.DoPretty(); err != nil {
		t.Fatalf("DoPretty: %v", err)
	}
	if err := tr.DoChineseToUnicode(); err != nil {
		t.Fatalf("DoChineseToUnicode: %v", err)
	}
	// 此时 name 字段值应该变成 \u5f20\u4e09
	got := tr.GetString()
	if !strings.Contains(got, `\u5f20\u4e09`) {
		t.Errorf("链式操作后期望包含 \\u5f20\\u4e09，实际 %q", got)
	}
	if err := tr.DoUnicodeToChinese(); err != nil {
		t.Fatalf("DoUnicodeToChinese: %v", err)
	}
	if !strings.Contains(tr.GetString(), "张三") {
		t.Error("链式操作 Unicode→中文 还原失败")
	}
}
