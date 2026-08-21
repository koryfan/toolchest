# 后端驱动的工具架构重构 — 设计文档

日期：2026-08-17
状态：待用户评审

## 背景与目标

当前架构分散：一个工具的实现散在 5 处（Go services、Go transform、前端页面、前端 Actions 组件、App 注册表），
改一个选项要跨两端多处。用户期望：

1. **Go 后端掌握状态与配置权威**：操作区有哪些按钮/菜单/设置项全部由 Go 配置定义，
   删掉 Go 配置里的一项，前端自动跟着变化。
2. **内聚**：一个工具的所有后端知识收口在一个目录；新增工具成本最小化。
3. **状态持久化**：设置类状态（缩进、展开层级等）由 Go 持有并落盘，刷新不丢。

明确不做（YAGNI）：

- 内容区（CodeMirror 编辑器、树视图、时间戳表单）不配置化——强交互 UI 保留为前端组件，Go 只出 `contentType` 名字。
- 内容权威在 Go（同步策略 A，见决策 4）：前端是纯展示镜像，打字本地回显，debounce 推送。

## 总体架构

```
internal/tools/                # 工具域：接口 + 注册表（唯一注册点）
  tool.go                      # Tool 接口 + Action 配置类型 + All 注册表
  jsontool/tool.go             # JSON 工具：Meta + Actions 配置 + Invoke 分发 + 状态（缩进/展开层级）
  timetool/tool.go             # 时间戳工具：同构
internal/config/               # 配置持久化：~/.config/toolchest/config.json 读写
internal/transform/            # 纯函数库保留不动（jsontool 调用）
internal/services/             # 桥接层收口：Tools() / Invoke() / UpdateContent() 三个方法
frontend/src/
  store.ts                     # zustand 全局内容镜像：setContent(本地+debounce 推 Go)/replaceContent/flush
  App.tsx                      # 骨架：拉 Tools() 渲染侧栏/工具栏，内容区查 contentRegistry
  components/Toolbar.tsx       # 通用渲染器：吃 Action 配置，渲染按钮/菜单/RadioGroup
  handlers.ts                  # 前端 local handler 注册表 {clear, copy, ...}
  contentRegistry.ts           # contentType → React 组件映射（唯一前端映射点）
  components/JsonEditor.tsx    # 内容组件保留
  components/JsonTree.tsx      # 内容组件保留
  pages/JsonPage.tsx           # 只剩内容区 + Invoke 结果写回
  pages/TimePage.tsx           # 同上
```

依赖方向：`main → services → tools → transform / config`，无循环。

## 核心接口（Go）

```go
// internal/tools/tool.go
type Tool interface {
    Meta() Meta  // ID / 标题 / 图标名 / contentType
    Actions() []Action  // 操作区配置（含当前选中态）
    Invoke(op string, arg string) Result  // 执行操作（内容权威在 Go，直接用自身副本）
    SetContent(string)                    // 前端 debounce 推送的内容镜像同步
}

type Meta struct {
    ID, Title, Icon, ContentType string  // Icon 是名字（"json"/"time"），前端按名查 SVG
}

// Result：操作结果 + 变化后的配置一次带回，前端不用拉两次
type Result struct {
    Content string   // 新内容（空 = 不变）
    Notice  string   // 成功提示文案（空 = 不提示）
    IsError bool     // true = Notice 是错误提示
    Actions []Action // 变化后的操作区配置（nil = 无变化）
}
```

### Action 配置类型（三种形态）

```go
type Action interface{ action() }

type Button struct {          // 工具栏纯文字按钮
    ID, Label string
    Local   string            // 空 = Invoke("toolID", ID, arg)；非空 = 前端 handler 注册表函数名（如 "clear"/"copy"）
}

type Menu struct {            // 「更多」式下拉菜单
    ID, Label string
    Items  []MenuItem         // MenuItem{ID, Label}
}

type RadioGroup struct {      // 「设置」式选项组（带选中态，选中即生效）
    ID, Label string
    Options []RadioOption     // RadioOption{Value, Label}
    Current string            // 当前选中值
}
```

序列化不用 Go interface（Wails bindings 对 interface 的类型生成不可靠），而是判别字段 struct：

```go
type Action struct {
    Type   string      // "button" | "menu" | "radio"
    ID, Label string
    Local  string      // button 专用
    Items  []MenuItem  // menu 专用
    Options []RadioOption // radio 专用
    Current string     // radio 专用
}
```

Go 侧工具内部仍可用独立构造函数（`NewButton(...)`/`NewMenu(...)`）生成 Action，保持配置可读性。

### 状态与持久化

- 各工具结构体持有自己的状态字段（单例指针，注册表 `All` 里持有）。
- `internal/config/`：启动时读 `~/.config/toolchest/config.json`，按工具 ID 分节；
  工具实现 `LoadState(map[string]any) / SaveState() map[string]any` 两个钩子（接口内聚或独立小接口，实施时定）。
- Invoke 改变状态时同步写盘（写失败不阻断，toast 提示）。
- 内容文本不入盘（重启即空，同现状）；落盘的只是设置类状态（缩进、展开层级等）。

### 调用协议（前端视角）

```
启动 → services.Tools() → [{meta, actions}] → 渲染侧栏 + 当前工具的工具栏
编辑内容    → store.setContent（本地立即回显）→ debounce 300ms → services.UpdateContent(toolID, content)
点击按钮(local="")     → services.Invoke(toolID, opID, arg) → Result → store.replaceContent + 应用 notice/actions
点击按钮(local="clear")→ handlers.clear() → 前端自理（改内容也走 store.setContent）
RadioGroup 选中        → services.Invoke(toolID, opID, value) → 新配置 + 新内容一次带回
切换工具/关窗 → store.flush(toolID) 立即推送未同步尾巴
```

Tool.Invoke 签名去掉 input 参数（原设计 `Invoke(op, input, arg)` → `Invoke(op, arg)`）：
内容权威在 Go，操作直接用 Go 持有的副本计算。

## 关键决策记录

1. **配置全在 Go，实现可标注 local/backend**：Go 是唯一配置权威（删选项只动 Go 一个文件）；
   清空/复制等纯 UI 操作标 `Local` 由前端 handler 处理，不绕 IPC。
2. **侧栏工具列表也由 Go 下发**：`Meta()` 全量走 `Tools()`；加工具/改标题/调顺序只动 Go；
   前端只剩 `contentRegistry`（contentType → 组件）一个纯映射，内容形态不新增时零改动。
3. **内容区不配置化**：CodeMirror/树/表单是 UI 形态不是工具业务知识，保留前端组件。
4. **内容权威在 Go，前端是纯展示镜像（方案 A）**：打字本地立即回显（零延迟），debounce 300ms 推
   `UpdateContent`；操作结果 replaceContent 立即同步；切换工具/关窗 flush 兜底。丢失窗口恒定 ~300ms，
   自用可接受。否掉的备选：B 每键同步等确认（打字有可感知延迟）；C 操作时才传（Go 平时持旧副本，
   「掌握状态」名不副实 + 操作两次 IPC 有时序竞争 + 丢失窗口更大）。前端配 zustand 全局 store
   （~1KB，组件零重复同步逻辑，只订阅自己工具的内容）。
5. **持久化这次一起做**：趁状态收口一并落盘，SSH 隧道的连接配置之后直接复用。

## 现有功能的对齐清单（不退化）

- JSON：清空/复制/格式化按钮、设置（缩进 2/4/Tab，选中立即重新格式化）、更多（压缩/转义/反转义/中→U/U→中）、
  代码视图（行号/折叠/参考线/搜索）、树形视图（@uiw/react-json-view，默认展开 5 层）、复制 toast。
- 时间戳：输入解析（秒/毫秒/日期串）、结果行点击复制、工具栏「当前时间戳」。
- 新增：缩进/展开层级状态持久化，重启保留。

## 测试策略

- Go 表驱动单测（internal/tools 各工具）：
  - 配置里声明的每个非 local op 都有 Invoke 实现（防字符串分发写错）。
  - Invoke 分发正确：各 op 的输入→输出。
  - RadioGroup 选中返回新配置 + 新内容（缩进立即格式化）。
  - 配置存取：SaveState→LoadState 状态还原。
  - SetContent 后 Invoke 用新副本计算（镜像同步生效）。
- 前端：build 通过 + 手动验证清单（上节功能对齐清单逐项）。

## 分步实施（每步独立可验证，做完停下等 review）

1. **Go 侧工具域**：`internal/tools/` 接口 + Action 类型 + jsontool/timetool 实现（含状态字段）+
   `internal/config/` 持久化 + services 收口成 `Tools()/Invoke()/UpdateContent()` + 单测。前端未动，Go 先立稳。
2. **前端通用渲染器 + 全局 store**：zustand store（setContent/replaceContent/flush + debounce 推送）、
   Toolbar 吃 Action 配置（三形态 + local handler 注册表）、App 拉 `Tools()` 渲染侧栏、内容组件接 store、
   删各页面旧 Actions 组件，bindings 重新生成。
3. **状态接线与清理**：Invoke 去掉 input 参数改为 Go 副本计算、缩进选中即生效链路换到 Invoke、
   展开层级进设置、删旧事件总线（json:format/time:now）、切换工具/关窗 flush、持久化联调、AGENTS.md/README 更新。

每步验收：`npm run build` + `go build .` + `go test ./internal/... ./pkg/... .` 全绿 + 用户跑 `./toolchest` 按清单验证。
