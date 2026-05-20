---
name: 视觉 silent 渲染失败必须量化检测 (不能靠 "@media 已经写了")
description: Mobile / layout / component import / @media 改动后必须代码层量化占宽 + 视觉门补 mobile project. 信任 "应该 OK" 导致 silent bug 累积多 release 无人发现 (v0.2.0 NInput 22-release 隐藏 bug, v0.2.24 mobile 320 title 0px).
type: feedback
---

# 视觉 silent 渲染失败必须量化检测

## 背景

v0.2.24 Phase B Flag #34 — Cards.vue admin 页 mobile 320 viewport 下 title 可
用宽 0px (完全压扁), 用户看不到卡名. Claude Code 代码层量化占宽 (固定元素
272px) 才 surface. Playwright `admin-cards` mobile project 长期 skip
(readme-screenshots.spec.ts:330), 截图未检测. Bevan daily 用 PC, mobile 真机
访问 admin 页频率低 → 长期 silent bug.

v0.2.0 同类: NInput 全工程无显式 import, 但 emit 无 error, 浏览器渲染为
unknown element (0×0), 22 release 无人发现.

## 错误模式

- 信任 "@media 已经写了, mobile 应该 OK"
- 信任 "playwright project 已经跑, mobile 没问题"
- 信任 "我用 PC 没看到问题, 应该没事"
- 信任 "type-check + vitest pass = UI 正常"

v0.2.0 NInput 案例: SiteSettings.vue template 写 `<NInput v-model="...">`,
没 `import { NInput } from 'naive-ui'`. emit clean, type-check pass, vitest
pass, 浏览器渲染 0×0 unknown element. 22 release 无人发现 "站点名称" 输入框
silently broken.

v0.2.24 Cards.vue 案例: mobile @media 只隐 `group-tag` + `sort`, 没考虑
dual-URL icons (60px) + actions (110px) 仍占 170px, 320 viewport title 无空间.

## 正确做法

1. 代码层量化先行:
   - 列出固定占宽元素 (padding + handle + thumb + gap + icons + actions)
   - 计算 `viewport - 固定 = title 可用宽`
   - 推断问题视口 (320 / 375 / 414)

2. 视觉门补 mobile project:
   - Playwright 别 `test.skip(info.project.name === 'mobile')` 关键 admin 页
   - 截图 320 / 375 / 414 三档
   - 视觉门检查 title 可见 + actions 不压

3. 缺失检测的 selector 加回归 (selector 缺失即 silent bug 温床)

## 适用范围

- 任何视觉 / layout / mobile 改动
- 任何 component import 后渲染未 visual check
- 任何 @media query 改动后 mobile project 仍 skip
- 触发关键词: silent, 0×0, unknown element, 看不到, 压扁, viewport 不够

## 关联 reference

- CLAUDE.md "Visual rendering" section (onboarding checklist)
- @memory/feedback_mobile_layout_total_width_audit.md (量化总宽 vs viewport 方法论)
