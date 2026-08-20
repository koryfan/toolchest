# tool-dev 开发约定

自用开发者工具集（Wails v3 + React + TypeScript）。架构为「后端驱动的工具配置」：Go 是配置/状态/内容的唯一权威，前端是通用渲染器。

## 架构

```
main.go                    入口：装配 config/toolset/services，启动应用，退出时 CloseAll
internal/
├── domain/                工具域核心类型（零依赖）：Tool 契约（含 State()/
│                          RunningInBackground()）、Meta、Action/ActionType、
│                          Result（Data any）、OpHandler、构造函数
├── tools/                 工具收纳目录（无自身包，子包平铺）
│   ├── jsontool/          JSON 工具：State 注入 + handlers 回调表
│                          + ops.go（转换纯函数，含表驱动单测）
│   ├── timetool/          时间戳工具（parse 结果走 Result.Data）
│   └── tunneltool/        SSH 隧道：state.go（TunnelConfig/State）+ tunnel.go
│                          （认证三式/转发）+ manager.go（运行时+状态事件）
│                          + tool.go（Invoke: list/start/stop/test/save/delete）
├── toolset/               装配：Init(cfg) 注入各工具 State、events.go
│                          （tunnel-update 事件发射）、lifecycle.go（CloseAll）
├── services/              Wails 服务层（AppService：Tools/Invoke/UpdateContent）
├── config/                YAML 持久化（~/.config/tool-dev/config.yaml）：
│                          通用容器 map[string]any + 泛型 Decode[T] 取强类型，
│                          State 结构体由各工具包自定义（config 零感知）
└── transform/             纯转换逻辑（现仅 time.go TimeParser）
pkg/strutil/               通用字符串工具（escape/unicode 转换）
frontend/src/
├── store.ts               zustand：内容镜像（debounce 300ms 推 Go）
│                          + 隧道数据（tunnel-update 事件）+ 弹窗状态
├── actions.ts             invoke 统一 Result 处理 + 本地 handler 注册表
│                          （clear/copy/tunnelNew）
├── App.tsx                contentRegistry（contentType→组件）+ icon 映射 + 侧栏提醒灯
├── pages/                 JsonPage/TimePage/TunnelPage/TunnelDialog
└── components/            ActionRender（Button/Menu/Radio 三形态）、
                           JsonEditor（CodeMirror）、JsonTree、Toolbar、Sidebar
```

依赖方向：main → toolset/services → tools/* → domain + transform；前端只依赖 bindings 生成的类型与 AppService 三方法 + @wailsio/runtime 事件。

## 常用命令

- 构建：`go build .`（**不要 `./...`**，会扫到模板平台目录报错）
- 测试：`go test ./internal/... ./pkg/... .`
- 前端构建：`cd frontend && npm run build`
- bindings 再生成：`wails3 generate bindings -clean -ts -i`（仓库根目录，改 Go 服务/类型后必跑）
- 安装到本机：`task install`（构建前端 + 二进制 + ~/.local/bin + 桌面图标，无需 sudo）
- 卸载：`task uninstall`
- 开发模式：`wails3 dev` 或 `task dev`

## 关键约定

- **加新工具**：internal/tools/ 下建子包（定义自己的 State + handlers 回调表，实现 domain.Tool）+ toolset.Init 注册一行 + config 无需改（通用容器）+ 前端 contentRegistry 加一行映射（仅当有新内容形态）+ icon 映射一行
- **操作配置**：Action 三形态（button/menu/radio），判别字段 struct 序列化；`Local` 非空 = 前端 handler 自理（clear/copy/tunnelNew 等），否则 Invoke 走 Go
- **内容权威在 Go**：前端输入经 store debounce 推 UpdateContent；op 结果 Content 经 replaceContent 即时回写；内容不入盘，设置类状态（缩进/隧道配置）以强类型 State 落 config.yaml 对应节
- **配置删除联动**：Go 侧删 Actions 配置项，前端渲染自动跟随，无前端代码要改
- **后台状态推送**：有自主变化状态的工具（如隧道掉线）经 Wails 事件（toolset/events.go Emit）推前端，前端 store 订阅刷新 + 侧栏提醒灯（RunningInBackground）
- **Wails v3 beta.9 无窗口关闭拦截**：退出确认弹窗不可实现；退出时 main defer CloseAll 清理隧道资源
- **包名不用下划线**（jsontool/timetool）；构建只用 `go build .`（`./...` 会扫到模板平台目录）；测试 `go test ./internal/... ./pkg/... .`
- **bindings 再生成**：改 Go 服务/类型后跑 `wails3 generate bindings -clean -ts -i`（必须从仓库根目录跑，且带 -ts -i）；然后 frontend `npm run build`
- **提交纪律**：用户明确说「提交」才 git commit
- 提示 toast 用 react-hot-toast 原生默认样式（showSuccess/showError 纯函数导出，非 React 上下文可直接调）

## 进度

已完成：
- JSON 工具（CodeMirror 代码视图：行号/折叠/缩进参考线/VS Code 风格搜索；树形视图 @uiw/react-json-view；格式化/压缩/转义/中 U 互转，缩进选中即重格式化并持久化）
- 时间戳工具（多格式解析结果、当前时间戳）
- 后端驱动架构（spec 见 docs/superpowers/specs/，三步实施全部完成）
- SSH 隧道工具（列表卡片/分组、新增编辑弹窗、认证三式、开启停止测试删除、掉线检测事件推送、侧栏提醒灯、配置落 config.yaml tunnel 节）
- config 泛型容器重构（map[string]string → 强类型 State + config.Decode[T]）

待做：
- 无（Wails v3 支持关闭拦截后可补退出确认弹窗）
