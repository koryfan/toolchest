# tool-dev

自用开发者工具集。Wails v3 + Go + React + TypeScript，后端驱动架构（Go 持有配置/状态/内容权威，前端为通用渲染器）。

## 运行

```
# 开发
wails3 dev

# 构建前端 + 二进制
cd frontend && npm install && npm run build && cd ..
go build -o tool-dev .
./tool-dev
```

改了 Go 服务/类型后重新生成 bindings（须在仓库根目录）：

```
wails3 generate bindings -clean -ts -i
```

## 工具

- **JSON 工具**：CodeMirror 代码视图（行号/折叠/缩进参考线/搜索）、树形视图、格式化/压缩/转义/反转义/中文↔Unicode；缩进设置即时生效并持久化
- **时间戳转换**：秒/毫秒时间戳与日期字符串互转，多格式结果点击复制
- **SSH 隧道**：本地端口转发（`ssh -L` 等价）管理——卡片列表、三种认证（密码/代理/公钥）、测试/开启/停止、掉线自动检测（事件推送 + 侧栏提醒灯）；纯 Go 实现，不依赖系统 ssh

## 目录

```
main.go                  入口（装配依赖 + 启动）
internal/
├── domain/              工具域核心类型（Tool 契约、Action、Result 等）
├── tools/               工具实现（jsontool/、timetool/、tunneltool/）
├── toolset/             工具注册表
├── services/            Wails 服务层（AppService）
├── config/              配置持久化（通用容器 + 泛型取用）
└── transform/           纯转换逻辑（TimeParser）
pkg/strutil/             通用字符串工具
frontend/                React + TypeScript 前端（Vite）
docs/                    文档（知识库踩坑记录、架构 spec 与计划）
```

架构决策与实施计划见 `docs/superpowers/`；开发约定见 `AGENTS.md`。
