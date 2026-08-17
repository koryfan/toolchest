# 后端驱动工具架构 — 第二步（前端通用渲染器 + 内容镜像 store）实施计划

> **For agentic workers:** 本步 inline 执行（subagent 通道损坏，经用户确认降级）。步骤用 checkbox 跟踪。

**Goal:** 前端改为消费 Go 下发的工具配置（`AppService.Tools()`）：侧栏/工具栏由数据驱动渲染（Button/Menu/Radio 三形态），内容经 zustand store debounce 镜像同步给 Go（`UpdateContent`），操作全部走 `Invoke`。旧的 `tools` 数组注册表、`json:format`/`time:now` 事件总线、本地 handler 分发删除。

**Spec:** `docs/superpowers/specs/2026-08-17-backend-driven-tool-architecture-design.md`（决策 4：内容权威在 Go，方案 A debounce 300ms 镜像同步 + flush 兜底）

## 现状与目标对照

| 项 | 现状 | 目标 |
|---|---|---|
| 侧栏/工具栏 | 前端 `tools` 数组 + 每工具手写 Actions 组件 | Go `Tools()` 下发 ToolDesc 渲染 |
| 操作分发 | window CustomEvent 总线 + 各页面自接 | 通用 `invoke(op, arg)` → `AppService.Invoke` |
| 内容状态 | 各页面 useState 自持 | zustand store：本地即时 + debounce 300ms 推 Go；op 结果 `replaceContent` 即时回写 |
| 前端本地操作 | 清空/复制在页面组件里 | Action.Local 命中前端 handler 注册表（clear/copy 通用） |
| 旧服务调用 | JsonPage/TimePage 直接 import JSONService/TimeService | 全部走 AppService；旧服务 Go 侧删除 |

## Global Constraints

- 样式沿用现有 theme.css 变量与 `.toolbar-actions` 全局类，不引入新样式决策（用户已定稿的 toolbar 纯文字操作、菜单弹出形态不变）
- 交互惯例不变：radio 选中即触发（缩进立即重格式化由 Go 回 Content+Actions 实现）；提示仍走 react-hot-toast 原生样式
- 不做 CodeMirror/树视图的重写——内容区组件（JsonEditor/JsonTree/TimePage 输入区）保留，只改数据接线
- 每个 checkbox 完成后跑 `npm run build`；全部完成后 `go build .` + 重新生成 bindings + `go test ./internal/...`
- **不自动 git commit**，完成后停下等用户 review

## 前置：类型与 API 确认（bindings 已有）

- `frontend/bindings/tool-dev/internal/services/appservice.ts`：`Tools()` / `Invoke(toolID, op, arg)` / `UpdateContent(toolID, content)`
- `frontend/bindings/tool-dev/internal/domain/models.ts`：`ToolDesc{meta, actions}`、`Action{type, id, label, local, items, options, current}`、`ActionType` enum、`Result{content, notice, isError, actions, data}`
- 字段 PascalCase（Go 导出），前端引用照抄

## Task 1: zustand 内容 store

**Files:** `frontend/src/store.ts`（新建）

- [ ] 定义 store：`tools: Record<toolID, {content: string}>`；activeId
- [ ] `setContent(toolID, content)`：本地立即更新 + debounce 300ms 调 `AppService.UpdateContent(toolID, content)`（fire-and-forget，失败静默——镜像允许短暂落后）
- [ ] `replaceContent(toolID, content)`：op 结果回写，立即推 Go（无 debounce）
- [ ] `flush(toolID)`：切工具/关窗口时立即推（清 pending timer）
- [ ] `useToolContent(toolID)` hook：selector 订阅当前工具内容

## Task 2: 通用操作分发 + 本地 handler 注册表

**Files:** `frontend/src/actions.ts`（新建）

- [ ] `invoke(toolID, op, arg?)`：调 `AppService.Invoke` → Result 处理：`content` 非空 → `replaceContent`；`notice` 非空 → toast success/error；`actions` 非空 → 更新该工具的 actions 状态；`data` 由调用方自行取（时间戳页）
- [ ] `localHandlers: Record<string, (ctx) => void>`：`clear`（清 store 内容）、`copy`（读 store 内容 → navigator.clipboard → toast 已复制 N 字符/内容为空提示）。ctx 提供 toolID
- [ ] `runAction(toolID, action, arg?)`：Local 非空 → 查本地表；否则 `invoke`

## Task 3: Toolbar 通用渲染器（三形态）

**Files:** 改 `frontend/src/components/Toolbar.tsx`；`frontend/src/components/ActionRender.tsx`（新建，含 CSS Modules）

- [ ] ActionRender 按 `ActionType` 渲染：
  - `button`：`.toolbar-actions` 纯文字，click → runAction
  - `menu`：纯文字 + 点击弹出菜单（沿用现有 JsonActions 的下拉实现样式：菜单项列表，点击项 → runAction(item.id)）
  - `radio`：纯文字「设置 ▾」+ 弹出单选列表（当前项打勾/高亮），点选项 → runAction(id, option.value)
- [ ] 菜单/单选弹出层复用现有下拉的定位与样式（从 JsonPage.module.css 挪到 ActionRender.module.css，视觉不变）
- [ ] Toolbar 接口改造：`title` + `actions: Action[]`（数据）替代 `actions: ReactNode`（组件）；右侧渲染 ActionRender 列表
- [ ] Radio 选中后收起菜单，选中值来自 Go 下发的 `current`（Result.actions 回传刷新）

## Task 4: 页面接线（数据源切换）

**Files:** 改 `frontend/src/pages/JsonPage.tsx`、`TimePage.tsx`、`App.tsx`；删 `types.ts`、`JsonPage.module.css` 中事件总线相关样式

- [ ] App.tsx：启动 `Tools()` 拉取 ToolDesc[] → store；activeId 切换时 `flush(旧id)`；侧栏/工具栏渲染改用下发数据（icon 名 → URL 映射 `{'json': '/icons/code-{white,gray}.svg', 'time': '/icons/clock-{white,gray}.svg'}`）
- [ ] JsonPage：去掉 JsonActions/事件总线；内容读写走 `useToolContent('json')`（编辑器 onChange → setContent，op 结果经 replaceContent 已入 store，编辑器 value 直接订阅 store）；视图切换（代码/树形）本地 state 保留
- [ ] TimePage：输入行读写走 store（'time'）；解析触发改为 debounce 输入（回车/按钮删掉？→ 保留输入即解析：onChange → setContent + 立即 invoke('parse')，结果 Data 渲染结果卡）——**决策点：解析触发时机按现状保留（点击/回车触发）还是输入即解析，默认保留现状点击触发**；「当前时间戳」按钮由 Actions 下发（button now），结果 Content 回写输入行
- [ ] 清理：删 types.ts、JsonActions/TimeActions 导出、`json:format`/`time:now` CustomEvent 总线代码

## Task 5: 收尾验证 + 文档

- [ ] 全链路构建：`npm run build` + `wails3 generate bindings -clean -ts -i`（无 Go 变更也应无 diff）+ `go build .` + `go test ./internal/... ./pkg/... .`
- [ ] 删除 Go 侧旧服务：`internal/services/json.go`、`time.go`；main.go 去掉 JSONService/TimeService 注册；重新生成 bindings（应只剩 AppService）
- [ ] `./tool-dev` 用户手动冒烟：侧栏/工具栏渲染、格式化/压缩/转义/中U互转、缩进选中即重格式化并落盘（重启后缩进保持）、清空/复制、时间戳解析/当前时间戳、工具切换内容保留（内存）、树形视图、Ctrl+F 搜索
- [ ] 更新 AGENTS.md（架构图/目录/进度）与 README.md（目录树）
- [ ] 停下等用户 review + 「提交」指令

## Self-Review

- spec 决策 4 覆盖：debounce 300ms（Task1）、flush 兜底（Task1/App 切换）、replaceContent 即时（Task1/2）
- spec 调用协议覆盖：Tools/Invoke/UpdateContent 三方法全用上（Task2/4）
- Action 三形态渲染（Task3）对齐 spec 的 Button/Menu/RadioGroup
- TimePage parse 触发保留现状（点击），不给用户引入未确认的交互变更
