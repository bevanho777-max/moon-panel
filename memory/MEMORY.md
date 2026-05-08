# Moon Panel — Claude Code memory log

> This directory is the persistent memory written by [Claude Code](https://claude.com/claude-code) while building Moon Panel. Each entry below points to a `feedback_*` / `project_*` file capturing a lesson, decision, or correction from a real session — what to repeat, what to avoid, and why. The entries shaped Claude's behavior on later turns, so future contributors using Claude Code on this repo inherit them automatically. They're committed publicly (rather than kept private to the maintainer) so the AI-collaboration trail is auditable end-to-end. See the README's *AI Collaboration* section for context.

## Entries

- [Moon Panel 项目背景](project_moon_panel.md) — 自托管导航面板，替代 Sun-Panel，单密码 + 内外网切换 + 单文件部署
- [Memory 写入位置规则](feedback_memory_location.md) — 唯一权威 c:\moon-panel-dev\memory\(git tracked), P:\ 已废弃为 legacy 备份
- [状态变更操作前必须征求确认](feedback_state_changes.md) — push/rebase/删文件/装依赖等先问，代码改动直接做
- [子阶段交付前必跑的本地验证 checklist](feedback_delivery_routine.md) — 后端 vet+build、前端走 F-lite C:\moon-build 副本跑 npm run build、shell sh -n
- [D 盘 moon-panel 残留一律不动](feedback_d_drive_residue.md) — D:\Projects\moon-panel、D:\moon-modules-frontend 等永远不删不同步
- [Docker layer cache 对小幅 backend 改动的盲区](feedback_docker_cache_blindspot.md) — 新加 handler 后路由 404 多半是 BuildKit cache，交付时主动给全清三连
- [DB 重置后必须先 init 密码再做其他测试](feedback_db_reset_password.md) — rm moon.db 后系统回首次启动态，admin 不存在
- [localStorage key 命名规范](feedback_localstorage_naming.md) — 全部 moon. 前缀，点分隔子分类，便于隔离/调试/重置
- [国内网络镜像清单](feedback_cn_network_mirrors.md) — Docker 构建里 Go / Alpine apk / gcr.io / npm 的 CN 镜像配置，国外切回方法
- [前端子阶段交付必须含 Playwright 截图](feedback_screenshot_routine.md) — Phase 2.4 起，桌面 1280×800 + 移动 iPhone 14 Pro Max 各一张，命名规范 + 索引格式
- [外部资源必须有 fallback；Phase 3 icon proxy+cache](feedback_external_resource_fallback.md) — 用户 NAS 翻墙但其他客户端不假设；图片/API/CDN 全 fallback；Phase 3 图标走后端下载缓存
- [SSRF 防护规则](feedback_ssrf_protection.md) — 用户 URL → 后端 fetch 类端点必须 DNS 解析 + IP 段拒绝 + 复用 IP 防 rebinding；Phase 2.5b/3 共用
- [Gin 静态文件路由 + NoRoute 共存](feedback_gin_static_route_order.md) — r.Static() 注册的前缀路由不会被 NoRoute 拦截，遗漏会导致静态 URL 返回 SPA HTML
- [字面文本替换前必先征询](feedback_no_unilateral_substitution.md) — spec 里 user 给的字面字符任何替换都要先停下问，不擅自决定
- [工作目录权威性核对](feedback_workdir_authority.md) — 三件证据法（remote/HEAD/reflog）核对 git 工作目录是否真权威，防改错副本/推错 repo
- [PowerShell 5.1 native exe arg quoting 不可靠](feedback_ps_native_arg_quoting.md) — UTF-8 过 / ASCII " 丢; commit msg + tag annotation 必走 -F file 模式
- [NaiveUI :deep() override 改组件内部 BEM 的副作用](feedback_naiveui_deep_override.md) — 改 .n-button__content 等内部 box 属性会改变 outer width, 在 flex 父容器中触发 wrap; 安全做法在组件根类加 align
- [Mobile flex 容器多元素 wrap 诊断要量化总宽](feedback_mobile_layout_total_width_audit.md) — wrap 类问题先量化元素总宽 vs viewport, 别只看截图局部; v0.2.10 Task 2.13 误诊教训
- [Monitor jq 可用性教训](feedback_monitor_jq_availability.md) — bash CI/release Monitor 用外部 jq 必须先验证可用; gh --jq flag 是首选, 否则 polling silent fail 触发 false STUCK_WARNING (v0.2.13 Ship 1)
- [Spec 重发主动识别](feedback_patch_repeat_detection.md) — 收到 spec 先 grep 验证当前状态, 已实施完整时主动报告等指令, 不擅自重跑 (v0.2.13 Patch 3)
- [视觉门 patch 迭代健康模式](feedback_visual_gate_iter_healthy.md) — 每个 patch 必须对应 Bevan 真实反馈, 健康范围 1-5, 6+ patches 必有 spec 误判 (v0.2.13 累积)
- [E2E 测试同步教训](feedback_e2e_test_sync.md) — Spec 阶段删 UI element 时必 audit e2e test 引用, 否则 CI fail; Task 1 grep 必加 frontend/tests/e2e/ pattern (v0.2.15 Patch A)
- [Spec 决策 default 保守倾向教训](feedback_decision_default_bias.md) — Task 1 候选不 default 推保守, 让 Bevan 从 daily UX 视角拍板; v0.2.15 P0 a B.1a→B.1b 修正实例
- [Chore commit 路径策略](feedback_chore_commit_path_strategy.md) — chore 不 amend, 单独 commit 紧跟 feat; tag 指向 HEAD 含 chore; paths-ignore 决定是否触发 CI (v0.2.13/14/15 累积)
