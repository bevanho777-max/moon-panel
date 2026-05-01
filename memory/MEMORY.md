# Moon Panel — Claude Code memory log

> This directory is the persistent memory written by [Claude Code](https://claude.com/claude-code) while building Moon Panel. Each entry below points to a `feedback_*` / `project_*` file capturing a lesson, decision, or correction from a real session — what to repeat, what to avoid, and why. The entries shaped Claude's behavior on later turns, so future contributors using Claude Code on this repo inherit them automatically. They're committed publicly (rather than kept private to the maintainer) so the AI-collaboration trail is auditable end-to-end. See the README's *AI Collaboration* section for context.

## Entries

- [Moon Panel 项目背景](project_moon_panel.md) — 自托管导航面板，替代 Sun-Panel，单密码 + 内外网切换 + 单文件部署
- [Memory 写入位置](feedback_memory_location.md) — 新 memory 只写 P:\moon-panel\memory\，不要写自动 memory 目录
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
