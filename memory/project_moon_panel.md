---
name: Moon Panel 项目背景
description: moon-panel 是用户自托管的个人导航面板，用来替代 Sun-Panel
type: project
originSessionId: 2bf565ba-ceec-48ee-9795-2d8ca3d79e24
---

用户在 **c:\moon-panel-dev**（PC HOMENET，本地 NTFS）自建导航面板项目 "Moon Panel"，定位为 Sun-Panel 的轻量替代品。原始路径 `d:\Projects\moon-panel` 和 SMB 副本 `P:\moon-panel\` 都已废弃，所有 git 操作走 `c:\moon-panel-dev\`。源码权威 = `c:\moon-panel-dev\`，GitHub remote = https://github.com/bevanho777-max/moon-panel.git。

**Why:** Sun-Panel 经常要求重新登录、近期开始收费，用户想自己掌握。

**核心需求要点（区别于一般导航面板）:**
- 默认公开模式，主页免登录；管理后台才要密码
- 单密码认证（明确不要邮箱/注册流程）
- 每张卡片有"内网地址 + 外网地址"双字段，前端一键切换
- 数据全 SQLite + JSON 导入导出
- Docker 单文件部署，目标平台含 NAS

**技术栈（已与用户对齐）:**
- 后端 Go 1.22+ + Gin + GORM + modernc.org/sqlite（纯 Go 免 CGO）
- 前端 Vue 3 + Vite + TS + Naive UI + Pinia
- 前端通过 `go:embed` 嵌入后端二进制，真正单文件部署
- httpOnly Cookie 而非 localStorage 存 token

**Docker 镜像源**：`Dockerfile` 用 `gcr.nju.edu.cn/distroless/static-debian12:nonroot`（南京大学公益镜像，国内访问稳定，内容与官方一致）。国外用户改回 `gcr.io` 即可。

**How to apply:** 写代码时遵循 PROJECT.md 的结构和阶段划分；遇到设计取舍优先考虑"NAS 友好/低资源/部署简单"；不要引入用户系统、邮件、外部依赖等违背"轻量"定位的特性。
