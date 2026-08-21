---
title: DBeaver-凭据解密
date: 2026-08-21
updated: 2026-08-21
tags: [dbeaver, aes, 凭据, 逆向]
track: knowledge
module: internal/tools/dbeavertool
---

## 概述

DBeaver 把数据库连接密码存在 workspace 的 `.dbeaver/credentials-config.json`，用固定 key 的 AES-128-CBC 加密（key 社区公开）。toolchest 的 DBeaver 凭据工具用纯 Go 复刻了 openssl 解密链路，并关联同目录 `data-sources.json` 补全连接名称/驱动/主机。

## 核心内容

### 加密方案（社区公开）

- 算法：AES-128-CBC
- key：`babb4a9f774ab853c96c2d653dfe544a`（hex）
- iv：全零（16 字节 0）
- 明文结构：**16 字节随机前缀 + JSON**，PKCS7 padding
- 等价命令：
  ```bash
  openssl aes-128-cbc -d -K babb4a9f774ab853c96c2d653dfe544a \
    -iv 00000000000000000000000000000000 \
    -in <workspace>/.dbeaver/credentials-config.json | dd bs=1 skip=16
  ```

### 凭据文件结构（解密后，无外层包装）

```json
{"<连接ID>": {"#connection": {"user": "...", "password": "..."}}}
```

连接 ID 形如 `mysql8-193ab26b9e4-2527dc101f9196b` / `com_clickhouse-1a01d70b83c-...`——**本身无业务含义**，需用 ID 关联 `data-sources.json` 才能得到名称。

### data-sources.json（同目录，明文）

```json
{"connections": {"<同款ID>": {"name": "logs", "driver": "com_clickhouse",
  "configuration": {"host": "...", "port": "3306"}}}}
```

- 文件缺失/损坏不致命：凭据仍可展示，名称回退 ID
- **port 是字符串** `"5432"` 不是数字——Go 侧声明 `int` 会让整个对象 Unmarshal 失败、名称静默关联不上（踩过）；须用 `any` + 分支解析

### DDD Lite 边界（grill 定稿）

查询型工具，无跨实体不变量，**不立聚合根**：
- 领域 = 纯函数组 `Decrypt` / `Parse` / `OpenWorkspace` + sentinel 错误（ErrNotCredentials/ErrNotWorkspace/ErrBadContent）
- `CredentialEntry` 值对象（含 data-sources 关联出的 Name/Driver/Host）
- 目录记忆（RecentDirs ≤8 去重置顶，打开失败不进缓存）是应用层便利状态，留在 Tool State

## 关键代码路径

- `internal/tools/dbeavertool/workspace.go` — 领域纯函数：Decrypt（含 PKCS7 校验+跳前缀）/ Parse / OpenWorkspace / readDataSourceNames
- `internal/tools/dbeavertool/state.go` — 目录缓存 State（持久化 config.yaml dbeaver 节）
- `internal/tools/dbeavertool/tool.go` — 应用层：ops select/refresh；工具栏 Actions（选择目录=本地 file picker / 目录下拉=radio / 刷新）
- `frontend/src/pages/DbeaverPage.tsx` — 卡片渲染 + 点击复制 + 原生目录选择（Wails `Dialogs.OpenFile` CanChooseDirectories）

## 注意事项

- PKCS7 padding 校验失败即报「不是有效的凭据文件」——部分 workspace（如 DBeaverDataSync/General）用这组 key 解不开（加密参数可能不同/新版本），报错即可，不用穷举
- credentials-config.json 可能不存在（连接从未存密码），与「存在但解不开」是不同错误（ErrNotWorkspace vs ErrNotCredentials）
- Actions 的 radio 形态点击走 `invoke(toolID, action.id, value)`——**Action ID 必须与 Go handler 名一致**（踩过：ID 起成 `dir` 报「未知操作」）

## 关联

- [[Wails-v3-Linux-桌面图标与GTK-app-id]] — 同批交付
