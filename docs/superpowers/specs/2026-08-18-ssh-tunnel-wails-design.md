# SSH 隧道工具设计 — Wails 后端驱动版

- 日期：2026-08-18
- 状态：已评审通过（对话中逐节确认）
- 前置：`2026-08-17-backend-driven-tool-architecture-design.md`（本工具是其架构下的第一个「有状态后台任务」工具）
- 取代：`2026-08-14-ssh-tunnel-design.md`（Shirei 时代版本，业务需求沿用、技术载体全部重设计）

## 0. 已确认决策记录

- **状态推送**：Go→JS 事件（`EmitEvent`），前端订阅刷新 + 初始 Invoke('list') 拉全量。Go 是唯一权威，前端只渲染。
- **持久化**：config.yaml 单文件。config 重构为通用容器（内部 `map[string]any` + 泛型 `State[T]` 取用），强类型 State 定义在各工具包内，AppService 保持唯一落盘通道。
- **退出行为**：退出时 CloseAll 全部隧道，但先弹确认框（有隧道运行时）。
- **侧栏提醒灯**：`domain.Tool` 契约新增 `RunningInBackground() bool`，tunnel 状态变化事件驱动前端渲染指示灯。
- **操作界线**：Toolbar = 工具级操作（新增隧道按钮，走 Action 配置）；卡片/弹窗 = 实体级操作（React 页内实现，invoke 调 Go）。

## 1. 目标与非目标

### 目标
- 列表主页：按「已开启 / 未开启」分组，2 列网格卡片，已开启置顶（无则不显示该组）。
- 卡片：隧道名称、连接信息（user@host:port）、认证方式标识、转发信息（local → remote）、操作（开启/停止、测试、修改、删除）。
- 新增/编辑弹窗：完整表单（名称/主机/端口/用户名/认证三式/转发配置），「测试隧道配置」「保存隧道配置」。
- 三种认证：密码 / 代理（agent + 默认密钥）/ 公钥（IdentityFile，支持 ~ 展开）。
- 纯 Go 实现（`golang.org/x/crypto/ssh`），不依赖系统 ssh 二进制。
- 配置持久化到 `~/.config/tool-dev/config.yaml` 的 tunnel 节（结构化，无字符串化）。
- 掉线自动检测：转发 goroutine 检测断开 → 置断开态 + 事件通知前端。
- 退出确认：有运行中隧道时关窗口弹确认，确认后 CloseAll 再退。

### 非目标（v1 简化，沿用旧 spec）
- 反向转发（-R）、动态转发（-D SOCKS）。
- 一条隧道多条转发规则。
- 配置/密码加密存储（本地自用，明文）。
- 自动重连（掉线置断开态，手动重开）。
- known_hosts 校验（InsecureIgnoreHostKey）。

## 2. config 重构（前置独立步骤）

### 现状问题
`Config.States map[string]ToolState`（`ToolState map[string]string`）只能表达扁平字符串设置；隧道是结构化嵌套列表。且方案 B（config 写死各工具字段）违背内聚——已否决。

### 新形态

```go
// internal/config/config.go
type Config struct {
    path   string
    states map[string]any // key = 工具 ID，value = 该工具的 State 结构体（工具包定义）
}

// 泛型取用（Go 方法不能有类型参数，包级函数）
func State[T any](c *Config, id string) (T, bool) {
    v, ok := c.states[id]
    if !ok { var zero T; return zero, false }
    s, ok := v.(T)
    if !ok { var zero T; return zero, false }
    return s, true
}

func (c *Config) SetState(id string, s any) // 写入（内存）
func (c *Config) Save() error               // 全量 yaml 原子写（tmp+rename）
```

- **工具侧**：工具包定义自己的 `State` 结构体（带 yaml tag），`New(state State)` 注入，直接读写字段。**删除 Tool 接口的 `SaveState()/LoadState()` 方法**。
- **装配**：main 里 `jsontool.New(config.State[jsontool.State](cfg, "json"))`；断言失败/缺省（ok=false 或文件结构对不上）→ 工具用默认 State 初始化（文件损坏自然降级）。
- **落盘**：AppService.Invoke 后取 `tool.State()`（Tool 契约新方法，返回 any）与快照对比（reflect.DeepEqual），变化才 SetState + Save。
- **YAML 效果**：`states` 容器直接展开为顶层工具 ID 节，输出如：
  ```yaml
  json:
    indent: "  "
  tunnel:
    tunnels:
      - name: 我的隧道
        host: 1.2.3.4
        port: 22
        ...
  ```
- **yaml 节点解码**：Load 时用 `map[string]yaml.Node` 解顶层，再逐节反序列化到工具 State——但工具 State 类型 config 不知道，所以 Load 解析为 `map[string]any` 需要 yaml 反序列化目标类型……**实际实现**：Load 时 `yaml.Unmarshal(data, &map[string]any{})` 会得到 `map[string]any`（嵌套 map/slice），而 SetState 存的是强类型结构体——类型不一致会导致 Save 时结构漂移。
  **解决**：Load 后每个工具 ID 的 `any` 保持为「二次 marshal→unmarshal 到工具 State」的产物由调用方完成——即 main 装配处：
  ```go
  raw, ok := cfg.RawState("tunnel")          // map[string]any（yaml 泛型解码）
  st := tunneltool.DefaultState()             // 默认
  if ok {
      if b, err := yaml.Marshal(raw); err == nil {
          _ = yaml.Unmarshal(b, &st)          // 经二次序列化转到强类型（失败保默认）
      }
  }
  tunnel := tunneltool.New(st, ...)
  cfg.SetState("tunnel", st)                  // 容器里始终放强类型
  ```
  config 提供 `RawState(id) (any, bool)` 辅助，转换样板收进 main（或 toolset 装配包）。容器在运行期只存强类型 State，Save 输出稳定。

### 迁移
- jsontool：`State{Indent string}`（yaml: `indent`），删 SaveState/LoadState，New(state) 注入。
- timetool：无状态。`State struct{}` 不定义，装配直接 `timetool.New()`；Tool 契约的 State() 返回 nil（AppService 跳过对比）。
- 旧 config.yaml 的 `states: {json: {indent: ...}}` 旧格式不迁移（自用工具，直接删文件重来）。

## 3. tunneltool 包结构

```
internal/tools/tunneltool/
├── tool.go       Tool 实现：Meta/Actions/Invoke/State/RunningInBackground
├── state.go      State + TunnelConfig 结构体（yaml 定义 + 默认值规范化）
├── tunnel.go     单隧道：dial + 认证构造 + 本地监听转发循环（纯逻辑）
└── manager.go    TunnelManager：runtimes map + 状态查询 + 事件发射 + CloseAll
tool_test.go / tunnel_test.go   表驱动 + mock ssh server
```

### 数据模型

```go
type AuthMethod string
const (
    AuthPassword  AuthMethod = "password"
    AuthAgent     AuthMethod = "agent"
    AuthPublicKey AuthMethod = "publickey"
)

type TunnelConfig struct {
    Name       string `yaml:"name"`       // 唯一标识（列表展示 + map key）
    Host       string `yaml:"host"`
    Port       int    `yaml:"port"`       // 默认 22
    Username   string `yaml:"username"`
    AuthMethod AuthMethod `yaml:"authMethod"`
    Password   string `yaml:"password,omitempty"`
    KeyPath    string `yaml:"keyPath,omitempty"`   // 支持 ~ 展开
    LocalHost  string `yaml:"localHost"`  // 默认 127.0.0.1
    LocalPort  int    `yaml:"localPort"`
    RemoteHost string `yaml:"remoteHost"` // 默认 localhost
    RemotePort int    `yaml:"remotePort"`
}

type State struct {
    Tunnels []TunnelConfig `yaml:"tunnels"`
}
```

### 运行时（manager.go）

```go
type Status string
const (
    StatusStopped Status = "stopped"  // 未开启
    StatusRunning Status = "running"  // 已开启
    StatusError   Status = "error"    // 已断开（掉线/错误）
)

type runtime struct {
    cfg      TunnelConfig
    status   Status
    errMsg   string
    client   *ssh.Client
    listener net.Listener
    done     chan struct{}
}

type Manager struct {
    mu       sync.Mutex
    tunnels  []TunnelConfig   // 配置（State 的镜像，权威在 State）
    runtimes map[string]*runtime
    emit     func(list []TunnelView, running bool)  // 事件发射回调（tool 注入）
}
```

- `Start(name)`：dial → listen → accept 循环（每连接两 goroutine io.Copy 桥接）→ running。
- `Stop(name)`：关 listener → 关 client → stopped。
- `Test(cfg)`：dial → client.Dial(remote) → close，不保持。错误分类：网络不通/认证失败/远程端口不通/本地端口占用。
- 断开检测：桥接 goroutine 出错或 accept 异常 → status=error + errMsg → emit 事件。
- `CloseAll()`：遍历 running 全停。
- 所有状态变化后 emit（list 全量 + running 数），事件发射在 goroutine 里（防 UI 锁死）。

### Invoke op 表

| op | arg | 行为 | Result |
|---|---|---|---|
| list | - | 全量隧道视图（配置+状态+错误） | Data.tunnels（数组） |
| start | name | 开启 | notice + Data 全量 |
| stop | name | 停止 | notice + Data 全量 |
| test | name 或表单 JSON | 三步验证 | notice |
| save | 表单 JSON | 校验+规范化+增/改 | notice + Data 全量 |
| delete | name | running 先 stop 再删配置 | notice + Data 全量 |

TunnelView（下发前端的结构，含运行状态）：

```go
type TunnelView struct {
    TunnelConfig
    Status Status `json:"status"`
    ErrMsg string `json:"errMsg"`
}
```

注意：bindings 序列化用 json tag（Wails 走 JSON），TunnelConfig 双 tag（yaml 持久化 + json 下发）。

### Actions（工具级）

```go
func (t *Tool) Actions() []domain.Action {
    return []domain.Action{domain.NewButton("new", "新增隧道", "tunnelNew")}
}
```
`Local = "tunnelNew"`：前端本地 handler 打开新增弹窗（纯 UI 行为，无需 Go）。

## 4. 事件与前端

### 事件
- Go：`application.Get().EmitEvent("tunnel-update", payload)`，payload = `{tunnels: []TunnelView, running: number}`。
- 前端：`Events.On("tunnel-update")`（@wailsio/runtime），更新 zustand store 的 `tunnelList` + `tunnelRunning`。
- 挂载时先 `invoke('tunneltool', 'list')` 拉一次。

### 前端组件

```
frontend/src/pages/TunnelPage.tsx          列表页（分组+2列网格+卡片+新增卡）
frontend/src/pages/TunnelDialog.tsx        新增/编辑弹窗（表单+测试+保存）
frontend/src/components/TunnelCard.tsx     单卡片（或并入 TunnelPage）
```

- contentRegistry 加 `tunnel: TunnelPage`；icon 映射加 tunnel 图标（Feather 风格 SVG，terminal/plug 形象，白/灰两色）。
- 侧栏提醒灯：Sidebar 组件在 icon 上叠加小圆点（`running` 时显示，绿色），数据源 = store.tunnelRunning；通用化设计：ToolDesc 或事件载荷带 `runningInBackground` 标志（Meta 不合适——运行态不是元信息）。**实现**：tunnel-update 事件载荷带 running 数，store 存 `toolRunning: Record<toolID, boolean>`，Sidebar 渲染时查。
- 卡片操作 → `invoke('tunneltool', 'start'/'stop'/'test'/'delete', name)`；test/modify 在卡片/弹窗上，notice 走统一 toast。
- 弹窗表单：受控 React state；「测试隧道配置」用当前表单值构造临时配置 invoke('test', JSON)；「保存」invoke('save', JSON)。
- 删除确认：页内二次确认（按钮变「确认删除？」或小确认条），不引入 Go 侧 dialog。

### 退出拦截
- main.go：`w.SetCloseIntercept(func(){ ... })` → 查 `tunneltool.AnyRunning()`：
  - 无 → 直接退出。
  - 有 → 前端弹确认（EmitEvent("confirm-exit")，前端渲染模态框）→ 用户确认 → Events_On("exit-confirmed") → Go CloseAll + 真正退出。
  - 简化备选（若前端模态链路繁）：Go 侧 `dialog.Message` 系统确认框。**定：先用 Wails Go 侧 dialog**（v3 `application.Dialog`），一条链路最短；前端模态确认后续可换。

## 5. 测试策略

- `config`：泛型 State 取用（存在/缺省/类型不符）、SetState+Save 往返、Load 损坏降级。
- `tunneltool/tunnel_test.go`：mock ssh server（ssh.NewServerConn）验证密码认证/公钥认证/转发连通（本地 listener → mock 目标端口）/Test 错误四分类。
- `tunneltool/tool_test.go`：Invoke 各 op（save 校验/规范化、start/stop 状态流转、delete 联动）。
- manager 并发：并发 start/stop 不死锁（-race）。
- toolset 注册覆盖检查自动纳入（Actions 声明的 new 是 Local，前端自理，不进 Invoke 覆盖检查）。
- 前端不写单测（沿用项目惯例，目检）。

## 6. 实施顺序（写计划时细化）

1. config 重构（泛型容器 + jsontool/timetool 迁移 + AppService 对比逻辑改造 + 测试）
2. tunneltool State/配置结构 + tool.go 骨架（Meta/Actions/Invoke 空转 + 注册 + 前端占位页）
3. tunnel.go 隧道核心 + mock server 单测
4. manager.go + 事件发射 + Invoke 完整实现
5. 前端 TunnelPage 列表 + 卡片
6. TunnelDialog 弹窗（表单/测试/保存）
7. 退出拦截 + 侧栏提醒灯 + icon
8. 全链路构建 + bindings + 冒烟清单
