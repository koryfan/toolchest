# tool-dev 开发约定

自用开发者工具集（Wails v3 + React + TypeScript）。架构为「后端驱动的工具配置」：Go 是配置/状态/内容的唯一权威，前端是通用渲染器。

## 架构

```
main.go                    入口：装配 config/toolset/services，启动应用
internal/
├── domain/                工具域核心类型（零依赖）：Tool 契约、Meta、
│                          Action/ActionType、Result、OpHandler、构造函数
├── tools/                 工具收纳目录（无自身包，子包平铺）
│   ├── jsontool/          JSON 工具：tool.go（状态+配置+handlers 回调表）
│                          + ops.go（转换纯函数，含表驱动单测）
│   └── timetool/          时间戳工具（parse 结果走 Result.Data）
├── toolset/               注册表装配：All/Find/Init/Snapshot/ToolDescs
│                          （独立包避免 jsontool→tools 导入环）
├── services/              Wails 服务层（AppService：Tools/Invoke/UpdateContent）
├── config/                YAML 持久化（~/.config/tool-dev/config.yaml，
│                          结构体化 Config，AppService 注入唯一落盘通道）
└── transform/             纯转换逻辑（现仅 time.go TimeParser）
pkg/strutil/               通用字符串工具（escape/unicode 转换）
frontend/src/
├── store.ts               zustand 内容镜像（debounce 300ms 推 Go）
├── actions.ts             invoke 统一 Result 处理 + 本地 handler 注册表
├── App.tsx                contentRegistry（contentType→组件）+ icon 映射
└── components/            ActionRender（Button/Menu/Radio 三形态）、
                           JsonEditor（CodeMirror）、JsonTree、Toolbar、Sidebar
```

依赖方向：main → toolset/services → tools/* → domain + transform；前端只依赖 bindings 生成的类型与 AppService 三方法。

## 关键约定

- **加新工具**：internal/tools/ 下建子包（实现 domain.Tool，操作用 handlers 回调表）+ toolset 注册一行 + 前端 contentRegistry 加一行映射（仅当有新内容形态）+ icon 映射一行
- **操作配置**：Action 三形态（button/menu/radio），判别字段 struct 序列化；`Local` 非空 = 前端 handler 自理（clear/copy 等），否则 Invoke 走 Go
- **内容权威在 Go**：前端输入经 store debounce 推 UpdateContent；op 结果 Content 经 replaceContent 即时回写；内容不入盘，只有设置类状态（缩进等）落 config.yaml
- **配置删除联动**：Go 侧删 Actions 配置项，前端渲染自动跟随，无前端代码要改
- **包名不用下划线**（jsontool/timetool）；构建只用 `go build .`（`./...` 会扫到模板平台目录）；测试 `go test ./internal/... ./pkg/... .`
- **bindings 再生成**：改 Go 服务/类型后跑 `wails3 generate bindings -clean -ts -i`（必须从仓库根目录跑，且带 -ts -i）；然后 frontend `npm run build`
- **提交纪律**：用户明确说「提交」才 git commit
- 提示 toast 用 react-hot-toast 原生默认样式（showSuccess/showError 纯函数导出，非 React 上下文可直接调）

## 进度

已完成：
- JSON 工具（CodeMirror 代码视图：行号/折叠/缩进参考线/VS Code 风格搜索；树形视图 @uiw/react-json-view；格式化/压缩/转义/中 U 互转，缩进选中即重格式化并持久化）
- 时间戳工具（多格式解析结果、当前时间戳）
- 后端驱动架构（spec 见 docs/superpowers/specs/，三步实施全部完成）

待做：
- SSH 隧道工具（Step 5，架构就绪后按同模式新增）
