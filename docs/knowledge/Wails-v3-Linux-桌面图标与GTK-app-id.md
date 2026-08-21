---
title: Wails-v3-Linux-桌面图标与GTK-app-id
date: 2026-08-21
updated: 2026-08-21
tags: [wails, gnome, desktop-entry, 图标]
track: bug
module: 打包/安装
---

## 问题

Wails v3 应用安装到 GNOME 后，抽屉（app grid）和活动搜索里显示通用图标，名称显示为裸 app-id `org.wails.toolchest` 而不是 desktop 文件里的 `Name=Tool Dev`。

## 症状

- 任务栏/dock 图标正常（窗口自带 icon）
- 抽屉和搜索结果：通用图标 + `org.wails.toolchest` 名字
- `desktop-file-validate` 通过、SVG 能被 librsvg/convert 渲染、系统 144 个同目录 SVG 图标均正常

## 排查过程

1. desktop 文件改名 `org.wails.toolchest.desktop`（app-id 匹配）→ Shell 重启后仍不显示
2. `update-desktop-database` + `gtk-update-icon-cache -f` → 无效（GNOME Shell 不读 icon cache，自己走 StTextureCache）
3. 注销重进（Shell 重启晚于图标安装，排除缓存）→ 仍不显示
4. 验证 SVG 本身可渲染（`convert icon.svg test.png` 成功）→ 排除 SVG 兼容性
5. **Icon= 改成绝对路径 → 立即生效**（Shell 对 desktop 文件有 inotify 监听，换 Icon 字符串即新缓存 key，无需注销）

## 根因

Wails v3 beta.9 在 Linux 上把 GTK application id **硬编码**为 `org.wails.<二进制名>`（`pkg/application/linux_cgo.go:149` 的 `appNew()`，不可配置）。窗口 app-id 与 desktop 文件按 app-id 匹配后关联正确；但 GNOME Shell 对用户级 hicolor 目录下**带点号的名字**（`org.wails.toolchest.svg`）theme 查找失败（系统级 `/usr/share/icons/hicolor` 同名机制正常，用户级 `~/.local/share/icons` 的带点 SVG 未被 Shell 的 icon theme 查找命中），于是回退通用图标。

## 解决方案

desktop 文件 `Icon=` 用**绝对路径**，绕过 theme 查找（GNOME Shell 支持绝对路径 Icon）：

```ini
# before（theme 名查找，用户级带点号失效）
Icon=org.wails.toolchest

# after（绝对路径，立即生效）
Icon=/home/<user>/.local/share/icons/hicolor/scalable/apps/org.wails.toolchest.svg
```

Taskfile install 生成时模板化用户路径（见 `Taskfile.yml` install 任务）。

## 预防

- Wails v3 Linux 应用的 desktop 文件必须按 `org.wails.<name>.desktop` 命名（窗口 app-id 匹配），Icon 用绝对路径
- 排查此类问题先确认三件事：desktop 文件名 == 窗口 app-id；SVG 可渲染；`xprop WM_CLASS`（X11）/ app-id（Wayland）与 desktop 文件对应
- GNOME Shell 的图标缓存重启 Shell 才全量重建；Wayland 下 **不能** `killall gnome-shell`（会结束会话），X11 才可 Alt+F2 → r

## 关联

- [[DBeaver-凭据解密]] — 同批交付的工具
