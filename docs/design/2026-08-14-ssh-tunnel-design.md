# SSH 端口转发隧道工具 — 设计文档

- 日期：2026-08-14
- 状态：待评审
- 范围：在 toolchest 应用中新增「SSH 隧道」工具（侧栏第三项），支持以列表方式管理本地端口转发（`ssh -L` 等价）隧道，支持密码 / 代理 / 公钥三种认证，配置持久化到 `~/.config/toolchest/settings.jsonc`。

## 1. 目标与非目标

### 目标
- 列表主页：按「已开启 / 未开启」分组，2 列网格卡片排版，已开启置顶（无则不显示该组）。
- 卡片展示隧道名称、连接信息、认证方式标识、端口转发信息。
- 卡片操作：开启/停止、测试、修改、更多 → 删除。
- 新增/编辑弹窗：主机、端口、用户名、认证方式（密码/代理/公钥）、本地与远程端口转发配置，提供「测试隧道配置」「保存隧道配置」。
- 配置读写 `~/.config/toolchest/settings.jsonc`（JSON with Comments）。
- 用 `golang.org/x/crypto/ssh` 纯 Go 实现 SSH 连接与转发，不依赖系统 `ssh` 二进制。

### 非目标
- 反向转发（`-R`）、动态转发（`-D` SOCKS）—— 不考虑。
- 一条隧道多条转发规则 —— 不支持，需要多条就配多个隧道。
- 配置加密 / 密码加密存储 —— 明文存储（本地自用工具）。
- SSH agent 转发、端口自动重连 —— v1 不做（掉线后置为已断开，由用户手动重新开启）。

## 2. 认证方式定义

下拉框（实际用 Shirei `SegmentedControl` 实现，比下拉框更紧凑）三选一：

| 选项 | 含义 | 实现方式 |
|------|------|----------|
| **密码** | SSH 远程服务器登录密码 | `ssh.Password(password)` |
| **代理** | `ssh-copy-id` 后系统默认身份，无需指定任何密钥/密码 | 优先连本机 `ssh-agent`（`SSH_AUTH_SOCK`，用 agent 里已加载的密钥）；无 agent 则尝试默认密钥文件 `~/.ssh/id_rsa` / `id_ed25519` |
| **公钥** | 指定 IdentityFile 公钥登录（等价 ssh config `IdentityFile`） | 读取指定密钥文件（支持 `~` 展开），`ssh.PublicKeys(signer)` |

「代理」等价于不传任何身份时的默认 ssh 行为：先问 agent，agent 没有就读默认密钥文件。

## 3. 架构分层

遵循项目既有约定（纯逻辑 / UI 分离）：

```
internal/tunnel/              ← 纯逻辑层（零 Shirei 依赖，表驱动单测）
  config.go                   配置结构 + jsonc 解析/序列化
  tunnel.go                   单隧道：Connect / Test / Close（ssh.Dial + listener 转发）
  manager.go                  TunnelManager：所有隧道的运行时状态（并发安全）
  config_test.go              jsonc 解析/序列化单测
  tunnel_test.go              本地回环 mock 隧道连通性单测（用 mock ssh server）

views/
  tunnel_view.go              列表视图（分组 + 2 列网格 + 卡片 + 操作按钮）
  tunnel_dialog.go            新增/编辑弹窗（Modal）
  state.go                    新增 TunnelToolState（编辑表单缓冲 + 测试结果）
  sidebar.go                  新增侧栏项 ToolTunnel
```

### 纯逻辑 ↔ UI 交互边界
- `TunnelManager` 持有所有网络资源（`*ssh.Client`、`net.Listener`、后台 goroutine），**不放进 AppData**（AppData 只放 UI 表单状态）。
- UI 每帧调用 manager 的只读查询方法（`IsRunning(name)`、`Status(name)`）获取状态，manager 内部用 `sync.Mutex` 保护。
- UI 触发的动作为方法调用：`manager.Start(cfg)`、`manager.Stop(name)`、`manager.Test(cfg) error`。

## 4. 数据模型

### 配置（持久化）— `internal/tunnel/config.go`

```go
type AuthMethod string

const (
    AuthPassword  AuthMethod = "password"  // 密码
    AuthAgent     AuthMethod = "agent"     // 代理（ssh-copy-id 后系统默认）
    AuthPublicKey AuthMethod = "publickey" // 公钥（指定 IdentityFile）
)

// TunnelConfig 是一条隧道的完整配置，可序列化为 JSON
type TunnelConfig struct {
    Name       string    // 隧道名称（唯一标识，列表展示 + manager map key）
    Host       string    // 主机/IP
    Port       int       // SSH 端口，默认 22
    Username   string    // 登录用户名
    AuthMethod AuthMethod
    Password   string    // AuthMethod=password 时用，明文
    KeyPath    string    // AuthMethod=publickey 时用，支持 ~ 展开

    LocalHost  string    // 本地绑定主机，默认 127.0.0.1
    LocalPort  int       // 本地监听端口
    RemoteHost string    // 远程主机（SSH 服务端视角），通常 localhost
    RemotePort int       // 远程目标端口
}

// ConfigFile 是 settings.jsonc 的整体结构
type ConfigFile struct {
    Tunnels []TunnelConfig `json:"tunnels"`
}
```

默认值约定（保存前规范化，空值填默认）：`Port=22`、`LocalHost="127.0.0.1"`、`RemoteHost="localhost"`。

### 运行时状态（非持久化）— `internal/tunnel/manager.go`

```go
type tunnelRuntime struct {
    cfg      TunnelConfig
    running  bool
    errMsg   string        // 运行/测试时的错误（空=正常）
    client   *ssh.Client
    listener net.Listener
    cancel   context.CancelFunc
}

type TunnelManager struct {
    mu       sync.Mutex
    configs  []TunnelConfig          // 已保存的配置（与文件一致）
    runtimes map[string]*tunnelRuntime // key = tunnel name
}
```

`TunnelManager` 在应用启动时构造，随 app 生命周期常驻。掉线检测：转发 goroutine 检测到 `client` 断开（accept 后 dial 失败、或 io.Copy 返回错误）时，置 `running=false` + 记 `errMsg`，UI 下一帧自动反映「已断开」。

## 5. 隧道实现（golang.org/x/crypto/ssh）

### 认证构造 — 按 AuthMethod 选 AuthMethod
- `password`：`ssh.Password(cfg.Password)`
- `agent`：连本机 ssh-agent——`net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))` → `agent.NewClient(conn)` → `agentClient.Signers()` 拿到 agent 里所有 signer；`SSH_AUTH_SOCK` 未设或连不上则读默认密钥文件（`~/.ssh/id_rsa` / `id_ed25519`）。构造出的 signer 列表用 `ssh.PublicKeys(signers...)` 提供给 `ssh.Dial`（SSH 客户端会依次尝试）。
- `publickey`：`os.ReadFile(expandTilde(cfg.KeyPath))` → `ssh.ParsePrivateKey` → `ssh.PublicKeys(signer)`。

### Start(cfg) error
1. 构造 `ssh.ClientConfig{User, Auth, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 10s}`。
2. `ssh.Dial("tcp", host:port, config)` —— 失败返回带分类的错误（认证失败/网络超时/拒绝连接）。
3. `net.Listen("tcp", localHost:localPort)` —— 本地端口被占返回错误。
4. 起后台 goroutine：accept 循环，每个连接 `client.Dial("tcp", remoteHost:remotePort)` → 两个 goroutine 双向 `io.Copy` 桥接。
5. 记录 runtime，置 `running=true`。

### Test(cfg) error
- 只做「能连 + 能认证 + 远程端口能 dial」三步验证，**不保持转发**：`ssh.Dial` → `client.Dial("tcp", remoteHost:remotePort)` → 立即 close client。
- 返回的错误按类型归类为可读信息：网络不通 / 认证失败 / 远程端口不通 / 本地端口占用（Listen 探测）。
- Test 不写 runtime，是只读探测。

### Stop(name)
- cancel context → 关 listener（accept 循环退出）→ `client.Close()` → 置 `running=false`，清理 runtime（保留配置）。

### HostKeyCallback
- v1 用 `ssh.InsecureIgnoreHostKey()`（本地自用工具，首次连即信任，不做 known_hosts 管理）。此点在 spec 标注为已知简化，后续可加 known_hosts 校验。

### 依赖版本
- `golang.org/x/crypto` 选最新稳定版（当前模块缓存有 v0.50.0），`go get` 后由 `go mod tidy` 锁定。

## 6. 列表 UI（`views/tunnel_view.go`）

### 布局
- 上下两组：「已开启」组（顶部，无则不渲染）→ 「未开启」组。
- 每组内部 2 列网格：用 `Row` 容器，每 2 个卡片为一行（奇数个时最后一行单卡 + 占位）。
- 列表末尾固定一个「+ 新增」卡片（虚线边框 + 加号图标）。

### 卡片布局（每个隧道）
```
┌─────────────────────────────┐
│ ● 我的隧道        🔑 id_rsa │ ← 左：运行中绿点 / 右：认证标识
│ root@1.2.3.4:22             │ ← name@host:port
│ :8080 → :80                 │ ← localHost:localPort → remoteHost:remotePort
│                             │
│ [开启] [测试] [修改] [更多▼]│ → 更多展开「删除」
└─────────────────────────────┘
```

- **开启状态高亮**：卡片左侧绿色竖条（`BorderWidth` 左边） + 浅绿底色 + 名称前绿点（`●`）。未开启为灰色静默态。
- **认证标识**：公钥显示 🔑 + 密钥文件名（basename）；代理显示 🖥 或通用代理图标；密码不显示标识（或显示 🔒 但不显示密码内容）。
- **操作按钮**：
  - `开启` / `停止`（运行中显示「停止」）。
  - `测试`：调 `manager.Test(cfg)`，结果以绿/红字显示在卡片底部（临时，几秒后消失或下次操作覆盖）。
  - `修改`：打开编辑弹窗（表单预填该隧道配置）。
  - `更多`：`MenuButton` + `PopupPanel` 展开下拉，含「删除」（删除前弹确认）。

### 「+ 新增」卡片
- 虚线边框居中大加号，点击打开空白新增弹窗。

## 7. 新增/编辑弹窗（`views/tunnel_dialog.go`）

用 Shirei `Modal(width, dismiss, fn)` 实现。字段：

```
┌── 新增隧道 ──────────────────────┐
│ 隧道名称  [______________]        │
│ 主机/IP   [______________]        │
│ SSH 端口  [22]    用户名 [______]  │
│                                  │
│ 认证方式  〔密码|代理|公钥〕       │ ← SegmentedControl 三选一
│   密码     [______________]       │ ← 认证方式决定下方显示哪组
│   公钥路径 [~/ssh/id_rsa]         │
│                                  │
│ ── 端口转发 ──                    │
│ 本地主机  [127.0.0.1]             │
│ 本地端口  [____]                  │
│ 远程主机  [localhost]             │
│ 远程端口  [____]                  │
│                                  │
│ （测试结果显示区，绿/红字）        │
│                                  │
│ [测试隧道配置]   [保存隧道配置]    │
└──────────────────────────────────┘
```

- **认证方式**：`SegmentedControl[AuthMethod]` 三段（密码/代理/公钥），切换时下方动态显示对应字段组（密码→密码框；代理→无额外字段；公钥→密钥路径输入框）。
- **测试隧道配置**：用当前表单值（未保存）构造临时 `TunnelConfig` 调 `manager.Test`，结果显示在按钮下方（绿=成功，红=失败 + 错误分类）。
- **保存隧道配置**：
  - 校验必填（名称、主机、用户名、端口、本地/远程端口）。
  - 规范化默认值（Port=22 等）。
  - 写入 manager 配置列表（编辑模式替换原项，新增模式追加）。
  - 全量重写 `settings.jsonc`。
  - 关闭弹窗回列表。

## 8. 配置文件加载/保存（`internal/tunnel/config.go`）

### 路径
`~/.config/toolchest/settings.jsonc`（`~` 展开，目录不存在则创建）。

### 加载
1. 读取文件，不存在 → 返回空配置。
2. 剥离 jsonc 注释：去掉 `//` 行注释和 `/* */` 块注释（注意不破坏字符串内的 `//`）。
3. `json.Unmarshal` 到 `ConfigFile`。

### 保存
- 全量重写（格式化输出，字段顺序固定，缩进 2 空格）。
- 文件头加一段 `//` 注释说明（由 toolchest 自动生成，请勿手动改格式之类）。
- 注释保留策略：v1 不保留用户手写注释（全量重写），文件头注释固定。后续如需保留用户注释再增强。

### 文件格式示例
```jsonc
// toolchest SSH 隧道配置文件（由应用自动管理）
{
  "tunnels": [
    {
      "Name": "我的隧道",
      "Host": "1.2.3.4",
      "Port": 22,
      "Username": "root",
      "AuthMethod": "publickey",
      "Password": "",
      "KeyPath": "~/.ssh/id_rsa",
      "LocalHost": "127.0.0.1",
      "LocalPort": 8080,
      "RemoteHost": "localhost",
      "RemotePort": 80
    }
  ]
}
```

## 9. 并发与生命周期

- `TunnelManager` 用 `sync.Mutex` 保护 `configs` 和 `runtimes`。
- UI 在 immediate mode 每帧调用只读方法（`IsRunning`/`Status`/`Configs`），锁开销可忽略。
- 转发 goroutine 在 `client` 断开时自动置 `running=false` + 记 errMsg，无自动重连（v1）。
- 应用退出（窗口关闭）：遍历所有 running 隧道调用 `Stop`，清理资源（main.go 在 app.Run 返回后调用 `manager.CloseAll`）。

## 10. 错误处理

- 统一在相关区域下方红字显示，不弹模态框打断（遵循项目约定）。
- 错误分类（Test/Start 失败时归类为可读中文）：
  - 网络不通（dial timeout / connection refused）
  - 认证失败（password/key 被拒）
  - 远程端口不通（client.Dial 远程失败）
  - 本地端口被占（Listen 失败）

## 11. 测试策略

- `config_test.go`：表驱动，覆盖空文件 / 带注释 / 三种认证方式 / 默认值规范化 / 序列化往返。
- `tunnel_test.go`：用 `golang.org/x/crypto/ssh` 起本地 mock SSH server（内存 listener + ssh.NewServerConn），验证密码认证、公钥认证、本地转发连通（连本地监听端口能转发到 mock server 的目标端口）。覆盖 Test 的三种失败分类。
- 纯逻辑层 100% 可测，UI 层不写单测（immediate mode，靠 `--png` headless 渲染目检）。

## 12. 待确认 / 已知简化

- **HostKeyCallback = InsecureIgnoreHostKey**：v1 简化，不做 known_hosts 校验。本地自用工具可接受，后续可加。
- **无自动重连**：掉线后置已断开，用户手动重开。
- **注释不保留**：全量重写 jsonc，只保留文件头固定注释。
- **密码明文**：本地自用工具，确认可接受。

## 13. 实施顺序（写计划时细化，供参考）

1. 依赖引入 + 配置层（`internal/tunnel/config.go` + 单测）。
2. 隧道核心（`internal/tunnel/tunnel.go` + mock server 单测）。
3. Manager（`internal/tunnel/manager.go`）。
4. 侧栏接入 + state 扩展（`sidebar.go` / `state.go`）。
5. 列表视图（`tunnel_view.go`）。
6. 弹窗（`tunnel_dialog.go`）。
7. main.go 接入 manager 生命周期 + 启动时加载配置。
8. headless `--png-tunnel` 调试分支 + 用户本机验证。
