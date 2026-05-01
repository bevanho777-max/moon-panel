---
name: 外部资源访问必须有 fallback；Phase 3 图标走 proxy + cache
description: 用户 NAS 已翻墙但其他主机/最终用户不假设；图片 / API / 第三方 CDN 全要 fallback；Phase 3 icon library 设计核心是 proxy+cache
type: feedback
---

## 网络访问假设

- **用户的 NAS（生产部署环境）**：已加入翻墙白名单，访问 Google / 外部图标 / 任何境外资源**畅通**
- **用户的 Windows 开发机**：翻墙状态**未知**——不要假设它能直连境外，遇到证据再说
- **moon-panel 最终用户**（如果项目开源后）：**绝大多数不能假设翻墙**

所以代码层永远按"外部资源可能失败"设计，不能用"NAS 能访问"来掩盖客户端访问问题。

## 必须做的 fallback

1. **`<img>` 加 `@error` handler**：任何外部图片都要有错误兜底
   - URL 失效、CDN 抽风、服务方下线、用户禁用第三方加载，**都会触发 `error` 事件**
   - Fallback 视觉：占位图标（如 `lucide:image-off`）+ tooltip 显示原始 URL
   - 例：`CardItem.vue` 的 icon thumbnail 在 Phase 2.4 第二轮加上

2. **API fetch 必须 try/catch**：网络异常 / 4xx / 5xx 都要走错误分支，UI 给反馈
   - 现状已基本覆盖：`api/client.ts` 拦截器 + 各页面 `try { await api(...) } catch (e) { message.error(...) }` 模式

3. **第三方 CDN 资源必须可 self-host**：禁止运行时直连 jsdelivr / cdnjs / unpkg / Google Fonts 等
   - 字体、JS lib 要么 npm 包打进 bundle，要么放 `/uploads/public/` 自托管
   - Naive UI 默认走 vfonts，已经是本地

## Phase 3 图标库设计核心：icon proxy + cache

用户输入流程（设想）：
1. 用户在 admin 编辑器输入 `https://example.com/icon.png`
2. **后端 download 一次**该 URL → 计算 hash 命名 → 存 `/data/uploads/public/icons/<hash>.<ext>`
3. 数据库 `card.icon` 改存 `upload:public/icons/<hash>.png`（统一前缀）
4. 之后所有用户访问主页 → 从本地 `/uploads/public/icons/<hash>.png` 拿，不再走外网

好处：
- 用户改图标地址时一次性下载固化，不依赖原 URL 长期存活
- 最终用户访问主页时**完全不出网**（NAS 内网用户也能看到图标）
- 图片 hash 命名 + 不可变 → 静态资源缓存策略简单

**还需要**：
- `lucide:` / `iconify:` 前缀走打包进 bundle 的 SVG，零网络
- `upload:public/icons/...` 自托管路径已经在内网
- 用户可强制不下载（"使用原始 URL"开关）以省磁盘空间，但默认 download

**Phase 3 backend 端点雏形**：
- `POST /api/admin/icons/fetch` body: `{url}` → 返回 `{path: "public/icons/abc.png"}`
- `GET /uploads/public/icons/:filename` 静态服务（已有 uploads 路径，扩展即可）

## How to apply

- 任何引入 `<img src="远程 URL">` / `<script src="远程">` / `<link href="远程">` 的代码必须有 fallback 或自托管
- code review 时看到 `https?://` 在前端代码里的硬编码，第一反应：**这玩意会失败，fallback 在哪？**
- Phase 3 一开始设计 icon library 时按 proxy+cache 落地，不要先做"直接渲染 URL"再回头改
