---
name: E2E 测试同步教训 (v0.2.15 Patch A)
description: Spec 阶段删 UI element 时必 audit e2e test 引用, 否则 CI fail (TimeoutError); Task 1 grep 阶段必加 e2e test pattern search
type: feedback
---

# E2E 测试同步教训 (v0.2.15 Patch A)

## 教训
Spec 阶段删除组件 / UI element 时, 必须同步 audit e2e test 是否引用该组件,
否则 CI 失败 (test stale, 找不到 button / class selector / hasText).

**Why**: v0.2.15 主 commit b7fc394 删了 CardsSortModal.vue + "调整顺序" 按钮, 但 phase-4a.spec.ts:162 06-cards-sort-modal 仍引用 → CI Run 25535307961 failed (TimeoutError 10000ms). 单独 chore commit 74a1a90 才修复.

**How to apply**: 任何 spec 内含"删除组件 / UI element" 的 P0 / Task, Task 1 grep 阶段必加 frontend/tests/e2e/ 目录搜索, Task 2 spec 必含"同步 e2e test" 子任务.

## 根因 (v0.2.15 实例)
- v0.2.15 Task 2 spec 要求删 CardsSortModal.vue + "调整顺序" 按钮 (X3 替代)
- phase-4a.spec.ts:162 06-cards-sort-modal 仍引用:
  * `await page.locator('button', { hasText: '调整顺序' }).waitFor(...)` → 找不到
  * `await page.waitForSelector('.cs__groups')` → CardsSortModal 已删, class 不存在
- CI Run 25535307961 failed: TimeoutError 10000ms exceeded

## 修法 (v0.2.15 Patch A)
- 单独 chore commit `chore(test): remove obsolete 06-cards-sort-modal e2e test`
- 删 06 测试整块 (-22+3 注释保留 v0.2.16 待删 07 路标)
- 不 amend v0.2.15 主 commit (保持历史清晰)
- 新 CI Run 25535643539 success 2m24s

## 应用纪律 (下次 spec 阶段)

### Task 1 grep 阶段必加项:

```bash
# 找 e2e test 引用待删组件
Get-ChildItem -Recurse frontend/tests/e2e -Include *.ts,*.js |
  Select-String -Pattern "(待删 component name|UI text|CSS class)"
```

### Task 2 spec 阶段必添:
- 明确"同步更新 e2e test" 子任务
- 列出受影响的 test 文件 + 行号
- 决定: 删除 vs 改写 (跟 daily use 验证一致选 + Bevan 拍板)

### 实施时:
- product 改 + e2e test 改 ★ 同 commit ★ (避免 CI fail 后单独修)
- 或: product 改在主 commit, e2e test 改在紧跟的 chore commit (Bevan 拍板)

## 历史失败案例
- v0.2.15 b7fc394 (X3 弃 CardsSortModal): CI failed, Patch A 修
- 未来 v0.2.16 (待删 GroupsSortModal): 必须同步删 07-groups-sort-modal e2e test (74a1a90 已埋路标)

## 不要重犯
- 不要假设 e2e test 跟产品代码自动同步 (Vue scoped CSS class 改了 test 不知)
- 不要在 Spec 阶段忽略 frontend/tests/e2e/ 目录 (跟 frontend/src 同等重要)
- 不要 amend 主 commit 修 e2e (混淆 product 改动 + test 改动)
