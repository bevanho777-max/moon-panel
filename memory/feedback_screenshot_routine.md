---
name: 前端子阶段交付必须含 Playwright 截图（Phase 2.4 起）
description: 每个 frontend 子阶段交付时跑 e2e 截图，桌面+移动各一组，省用户验收时间到 ~10 分钟
type: feedback
---

## 规则

从 **Phase 2.4 起**，每个 frontend 子阶段交付**必须**包含 Playwright e2e 截图。目的是把用户验收时间从 30 分钟压到 ~10 分钟（看图 2 分钟 + docker build 5 分钟 + 主页扫一眼 2 分钟）。

**用户主动同意**新增依赖（违反纪律 #4）以换取验收负担减轻——这是双方协商一致的例外，**仅限 Playwright 这一项**。

## 一次性 Setup（Phase 2.4 第一次跑时做）

```bash
cd C:\moon-build\frontend
npm i -D @playwright/test
npx playwright install chromium --with-deps
```

**关键约束**：
- `@playwright/test` 必须装在 **`C:\moon-build\frontend\node_modules`**（F-lite 副本），**永远不在 `P:\moon-panel\frontend\node_modules`**
- `package.json` / `playwright.config.ts` 等**配置文件**正常写在 P，由 robocopy 同步到 C
- node_modules 永远只在 C，避免重蹈 SMB esbuild 覆辙

## package.json scripts

```json
{
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest",
    "test:e2e": "playwright test",
    "test:e2e:headed": "playwright test --headed"
  }
}
```

`test:e2e:headed` 给我（或用户）调试时看真浏览器实际行为。

## 每子阶段执行流程

1. F-lite 现有流程不变：robocopy + npm test (vitest) + npm run build
2. **新增**：在 P:\moon-panel\frontend\tests\e2e\ 下写 `phase-X.Y.spec.ts`
3. robocopy 同步过来后在 C: 跑 `npm run test:e2e`
4. Playwright 配置 webServer 启动 vite preview，加上 page.route() mock 后端 API
5. 截图保存到 `P:\moon-panel\screenshots\phase-X.Y\<viewport>\<NN>-<descriptor>.png`
6. 交付文档加"截图索引"小节

## 截图覆盖要求

每个子阶段必须覆盖：
- 每个**新增/修改的 UI 路由** 各一张桌面图（1280×800）
- **关键路由**再各一张移动图（iPhone 14 Pro Max，430×932）
- **核心交互**各截一张（点编辑弹窗、点搜索栏、点切换器、右键菜单等触发后的状态）

## 命名规范

```
P:\moon-panel\screenshots\phase-X.Y\
├── desktop\
│   ├── 01-home-with-cards.png
│   ├── 02-home-empty-state.png
│   ├── 03-network-switcher-open.png
│   └── ...
└── mobile\
    ├── 01-home-with-cards.png
    └── ...
```

- 编号 `NN` 与浏览器验收清单的项号对应
- `<descriptor>` 用人话标识，避免缩写到看不懂
- 桌面 / 移动 同 NN = 同场景不同 viewport

## 交付文档格式

每次 frontend 交付的 delivery doc 必须包含：

1. **改动文件列表**（保持现状）
2. **本地验证证据**（保持现状：vitest pass / vite build pass / e2e pass）
3. **你跑的 docker build 三连**（保持现状）
4. **浏览器验收清单**：每条带截图编号，例：
   ```
   | # | 操作 | 预期 | 截图 |
   |---|---|---|---|
   | 2 | 进 / 公开主页 | 看到分组+卡片网格 | desktop/02 + mobile/02 |
   ```
5. **截图索引**（新增小节）：
   ```
   截图位置：P:\moon-panel\screenshots\phase-X.Y\
   - 图 1 (desktop/01-cards-table.png) → 验收项 #2 表格 5 列
   - 图 2 (desktop/02-edit-modal.png) → 验收项 #6 编辑弹窗预填
   - 图 3 (mobile/01-home.png) → 验收项 #11 移动端响应式
   ...
   ```

## screenshots/ 目录管理

- **不入 git**：`.gitignore` 已加 `/screenshots/`
- 旧 phase 的截图保留方便回看；用户手动清不需要的

## 边界 / 限制

- **Chromium-only**：iOS Safari、Firefox 行为差异截图测不到
- **静态截图**：长按、复杂动画行为不可见——必须**用户在群晖真容器跑**兜底
- **API mock**：测试用 `page.route()` 拦截 mock，不真请 Go backend——后端集成验证靠用户的 docker build + curl
- **种子数据要有视觉差异**：双 URL 状态指示要建一张只内网、一张只外网、一张双有的卡，覆盖三种视觉态——不能只放一张普通卡敷衍

## 不能省的部分

截图取代的是"用户开浏览器手点 13 项 UI 验收"那部分。**不取代**：
- docker build 在群晖真跑（PUID/PGID、镜像源、容器健康都靠这个验）
- 浏览器开真主页扫一眼（截图是 mock 数据 + Chromium，真容器是 Go backend + 真浏览器）
- HTTP API 的 curl 验证（任何 backend 改动）

用户每 Phase 验收三步：看截图 2 分钟 + docker build 5 分钟 + 真主页扫一眼 2 分钟 ≈ **10 分钟总**。

## How to apply

- Phase 2.4 第一次跑时执行 setup（已用户授权）
- 之后每个 frontend 子阶段交付前生成截图
- 截图作为视觉佐证，不替代真容器验证
- 黑屏/缺图标/布局错位的截图**不交付**，先修
