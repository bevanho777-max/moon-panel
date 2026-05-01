---
name: 前端 localStorage key 命名规范
description: 所有 localStorage key 用 moon. 前缀 + 点分隔子分类，便于隔离/调试/批量重置
type: feedback
---

前端写 localStorage（或 sessionStorage）时，所有 key 必须用 **`moon.`** 前缀，子分类用**点**分隔。

例：
- `moon.network.global` —— 全局网络模式（auto / internal / external）
- `moon.network.overrides` —— 单卡内/外网覆盖（JSON）
- `moon.ui.theme` —— 主题（Phase 4）
- `moon.search.engine` —— 当前搜索引擎 id（Phase 3）

**Why（用户 2026-04-28 明示）：**
1. **避开命名冲突**：moon panel 未来可能被 iframe 嵌入到其他页面。同 origin 共享 localStorage，不带前缀的 key（如 `theme`、`network`）几乎一定撞车。
2. **调试可识别**：用户在 DevTools Application → Local Storage 一眼分辨哪些是 moon panel 的 key。
3. **支持批量重置**：未来"重置所有用户偏好"功能可以 `Object.keys(localStorage).filter(k => k.startsWith('moon.'))` 一键清。

**How to apply:**
- 每次新增 localStorage key 都要：以 `moon.` 起头、一级分类（`network` / `ui` / `search` / `card` / `auth`...）、可选二级分类
- 不要用 camelCase 或下划线（`moonNetworkGlobal` / `moon_network_global` 都不要），统一点分
- 反例：`token`、`theme`、`networkMode`、`Moon-Panel-State`
- 写读建议封装到一个 `utils/storage.ts` helper（非强制，但能集中校验前缀）
- sessionStorage 同样规则
