# 后端驱动工具架构 — 第一步（Go 侧工具域）实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 `internal/tools` 工具域（Tool 接口 + JSON/时间戳工具内聚实现 + 状态持久化）并把 services 收口为 `Tools()/Invoke()/UpdateContent()` 三个方法，前端暂不动。

**Architecture:** Go 成为配置与状态的唯一权威：工具自己声明 Meta/Actions 配置并在 Invoke 内分发操作；内容镜像由前端 debounce 推给 `UpdateContent`；设置类状态经 `internal/config` 落 `~/.config/tool-dev/config.json`。旧的 JSONService/TimeService 本步保留（前端还在用），第二步删。

**Tech Stack:** Go 1.25 / Wails v3 beta.9（bindings 仅本步重新生成）/ 表驱动单测，无新依赖。

**Spec:** `docs/superpowers/specs/2026-08-17-backend-driven-tool-architecture-design.md`

## Global Constraints

- 旧 `JSONService`/`TimeService` 本步不删、main.go 继续注册（前端尚未迁移，删了前端 build 崩）。
- Action 用带 `Type` 判别字段的单个 struct 序列化（不用 Go interface，Wails bindings 对 interface 生成不可靠）。
- `Result` 含 `Data map[string]string` 承载时间戳解析等多字段结果（spec 补充：`Content` 是单串装不下结构化结果）。
- 内容文本不入盘；落盘只有设置类状态（缩进值）。
- 包名不用下划线：`jsontool`/`timetool`（目录 `json_tool/` 也不需要，直接平铺 `internal/tools/jsontool/`）。
- 纯逻辑全走 `internal/transform`（NewJSONTransformer / NewTimeParser），工具层不重写算法。
- 每个任务：先写失败测试 → 实现 → 测试过 → commit（信息用 Conventional Commits，中文描述）。
- 命令：构建只用 `go build .`（`./...` 会扫到模板 build/ 平台目录报 main undeclared）；测试 `go test ./internal/... ./pkg/... .`。

---

### Task 1: internal/config 配置持久化

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Load() (map[string]map[string]any, error)`、`config.Save(states map[string]map[string]any) error`、`config.Path() string`（内部 `os.UserConfigDir()/tool-dev/config.json`，测试经 `config.SetPath` 覆盖）

- [ ] **Step 1: 写失败测试**

```go
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
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/config/`
Expected: FAIL（SetPath/Save/Load 未定义，编译错误）

- [ ] **Step 3: 实现**

```go
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
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: internal/config 配置持久化（按工具分节 + 原子写）"
```

---

### Task 2: internal/tools 接口与注册表

**Files:**
- Create: `internal/tools/tool.go`
- Test: `internal/tools/registry_test.go`

**Interfaces:**
- Produces（后续任务与 services 依赖的全部类型，签名以此为准）:

```go
type Tool interface {
    Meta() Meta
    Actions() []Action
    Invoke(op string, arg string) Result
    SetContent(s string)
    SaveState() map[string]any
    LoadState(state map[string]any)
}

type Meta struct { ID, Title, Icon, ContentType string }   // json tag: id/title/icon/contentType
type Result struct { ... }                                 // 见实现
type Action struct { ... }                                 // 判别字段 struct
type MenuItem struct { ID, Label string }
type RadioOption struct { Value, Label string }
type ToolDesc struct { Meta Meta; Actions []Action }       // Tools() 下发单元

func All() []Tool                                          // 注册表（单例）
func Find(id string) Tool                                  // 按 ID 查找，无则 nil
func Init(states map[string]map[string]any)                // main 启动时灌入持久化状态
func Snapshot() map[string]map[string]any                  // 全量状态快照（持久化用）
func NewButton(id, label, local string) Action
func NewMenu(id, label string, items []MenuItem) Action
func NewRadio(id, label string, options []RadioOption, current string) Action
```

- [ ] **Step 1: 写失败测试**

```go
package tools

import "testing"

// 全部注册的工具：Actions 里声明的每个非 local 操作，Invoke 必须有实现
// （防字符串分发写错——删了实现忘删配置、或反之，测试直接红）
func TestAllToolsOpsImplemented(t *testing.T) {
	for _, tl := range All() {
		for _, a := range tl.Actions() {
			var opIDs []string
			switch a.Type {
			case "button":
				if a.Local == "" {
					opIDs = append(opIDs, a.ID)
				}
			case "menu":
				for _, it := range a.Items {
					opIDs = append(opIDs, it.ID)
				}
			case "radio":
				opIDs = append(opIDs, a.ID) // radio 选中即 Invoke(a.ID, value)
			}
			for _, op := range opIDs {
				// 用空内容试调用：只求「未知操作」路径不命中，不校验业务结果
				r := tl.Invoke(op, "")
				if r.IsError && r.Notice == "未知操作: "+op {
					t.Errorf("%s/%s: Actions 声明了但 Invoke 未实现", tl.Meta().ID, op)
				}
			}
		}
	}
}

func TestFindAndInit(t *testing.T) {
	if Find("json") == nil || Find("time") == nil {
		t.Fatalf("json/time 工具应已注册")
	}
	if Find("nope") != nil {
		t.Fatalf("未知 ID 应返回 nil")
	}
	// Init 灌状态后 Snapshot 能取回
	Init(map[string]map[string]any{"json": {"indent": "\t"}})
	snap := Snapshot()
	if snap["json"] == nil || snap["time"] == nil {
		t.Fatalf("Snapshot 应覆盖全部注册工具: %#v", snap)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tools/`
Expected: FAIL（编译错误：All/Find/Init/Snapshot 未定义）

- [ ] **Step 3: 实现 tool.go**

```go
// Package tools 工具域：Tool 接口 + 注册表。
// 配置权威在后端：每个工具自己声明标题/图标/操作区配置并分发操作，
// 前端只是通用渲染器。加新工具 = 实现接口 + all 里加一行。
package tools

import "sync"

// Tool 工具契约：Meta 元信息，Actions 操作区配置（含选中态），
// Invoke 执行操作（内容权威在工具自身，直接用持有副本计算），
// SetContent 同步前端镜像内容，Save/LoadState 设置类状态持久化钩子。
type Tool interface {
	Meta() Meta
	Actions() []Action
	Invoke(op string, arg string) Result
	SetContent(s string)
	SaveState() map[string]any
	LoadState(state map[string]any)
}

// Meta 工具元信息（侧栏 + 工具栏渲染依据）
type Meta struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Icon        string `json:"icon"`        // 图标名（"json"/"time"），前端按名查 SVG
	ContentType string `json:"contentType"` // 内容形态，前端 contentRegistry 映射
}

// Result 操作结果：新内容 + 提示 + 变化后的操作区配置一次带回
type Result struct {
	Content string            `json:"content"`           // 新内容（空 = 不变）
	Notice  string            `json:"notice"`            // 提示文案（空 = 不提示）
	IsError bool              `json:"isError"`           // true = Notice 为错误
	Actions []Action          `json:"actions,omitempty"` // 变化后的配置（nil = 无变化）
	Data    map[string]string `json:"data,omitempty"`    // 多字段结果（时间戳解析等）
}

// Action 操作区配置（判别字段 struct，bindings 友好）
type Action struct {
	Type    string        `json:"type"`             // button | menu | radio
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Local   string        `json:"local,omitempty"`  // button：非空 = 前端 handler 自理
	Items   []MenuItem    `json:"items,omitempty"`  // menu
	Options []RadioOption `json:"options,omitempty"` // radio
	Current string        `json:"current,omitempty"` // radio 当前选中值
}

type MenuItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type RadioOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ToolDesc Tools() 下发单元：元信息 + 操作区配置
type ToolDesc struct {
	Meta    Meta     `json:"meta"`
	Actions []Action `json:"actions"`
}

// ---- 构造函数：保持工具侧配置可读 ----

func NewButton(id, label, local string) Action {
	return Action{Type: "button", ID: id, Label: label, Local: local}
}

func NewMenu(id, label string, items []MenuItem) Action {
	return Action{Type: "menu", ID: id, Label: label, Items: items}
}

func NewRadio(id, label string, options []RadioOption, current string) Action {
	return Action{Type: "radio", ID: id, Label: label, Options: options, Current: current}
}

// ---- 注册表（单例指针，状态内聚在各自字段） ----

var (
	mu  sync.Mutex
	all []Tool
)

// all 工具注册表：加新工具 = 这里加一行
var registry = []Tool{
	// jsontool.Register() 在各包 init 前为 nil，见下方 All() 说明
}

// All 返回注册表（Task 3/4 完成后 registry 填入 jsontool/timetool 实例）
func All() []Tool {
	mu.Lock()
	defer mu.Unlock()
	return append([]Tool{}, all...)
}

// Find 按工具 ID 查找
func Find(id string) Tool {
	for _, t := range All() {
		if t.Meta().ID == id {
			return t
		}
	}
	return nil
}

// Init 启动时灌入持久化状态（各工具 LoadState）
func Init(states map[string]map[string]any) {
	for _, t := range All() {
		if s, ok := states[t.Meta().ID]; ok {
			t.LoadState(s)
		}
	}
}

// Snapshot 全量状态快照（持久化用）
func Snapshot() map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, t := range All() {
		out[t.Meta().ID] = t.SaveState()
	}
	return out
}
```

**注意**：`all` 的填充放 Task 3/4（工具实现落地后才能实例化）。本任务先让 `All()` 返回空表，
`TestFindAndInit` 在本任务先注释掉 json/time 断言（或拆到 Task 4 完成后再启用），
`TestAllToolsOpsImplemented` 空表自然通过（红 → 绿路径不破坏）。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/tools/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat: internal/tools 工具域接口与注册表（Tool 契约 + Action 判别配置）"
```

---

### Task 3: jsontool JSON 工具

**Files:**
- Create: `internal/tools/jsontool/tool.go`
- Test: `internal/tools/jsontool/tool_test.go`

**Interfaces:**
- Consumes: Task 2 的 `Tool/Meta/Result/Action/NewButton/NewMenu/NewRadio`；`transform.NewJSONTransformer(input)` + `SetIndent(s)` + `DoPretty/DoMinify/DoEscape/DoUnescape/DoChineseToUnicode/DoUnicodeToChinese` + `GetString()`
- Produces: `jsontool.New() *Tool`（实现 Tool 接口；Meta: ID "json"、Title "JSON 工具"、Icon "json"、ContentType "json"）

- [ ] **Step 1: 写失败测试**

```go
package jsontool

import "testing"

const src = `{"name":"工具","age":3,"list":[1,2]}`

func newTool() *Tool { t := New(); t.SetContent(src); return t }

func TestMeta(t *testing.T) {
	m := New().Meta()
	if m.ID != "json" || m.Title != "JSON 工具" || m.Icon != "json" || m.ContentType != "json" {
		t.Fatalf("Meta 不符: %#v", m)
	}
}

// 操作表：输入 → 期望内容（经 transform，算法本身有单测，这里只验分发与结果写回）
func TestInvokeOps(t *testing.T) {
	tests := []struct {
		name, op, arg, wantSub string // wantSub = 结果应包含的子串
	}{
		{"格式化默认缩进", "pretty", "", "\n  \"name\""},
		{"格式化Tab缩进", "setIndent", "\t", "\n\t\"name\""},
		{"压缩", "minify", "", src},
		{"转义", "escape", "", "\\\"name\\\""},
		{"反转义", "unescape", "", "\"name\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := newTool()
			r := tl.Invoke(tt.op, tt.arg)
			if r.IsError {
				t.Fatalf("%s 报错: %s", tt.op, r.Notice)
			}
			if !contains(r.Content, tt.wantSub) {
				t.Fatalf("%s 结果不含 %q: %q", tt.op, tt.wantSub, r.Content)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestSetIndentReturnsActions(t *testing.T) {
	tl := newTool()
	r := tl.Invoke("setIndent", "\t")
	if r.IsError || r.Actions == nil {
		t.Fatalf("setIndent 应带回新配置: %#v", r)
	}
	// 新配置里设置组的 Current 应是 \t
	for _, a := range r.Actions {
		if a.Type == "radio" && a.ID == "setIndent" && a.Current != "\t" {
			t.Fatalf("radio Current 应为 \\t: %#v", a)
		}
	}
	// 内容同时被重新格式化
	if !contains(r.Content, "\n\t\"name\"") {
		t.Fatalf("setIndent 应立即重新格式化: %q", r.Content)
	}
}

func TestSetContentThenInvoke(t *testing.T) {
	tl := New()
	tl.SetContent("  不合法  ")
	r := tl.Invoke("pretty", "")
	if !r.IsError {
		t.Fatalf("坏 JSON 应报错而非静默")
	}
}

func TestStateRoundtrip(t *testing.T) {
	tl := New()
	tl.Invoke("setIndent", "\t")
	st := tl.SaveState()
	tl2 := New()
	tl2.LoadState(st)
	if tl2.SaveState()["indent"] != "\t" {
		t.Fatalf("状态还原失败: %#v", tl2.SaveState())
	}
	// 还原后的缩进应作用于后续格式化
	tl2.SetContent(src)
	r := tl2.Invoke("pretty", "")
	if !contains(r.Content, "\n\t\"name\"") {
		t.Fatalf("还原后的缩进未生效: %q", r.Content)
	}
}

func TestUnknownOp(t *testing.T) {
	r := New().Invoke("nope", "")
	if !r.IsError || r.Notice != "未知操作: nope" {
		t.Fatalf("未知操作应有明确报错: %#v", r)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tools/jsontool/`
Expected: FAIL（New 未定义）

- [ ] **Step 3: 实现 tool.go**

```go
// Package jsontool JSON 工具：配置 + 状态 + 操作分发内聚一处。
// 删一个功能 = 删 Actions 里对应配置 + invoke 的一个 case（+ 对应测试行）。
package jsontool

import (
	"fmt"
	"sync"

	"tool-dev/internal/tools"
	"tool-dev/internal/transform"
)

// indentOptions 缩进选项（value 即 transform.SetIndent 的实参）
var indentOptions = []tools.RadioOption{
	{Value: "  ", Label: "2 空格"},
	{Value: "    ", Label: "4 空格"},
	{Value: "\t", Label: "Tab"},
}

// Tool JSON 工具（内容镜像 + 缩进设置状态）
type Tool struct {
	mu      sync.Mutex
	content string
	indent  string
}

// New 创建（默认 2 空格缩进）
func New() *Tool {
	return &Tool{indent: "  "}
}

func (t *Tool) Meta() tools.Meta {
	return tools.Meta{ID: "json", Title: "JSON 工具", Icon: "json", ContentType: "json"}
}

// Actions 操作区配置：清空/复制（前端自理）+ 格式化 + 设置（缩进，选中即生效）+ 更多
func (t *Tool) Actions() []tools.Action {
	t.mu.Lock()
	cur := t.indent
	t.mu.Unlock()
	return []tools.Action{
		tools.NewButton("clear", "清空", "clear"),
		tools.NewButton("copy", "复制", "copy"),
		tools.NewButton("pretty", "格式化", ""),
		tools.NewRadio("setIndent", "设置", indentOptions, cur),
		tools.NewMenu("more", "更多", []tools.MenuItem{
			{ID: "minify", Label: "压缩"},
			{ID: "escape", Label: "转义"},
			{ID: "unescape", Label: "反转义"},
			{ID: "c2u", Label: "中文 → Unicode"},
			{ID: "u2c", Label: "Unicode → 中文"},
		}),
	}
}

// SetContent 前端镜像同步（debounce 推送）
func (t *Tool) SetContent(s string) {
	t.mu.Lock()
	t.content = s
	t.mu.Unlock()
}

// Invoke 操作分发。op 取自 Actions 配置的 ID；arg 仅 radio 类携带选中值。
func (t *Tool) Invoke(op string, arg string) tools.Result {
	t.mu.Lock()
	content, indent := t.content, t.indent
	switch op {
	case "setIndent":
		t.indent = arg
		indent = arg
	case "pretty", "minify", "escape", "unescape", "c2u", "u2c":
		// 内容不变
	default:
		t.mu.Unlock()
		return tools.Result{IsError: true, Notice: "未知操作: " + op}
	}
	t.mu.Unlock()

	if op == "setIndent" {
		// 选中即生效：空内容只记录设置；有内容立即重新格式化
		if trimEmpty(content) {
			return tools.Result{Actions: t.Actions()}
		}
		out, err := do(content, indent, "pretty")
		if err != nil {
			return tools.Result{IsError: true, Notice: err.Error(), Actions: t.Actions()}
		}
		return tools.Result{Content: out, Notice: "格式化完成", Actions: t.Actions()}
	}

	out, err := do(content, indent, op)
	if err != nil {
		return tools.Result{IsError: true, Notice: err.Error()}
	}
	notice := map[string]string{
		"pretty": "格式化完成", "minify": "压缩完成", "escape": "转义完成",
		"unescape": "反转义完成", "c2u": "中文 → Unicode 完成", "u2c": "Unicode → 中文 完成",
	}[op]
	return tools.Result{Content: out, Notice: notice}
}

// do 走 transform 纯逻辑（本层不写算法）
func do(content, indent, op string) (string, error) {
	tr := transform.NewJSONTransformer(content)
	tr.SetIndent(indent)
	var err error
	switch op {
	case "pretty":
		err = tr.DoPretty()
	case "minify":
		err = tr.DoMinify()
	case "escape":
		err = tr.DoEscape()
	case "unescape":
		err = tr.DoUnescape()
	case "c2u":
		err = tr.DoChineseToUnicode()
	case "u2c":
		err = tr.DoUnicodeToChinese()
	default:
		return "", fmt.Errorf("未知操作: %s", op)
	}
	if err != nil {
		return "", err
	}
	return tr.GetString(), nil
}

func trimEmpty(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// SaveState/LoadState 设置类状态持久化（内容不入盘）
func (t *Tool) SaveState() map[string]any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return map[string]any{"indent": t.indent}
}

func (t *Tool) LoadState(state map[string]any) {
	if v, ok := state["indent"].(string); ok && v != "" {
		t.mu.Lock()
		t.indent = v
		t.mu.Unlock()
	}
}
```

- [ ] **Step 4: 注册表接线**

`internal/tools/tool.go` 的空 `all` 改为（导入 `tool-dev/internal/tools/jsontool`，注意 tool.go 在 tools 包、jsontool 导入 tools，无环）：

```go
var registryInit = register(jsontool.New())

func register(ts ...Tool) []Tool { return ts }
```

简化写法（直接包级变量初始化顺序无依赖问题）：

```go
var all = []Tool{jsontool.New()}
```

`All()` 已用锁读，直接替换即可。同时启用 Task 2 里暂缓的 `Find("json")` 断言。

- [ ] **Step 5: 跑测试确认通过**

Run: `go test ./internal/tools/... -v`
Expected: PASS（含 Task 2 的 ops 覆盖测试对 json 工具生效）

- [ ] **Step 6: Commit**

```bash
git add internal/tools/
git commit -m "feat: jsontool JSON 工具内聚实现（配置+状态+操作分发）"
```

---

### Task 4: timetool 时间戳工具

**Files:**
- Create: `internal/tools/timetool/tool.go`
- Test: `internal/tools/timetool/tool_test.go`

**Interfaces:**
- Consumes: Task 2 类型；`transform.NewTimeParser()` + `SetInput/Parse/Result() → ParseResult{Valid,LocalTime,UTCTime,RelativeTime,Weekday,UnixSecond,UnixMilli,Error}`
- Produces: `timetool.New() *Tool`（Meta: ID "time"、Title "时间戳转换"、Icon "time"、ContentType "time"）；`Invoke("parse","")` 结果走 `Result.Data`（键 localTime/utcTime/relativeTime/weekday/unixSecond/unixMilli）；`Invoke("now","")` 把内容设为当前秒级时间戳并在 `Content` 带回

- [ ] **Step 1: 写失败测试**

```go
package timetool

import (
	"strings"
	"testing"
)

func TestMeta(t *testing.T) {
	m := New().Meta()
	if m.ID != "time" || m.Title != "时间戳转换" || m.ContentType != "time" {
		t.Fatalf("Meta 不符: %#v", m)
	}
}

func TestParseTimestamp(t *testing.T) {
	tl := New()
	tl.SetContent("1720000000")
	r := tl.Invoke("parse", "")
	if r.IsError {
		t.Fatalf("解析报错: %s", r.Notice)
	}
	if !strings.HasPrefix(r.Data["localTime"], "2024-07-03") {
		t.Fatalf("localTime 不符: %q", r.Data["localTime"])
	}
	if r.Data["unixSecond"] != "1720000000" || r.Data["unixMilli"] != "1720000000000" {
		t.Fatalf("时间戳往返不符: %#v", r.Data)
	}
}

func TestParseInvalid(t *testing.T) {
	tl := New()
	tl.SetContent("abc")
	r := tl.Invoke("parse", "")
	if !r.IsError || r.Notice == "" {
		t.Fatalf("坏输入应报错: %#v", r)
	}
}

func TestNow(t *testing.T) {
	r := New().Invoke("now", "")
	if r.IsError || len(r.Content) != 10 {
		t.Fatalf("now 应返回 10 位秒级时间戳: %#v", r)
	}
}

func TestStateless(t *testing.T) {
	// 无设置类状态：Save 空 map、Load 不炸即可
	tl := New()
	if len(tl.SaveState()) != 0 {
		t.Fatalf("time 工具无持久化状态: %#v", tl.SaveState())
	}
	tl.LoadState(map[string]any{"x": 1})
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tools/timetool/`
Expected: FAIL（New 未定义）

- [ ] **Step 3: 实现 tool.go**

```go
// Package timetool 时间戳工具：输入（秒/毫秒时间戳或日期串）→ 多格式结果。
// 解析结果走 Result.Data（多字段），解析逻辑在 internal/transform。
package timetool

import (
	"strconv"
	"sync"
	"time"

	"tool-dev/internal/tools"
	"tool-dev/internal/transform"
)

// Tool 时间戳工具（内容镜像即输入行；无设置类状态）
type Tool struct {
	mu      sync.Mutex
	content string
}

func New() *Tool { return &Tool{} }

func (t *Tool) Meta() tools.Meta {
	return tools.Meta{ID: "time", Title: "时间戳转换", Icon: "time", ContentType: "time"}
}

// Actions 当前只有一个操作：填充当前时间戳
func (t *Tool) Actions() []tools.Action {
	return []tools.Action{
		tools.NewButton("now", "当前时间戳", ""),
	}
}

func (t *Tool) SetContent(s string) {
	t.mu.Lock()
	t.content = s
	t.mu.Unlock()
}

func (t *Tool) Invoke(op string, arg string) tools.Result {
	switch op {
	case "now":
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		t.mu.Lock()
		t.content = ts
		t.mu.Unlock()
		return tools.Result{Content: ts}
	case "parse":
		t.mu.Lock()
		content := t.content
		t.mu.Unlock()
		if content == "" {
			return tools.Result{IsError: true, Notice: "请输入时间戳或日期"}
		}
		p := transform.NewTimeParser()
		p.SetInput(content)
		p.Parse()
		pr := p.Result()
		if pr.Error != "" {
			return tools.Result{IsError: true, Notice: pr.Error}
		}
		return tools.Result{Data: map[string]string{
			"localTime": pr.LocalTime, "utcTime": pr.UTCTime,
			"relativeTime": pr.RelativeTime, "weekday": pr.Weekday,
			"unixSecond": pr.UnixSecond, "unixMilli": pr.UnixMilli,
		}}
	default:
		return tools.Result{IsError: true, Notice: "未知操作: " + op}
	}
}

func (t *Tool) SaveState() map[string]any { return map[string]any{} }
func (t *Tool) LoadState(map[string]any)  {}
```

- [ ] **Step 4: 注册表接线 + 启用 Find("time") 断言**

`internal/tools/tool.go`：`var all = []Tool{jsontool.New(), timetool.New()}`

- [ ] **Step 5: 跑测试确认通过**

Run: `go test ./internal/tools/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tools/
git commit -m "feat: timetool 时间戳工具内聚实现（parse 走 Data 多字段）"
```

---

### Task 5: services 收口 + main 接线 + bindings 再生成

**Files:**
- Create: `internal/tools/bridge.go`（放 services 侧需的 `ToolDescs()`）
- Modify: `internal/services/app.go`（新文件，装新服务；旧 json.go/time.go 本步不动）
- Modify: `main.go`（启动时 config.Load → tools.Init；注册新服务）

**Interfaces:**
- Consumes: Task 2-4 的 `tools.All/Find/Init/Snapshot`、`config.Load/Save`
- Produces（前端第二步消费，bindings 生成 TS）：`AppService.Tools() []tools.ToolDesc`、`AppService.Invoke(toolID, op, arg string) tools.Result`、`AppService.UpdateContent(toolID, content string)`

- [ ] **Step 1: tools 侧加 ToolDescs**

`internal/tools/bridge.go`：

```go
package tools

// ToolDescs 全量工具描述（services.Tools() 直接下发）
func ToolDescs() []ToolDesc {
	ts := All()
	out := make([]ToolDesc, 0, len(ts))
	for _, t := range ts {
		out = append(out, ToolDesc{Meta: t.Meta(), Actions: t.Actions()})
	}
	return out
}
```

- [ ] **Step 2: services 新服务**

`internal/services/app.go`：

```go
package services

import (
	"tool-dev/internal/config"
	"tool-dev/internal/tools"
)

// AppService 应用级服务：工具配置下发 + 操作转发 + 内容镜像同步。
// 旧 JSONService/TimeService 在前端迁移完成后删除。
type AppService struct{}

// Tools 全量工具描述（元信息 + 操作区配置）
func (s *AppService) Tools() []tools.ToolDesc {
	return tools.ToolDescs()
}

// Invoke 转发操作；状态变化时全量落盘（写失败不阻断操作）
func (s *AppService) Invoke(toolID, op, arg string) tools.Result {
	t := tools.Find(toolID)
	if t == nil {
		return tools.Result{IsError: true, Notice: "未知工具: " + toolID}
	}
	r := t.Invoke(op, arg)
	_ = config.Save(tools.Snapshot())
	return r
}

// UpdateContent 前端 debounce 推送的内容镜像同步
func (s *AppService) UpdateContent(toolID, content string) {
	if t := tools.Find(toolID); t != nil {
		t.SetContent(content)
	}
}
```

- [ ] **Step 3: main.go 接线**

在 `application.New` 前：启动加载持久化状态；Services 列表追加 `&services.AppService{}`（旧的保留）：

```go
// 启动时加载持久化设置（缩进等）
if states, err := config.Load(); err == nil {
	tools.Init(states)
}
```

imports 加 `tool-dev/internal/config`、`tool-dev/internal/tools`。

- [ ] **Step 4: 构建 + 全测 + bindings**

```bash
go build . && go vet ./internal/... ./pkg/... . && go test ./internal/... ./pkg/... .
cd frontend && wails3 generate bindings -clean -ts -i && npm run build
```

Expected: 全部通过；bindings 出现 `appservice.ts`（Tools/Invoke/UpdateContent），旧 jsonservice/timeservice 仍在。

- [ ] **Step 5: 手动冒烟（Go 侧，不起 GUI）**

临时测试（跑完删）：`go test ./internal/tools/... -run TestAll -v` 已覆盖分发；
另用 `go run` 不适用（弹窗），跳过 GUI 验证，留给第二步前端接线后统一验证。

- [ ] **Step 6: Commit**

```bash
git add internal/ main.go frontend/bindings/
git commit -m "feat: services 收口为 Tools/Invoke/UpdateContent（旧服务待前端迁移后删）"
```

---

## Self-Review 结果

- **Spec 覆盖**：spec 第 1 步的四项（tools 接口/两工具实现/config 持久化/services 收口）→ Task 1-5 一一对应；
  spec 补充点 `Result.Data`（时间戳多字段）→ Task 2 类型 + Task 4 实现。
- **占位符**：无 TBD/“稍后实现”；Task 2 的 registry 空表是刻意的接线顺序（Task 3/4 填充），已写明。
- **类型一致性**：`Invoke(op, arg)`、`SetContent(string)`、`SaveState/LoadState` 在 Task 2/3/4/5 间签名一致；
  `Result.Data map[string]string` 的键在 Task 4 定义、前端第二步消费时照抄。
- **本步验收**（spec）：`go build .` + `go test ./internal/... ./pkg/... .` 全绿；前端 build 不受影响（旧服务未删）。
