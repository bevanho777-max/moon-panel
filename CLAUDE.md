# Moon Panel — Claude Code Working Memory

本文件是 Claude Code (Anthropic 命令行编程助手) 在 Moon Panel 项目工作的
onboarding 文件。每次 Claude Code 启动会自动读这个文件了解项目背景 + 工程纪律。

完整累积教训详细版见 [`memory/MEMORY.md`](memory/MEMORY.md) (29 entries)。
本文件是高密度入口 + 高频引用 + v0.2.23 累积的最新边界。

---

## 项目背景

Moon Panel 是 Bevan 的自托管个人 NAS dashboard, Sun-Panel 轻量替代品。
单文件 Go 二进制, 前端通过 go:embed 内嵌, SQLite 存储 (无 CGO), Docker 多架构部署。

- **Bevan**: sole developer + sole user (GitHub: bevanho777-max)
- **沟通语言**: 中文
- **设计原则**: 单用户 + 公网可访问

详见 [`memory/project_moon_panel.md`](memory/project_moon_panel.md)。

---

## 技术栈

- **Backend**: Go + Gin + GORM + SQLite (无 CGO)
- **Frontend**: Vue 3 + NaiveUI + Pinia + TypeScript + Vite
- **Build**: 单 binary go:embed 前端 dist
- **Deploy**: Docker, GitHub Actions, ghcr.io multi-arch (amd64 + arm64)
- **CI 时长**: ~175s avg
- **release.yml 时长**: 2-9 min (multi-arch QEMU build)
- **NAS**: Synology DSM 7.2 @ 192.168.1.75, PUID=1026 PGID=100

---

## 核心工程原则

### 1. 做就做完美

Bevan 核心原则: 不留尾巴。选完整方案而非增量, 不 defer 已知问题。
累积工程债清单累积时, 整体 cleanup release 处理。

### 2. 不擅自决定

涉及产品 / 设计决策时:
- 给候选方案 + 推荐 + 理由
- 等 Bevan 拍板再继续
- 不替 Bevan 做产品决定

事实修正除外 (例如代码字段名拼写、Go zero value、grep 找到的客观差异)。

### 3. 主动 flag spec 假设错误

实施前 / 中发现 spec 跟实际不符:
1. 立刻 stop
2. 报告事实 (verbatim 关键)
3. 给候选方案 + 推荐 + 理由
4. 等 Bevan 拍板再继续

比错了再改便宜 1 个数量级。累积 v0.2.x 已 flag 44+ 项 spec 错误。

### 4. 单一职责 release

每个 release 一个清晰主题。scope creep 是项目延期主因。
累积 7 连 0-patch streak (v0.2.17-v0.2.23) 全部单一职责。

---

## Release 流程

1. **决策锁定** — Bevan 拍板 N 个决策点 (D1-D7 / E1-E5 等)
2. **Task 1 grep** — read-only 探索, 5 类 selector audit (V2 教训, 见 `feedback_e2e_test_sync_v2.md`)
3. **Phase A-D 实施** — 累积主动 flag spec 假设错误
4. **dev 视觉门** — 4 主题组合 (PC/Mobile × Risen/Moon) + 网络场景
5. **Stage A/B 报告** — Bevan 审 commit 前
6. **Push + CI + Tag + release.yml** — gh CLI 用 `--jq` flag
7. **NAS pull + 真机视觉验收** — Bevan SSH 192.168.1.75
8. **CHANGELOG + README 更新历史同步** — 文档跟 code 一致

### 关键 checkpoint (v0.2.23 教训)

- **Spec 留 placeholder** (e.g. `2026-MM-DD`) 必须 surface 给 Bevan 填,
  或拒绝 ship 直到填好
- **Release prep 必须 grep TODO / placeholder / MM-DD 等占位符**
- **CHANGELOG 跟 tag commit date 校准** (避免日期不一致)
- **GitHub Release notes 跟 CHANGELOG 同步** (release.yml extract 后改 CHANGELOG 不会自动重读,需 `gh release edit` 手动刷)

---

## 累积纪律 (v0.2.x lessons)

高价值高频引用 (完整 29-entry index 见 [`memory/MEMORY.md`](memory/MEMORY.md)):

- [`@memory/project_moon_panel.md`](memory/project_moon_panel.md) — 项目背景速读 (Sun-Panel 替代, 单密码, 内外网切换, 单文件部署)
- [`@memory/feedback_state_changes.md`](memory/feedback_state_changes.md) — push/rebase/删文件/装依赖前必征求确认, 代码改动直接做
- [`@memory/feedback_no_unilateral_substitution.md`](memory/feedback_no_unilateral_substitution.md) — Bevan 给的字面字符任何替换都要先停下问, 不擅自决定
- [`@memory/feedback_decision_default_bias.md`](memory/feedback_decision_default_bias.md) — Task 1 候选别 default 推保守, 让 Bevan 从 daily UX 视角拍板
- [`@memory/feedback_delivery_routine.md`](memory/feedback_delivery_routine.md) — 子阶段交付前必跑: vet+build / npm run build (走 C:\moon-build 副本) / sh -n
- [`@memory/feedback_e2e_test_sync_v2.md`](memory/feedback_e2e_test_sync_v2.md) — Task 1 grep 5 类 selector audit (自有 BEM + 第三方 internal + UI text + data-* + role/aria)
- [`@memory/feedback_e2e_test_avoid_third_party_internal.md`](memory/feedback_e2e_test_avoid_third_party_internal.md) — 优先项目自有 BEM, 不依赖 NaiveUI internal (`.n-data-table-tr` 等)
- [`@memory/feedback_naiveui_deep_override.md`](memory/feedback_naiveui_deep_override.md) — `:deep()` 改 NaiveUI 内部 BEM 副作用 (改 width → flex 父容器 wrap), 安全做法在组件根类加 align
- [`@memory/feedback_mobile_layout_total_width_audit.md`](memory/feedback_mobile_layout_total_width_audit.md) — Mobile flex wrap 诊断先量化元素总宽 vs viewport, 别只看截图局部
- [`@memory/feedback_visual_gate_iter_healthy.md`](memory/feedback_visual_gate_iter_healthy.md) — 视觉门 patch 健康范围 1-5, 6+ patches 必有 spec 误判
- [`@memory/feedback_patch_repeat_detection.md`](memory/feedback_patch_repeat_detection.md) — 收到 spec 先 grep 验证当前状态, 已实施完整时主动报告等指令, 不擅自重跑
- [`@memory/feedback_monitor_jq_availability.md`](memory/feedback_monitor_jq_availability.md) — gh CLI `--jq` flag 是首选, 外部 jq 必须先验证可用, 否则 false STUCK_WARNING
- [`@memory/feedback_chore_commit_path_strategy.md`](memory/feedback_chore_commit_path_strategy.md) — chore 不 amend, 单独 commit 紧跟 feat; tag 指向 HEAD 含 chore; paths-ignore 决定 CI
- [`@memory/feedback_screenshot_routine.md`](memory/feedback_screenshot_routine.md) — 前端子阶段交付必含 Playwright 截图 (1280×800 + iPhone 14 Pro Max), 索引格式
- [`@memory/feedback_workdir_authority.md`](memory/feedback_workdir_authority.md) — 三件证据法 (remote/HEAD/reflog) 核对 git 工作目录, 防改错副本
- [`@memory/feedback_memory_location.md`](memory/feedback_memory_location.md) — 唯一权威 c:\moon-panel-dev\memory\ (git tracked), P:\ 已废弃为 legacy 备份
- [`@memory/feedback_ps_native_arg_quoting.md`](memory/feedback_ps_native_arg_quoting.md) — PowerShell 5.1 native exe arg quoting 不可靠, commit msg / tag annotation 必走 `-F file` 模式
- [`@memory/feedback_docker_cache_blindspot.md`](memory/feedback_docker_cache_blindspot.md) — Docker layer cache 盲区: 新 handler 后路由 404 多半是 BuildKit cache, 交付时给全清三连
- [`@memory/feedback_gin_static_route_order.md`](memory/feedback_gin_static_route_order.md) — `r.Static()` 注册前缀路由不会被 NoRoute 拦截, 遗漏静态 URL 返回 SPA HTML
- [`@memory/feedback_ssrf_protection.md`](memory/feedback_ssrf_protection.md) — 用户 URL → 后端 fetch 必须 DNS 解析 + IP 段拒绝 + 复用 IP 防 rebinding
- [`@memory/feedback_external_resource_fallback.md`](memory/feedback_external_resource_fallback.md) — 用户 NAS 翻墙但其他客户端不假设; 图片/API/CDN 全 fallback
- [`@memory/accumulated_lesson_e2e_chore_pattern.md`](memory/accumulated_lesson_e2e_chore_pattern.md) — v0.2.15+v0.2.16 连续 2 release 同模式 e2e fail, 一次到位

---

## 已知边界 / 工程债

### Backend
- Default port = 3000 (dev 需 `$env:MOON_PORT="3001"`)
- Setting 表是 generic key→value (无 typed schema, 用 key-specific 验证)
- bootstrapXxxSettings 模式: main.go 定义 bootstrap 函数, 跟 startup chain 走
- Phase A new key 不需要 DB migration (model.Setting 通用表)

### Frontend
- NaiveUI NSpace v2 用 gap+inline-flex 子项, `flex: 1` 某些场景塌 → 用 `width: 100%` 在外层 NSpace
- ★ NaiveUI 组件用前必须显式 `import { N* } from 'naive-ui'`, 否则 silent 渲染 0×0 unknown element (v0.2.23 patch-5: SiteSettings.vue NInput 自 v0.2.0 起 22 release silently broken)
- Pinia store 用 setup API + ref(), 跟现有 ui.ts 模式对齐
- `getPanel()` 已 unwrap `{code, msg, data}` wrapper, 消费方直接访问 `panel.value.site`

### Browser API
- Chrome `fetch(mode:'no-cors')` 不允许 `redirect:'manual'` (同步 TypeError, v0.2.23 patch-2 教训)
- 探测策略用 `<img>` 绕过 CORS+redirect 限制 (v0.2.23 patch-3): img.onload + img.onerror 都算 reachable, 仅 setTimeout 触发算 unreachable
- `sessionStorage.setItem()` 可能在 iOS Safari private mode 抛 QuotaExceededError, 用 try/catch 包

### Visual rendering
- 静态渲染失败不 crash, 是 silent 0×0 (例 unknown element / undefined ref binding)
- 视觉门必须真看 UI, 不能依赖 type-check + vitest 通过就放过 (v0.2.23 patch-4/5 教训)
- "Prod 一直 work" 是高危假设 — 视觉静默失败可能多年未被发现, 不能基于代码 diff 单方面声明"修好"

### Dev environment
- dev 环境 themes / wallpapers / svg 资源 broken (Vite 处理跟 prod build 不一致), NAS 真机必测
- dev-backend.bat 当前缺失 (v0.2.24 backlog)
- Bash on Windows (MSYS / Git Bash): `gh api` 路径前导 `/` 被改写为 filesystem path, 用 `gh api users/...` 不带前导 `/`
- Playwright `readme-screenshots.spec.ts` 默认写 `docs/screenshots/`, 跑 e2e 前应该设 `MOON_README_SHOTS_DIR=/tmp/xxx` 避免覆写 committed 截图

### CI / GitHub Actions
- Node 20 actions (`checkout@v4`, `setup-go@v5`, `setup-node@v4`, `docker/*`) 即将 deprecation enforcement (2026-06-02), v0.2.24 backlog 升级到 v5+
- release.yml `Extract release notes from CHANGELOG.md` step 在 publish 时 extract, 不会自动重读 CHANGELOG — 后续 CHANGELOG 修订需 `gh release edit <tag> --notes-file` 手动刷

---

## Spec 协议

- Bevan 发整段中文 spec, Claude Code 按 Phase / Stage 分阶段报告
- 每阶段报告 verbatim diff (关键 5-15 行) + 主动 flag
- 不 commit 不 push 不 tag 直到 Bevan 审通过
- 单一 commit (除非明确说拆分)
- Commit message 用 `Co-Authored-By: Claude <noreply@anthropic.com>`
- PR / release tag / NAS pull 全部等 Bevan 明确批准

---

## 工具偏好

- **Terminal editor**: vi / vim (★ 不 nano ★)
- **GitHub CLI**: `gh run watch <id> --interval 10 --exit-status` 用 `--jq` flag
- **Plan B**: GitHub Actions cancel + retag same commit (累积命中率 4.5%)
- **Bash on Windows**: `gh api` 路径前导 `/` 不带 (MSYS 改写问题)
- **HEREDOC for git commit**: 多行 commit message 用 `git commit -m "$(cat <<'EOF' ... EOF)"` 避免 PowerShell quoting 问题

---

## 数据 (截至 v0.2.23 ship, 2026-05-16)

- 累积 release: v0.1.0 → v0.2.23 (含 23+ minor releases)
- 0-patch streak: 7 连 (v0.2.17 - v0.2.23)
- 主动 flag 累计: 44+ 项 (跨 release)
- 视觉门 fail safe 抓 bug: 多次
- silent prod bug 顺手修: 1 项 (v0.2.0 起 22 release 都没人发现)

---

## 文件位置

- **项目根**: `c:\moon-panel-dev\`
- **Backend**: `backend/` (Go modules)
- **Frontend**: `frontend/` (Vue 3)
- **累积教训**: `memory/` (29 个 .md, 含 MEMORY.md index)
- **Build sync**: `c:\moon-build\` (robocopy + type-check + build)
- **CI workflows**: `.github/workflows/*.yml`

---

## 跟 README / CHANGELOG 关系

- **README.md / README.zh-CN.md**: 给项目访客看的, 双语, 末尾「更新历史」section v0.2.23 起 backfill
- **CHANGELOG.md**: 详细每 release 变更记录, Keep a Changelog 格式
- **CLAUDE.md (本文)**: 给 Claude Code 看的工程纪律 + 项目上下文 + memory/ 索引入口

3 文件互不冗余, 各司其职。
