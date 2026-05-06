---
name: 子阶段交付前必跑的本地验证 checklist
description: 后端 vet+build、前端 npm run build（本地 NTFS 直跑 / SMB 走 F-lite 副本）、shell sh -n，三步缺一不可
type: feedback
---

每个子阶段交付**之前**必须本地跑通这三步，缺一不可：

1. **后端**：
   ```
   cd c:\moon-panel-dev\backend && go vet ./... && go build ./cmd/server
   ```
   Go 用系统 PATH（PC HOMENET 已装）启二进制 + curl 把所有改动端点对一遍预期。便携 Go（`D:\Projects\moon-panel\tools\go`）路径已废弃。

2. **前端**（v0.2.7+ 默认本地 NTFS 直跑；源码若位于 SMB 则走 F-lite 副本）：
   ```
   # 默认（c:\moon-panel-dev 本地 NTFS，v0.2.5+ 起）：
   cd c:\moon-panel-dev\frontend
   npm run build

   # 仅当源码在 SMB（P:\moon-panel\）时走 F-lite，理由见下文：
   robocopy P:\moon-panel\frontend C:\moon-build\frontend /MIR /XD node_modules /XF *.log
   cd C:\moon-build\frontend
   npm run build
   ```
   看到 `✓ built in X.XXs` 才算过。**不能用 `vue-tsc --noEmit` 替代** —— vue-tsc 只查 `<script>` 块的 TS 类型，`<template>` 的 HTML 解析（属性嵌套引号、标签结构）只有 vite 的 @vue/compiler-sfc 才完整跑。Phase 2.1 第一次交付就栽在这条上：Groups.vue 第 152 行嵌套双引号 vue-tsc 看不到，docker build 才崩。

3. **Shell 脚本**：
   ```
   sh -n c:\moon-panel-dev\docker\entrypoint.sh
   ```

---

## F-lite 验证副本工作流（★ 仅当源码在 SMB 时适用 ★）

**为什么要这个**：当源码位于群晖 SMB 共享 `P:\moon-panel\` 时，esbuild native 在 SMB drive-letter 上解析 node_modules 会失败（实测：`Failed to resolve entry for package "@vitejs/plugin-vue"`）。Windows junction (`mklink /J`) 也走不通——SMB 卷不支持 reparse point（内核硬约束）。

**v0.2.5+ 实际上**：源码已迁到 `c:\moon-panel-dev\`（本地 NTFS），F-lite **不需要**——`cd c:\moon-panel-dev\frontend && npm run dev / build / type-check` 直接通（v0.2.7 实战验证）。本节保留为 SMB 场景兜底，不再是首选。详见 [feedback_workdir_authority.md](feedback_workdir_authority.md)。

**F-lite 解法（SMB 场景）**：用本地 NTFS 路径 `C:\moon-build\frontend` 做**只读验证副本**，源码权威**当时**在 P:\。

**固定原则（仅 SMB 场景）**：
- `c:\moon-panel-dev\` 是**当前**源码权威（本地 NTFS git repo，remote = GitHub origin）；v0.2.5/2.6/2.7 全部 commit + push 都直接在 c:\moon-panel-dev\ 操作（reflog 反推证据见 [feedback_workdir_authority.md](feedback_workdir_authority.md)）
- 历史 SMB 场景下 `P:\moon-panel\` 曾是源码权威；现已废弃，仅 legacy 备份
- `C:\moon-build\frontend\` 是 SMB 场景下的**只读验证副本**（不进 git、不写代码）
- `node_modules` 永远只装在 C:，不污染 P:
- robocopy `/MIR` 单向同步，C 永远是 P 的纯净镜像（无 D/P 漂移可能）

**首次配置**（一次性 2026-04-28 已做）：
- `mkdir C:\moon-build\frontend`
- `robocopy P:\moon-panel\frontend C:\moon-build\frontend /MIR /XD node_modules /XF *.log`
- `cd C:\moon-build\frontend && npm install`

**之后每次前端交付前**：
1. `robocopy P:\moon-panel\frontend C:\moon-build\frontend /MIR /XD node_modules /XF *.log` （快，只同步源码差异）
2. `cd C:\moon-build\frontend && npm run build`
3. 看到 `✓ built in` 才能交付

**不要在 P: 上装 node_modules**：P:\moon-panel\frontend\node_modules **永远应该不存在**。如果不小心装了（比如某次 npm install 跑在了 P:），必须 `rm -rf` 删掉。`.dockerignore` 已排除该路径，对 docker build 无影响。

**SMB 上 robocopy 是单向**：删 P 上的源码文件 → 下次 robocopy /MIR 会同步删 C 上的对应文件。新增、修改同理。这是设计，不是 bug。

**调用语法注意**：在 git bash 里直接调 `robocopy /MIR ...` 会被 bash 的路径转换误解（把 `/MIR` 当 Unix 路径）。从 PowerShell 调用最干净，或在 bash 里用 `cmd //c "robocopy ..."`。

---

**Docker 由用户跑（协议 B）**，但交付文档必须给：
- 完整可粘贴的 `sudo docker compose build --no-cache && up -d && logs | head -25` 三连
- 预期启动日志关键行
- 完整 curl 清单（每条带预期响应）
- 浏览器验收清单
- **Backend 改动（新路由 / 新 handler / 新 register）必须额外给 BuildKit cache 全清三连作为兜底**（见 [feedback_docker_cache_blindspot.md](feedback_docker_cache_blindspot.md)）：
  ```bash
  sudo docker image rm moon-panel:latest -f
  sudo docker builder prune -af
  sudo docker compose build --no-cache --pull
  ```
  以"如果新路由仍 404，跑这三条"的形式给——不是建议第一次就跑（费时），是兜底

**How to apply:** 三步任意一步未跑就别交付。F-lite 副本工作流是硬性流程，不是临时绕路。
