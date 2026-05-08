---
name: 累积教训 删组件→CI fail→chore 修 模式 (v0.2.15 + v0.2.16)
description: 连续 2 个 release 同模式: feat 删组件 → CI 1st fail → chore 修 e2e selector → CI 2nd success → tag; V1 教训不彻底 → V2 修
type: feedback
---

# 累积教训: 删组件 → CI fail → chore 修 模式 (v0.2.15 + v0.2.16)

## 模式
连续 2 个 release 累积同一模式:
- 主 feat commit 删除组件 / UI element
- CI 1st run e2e test fail (test 仍引用待删元素)
- chore commit 单独修 e2e (selector swap / 整删)
- CI 2nd run success
- tag + release.yml ship

**Why**: V1 教训 (`feedback_e2e_test_sync.md`) 不彻底, 仅 grep 自有 BEM, 漏第三方 internal class. v0.2.15 + v0.2.16 都因此 CI 1st fail. v0.2.17+ 应用 V2 教训应该一次到位.

**How to apply**: Task 1 grep 阶段彻底 (V2 5 类 selector). Task 2 spec 明确 product + e2e 同 commit 修. 不让模式再重复.

## 实例

### v0.2.15 (commit b7fc394 → 74a1a90)
- feat: 删 CardsSortModal.vue + "调整顺序" button (X3 inline drag 替代)
- CI 1st: phase-4a.spec.ts:162 06-cards-sort-modal timeout (找不到 button)
- chore: 整删 06 测试块 + 注释保留 v0.2.16 待删 07 路标
- CI 2nd: 2m24s success ✓

### v0.2.16 (commit 0f0ba54 → aa7743d)
- feat: 删 NDataTable in Groups + Search Engines (X3 inline drag 替代)
- CI 1st: phase-4b/3c-2 共 10 tests timeout (找不到 .n-data-table-tr)
- chore: swap selector .n-data-table-tr → 自有 BEM (跨 2 文件)
- CI 2nd: 1m51s success ✓

## Root Cause

- feedback_e2e_test_sync.md V1 grep 阶段不彻底
  - V1: 仅 grep 自有 BEM (.cs__/.gs__/.es__)
  - 漏: 第三方 internal class (.n-data-table-tr) / UI text / 别的
- V2 升级 (feedback_e2e_test_sync_v2.md) 修复 grep 范围

## 避免再蹈

### Task 1 grep 阶段必加 (V2 实施):
- 第三方 internal class (NaiveUI 内部)
- UI text (hasText)
- data-* attributes
- role / aria selectors
- 自有 BEM (V1)

### Task 2 spec 阶段必明确:
- 同 commit 修 product + 修 e2e (跟 v0.2.16 教训一致)
- 不分 separate chore commit (除非应急, 跟 v0.2.15 Patch A 是补救)

## 应用历史
- v0.2.13: 0 e2e fail (没删组件, 仅改样式)
- v0.2.14: 0 e2e fail (admin header 改, 没删组件)
- v0.2.15 Patch A: 06 e2e fail → 单独 chore 修 (★ 第一次 ★)
- v0.2.16 Patch B: 10 tests fail → 单独 chore 修 (★ 第二次, 教训未彻底应用 ★)
- v0.2.17+: 应用 V2 一次到位 (★ 期望 ★)

## 不要重犯
- 不要假设 V1 教训足够 (V2 才彻底)
- 不要 amend 主 commit 修 e2e (跟 chore commit strategy 一致)
- 不要在 chore commit 后立刻 tag (等新 CI 全绿才 tag)
