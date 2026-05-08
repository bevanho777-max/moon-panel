---
name: E2E 测试同步教训 V2 (v0.2.16 Patch B 升级)
description: V1 grep 不彻底, V2 扩展到第三方 internal class + UI text + data-* + role/aria; supersedes feedback_e2e_test_sync.md
type: feedback
---

# E2E 测试同步教训 V2 (v0.2.16 Patch B 升级)

## 升级背景
v0.2.15 教训 feedback_e2e_test_sync.md 第一版只 grep 自有 BEM (.cs__/.gs__/.es__), 但 v0.2.16 删 NDataTable 时, e2e test 用了 ★ 第三方 internal class .n-data-table-tr ★ (NaiveUI 组件库内部 class), Task 1 grep 漏掉这类引用, 导致 CI Run 25554466731 仍 failed (10 tests).

★ 教训不彻底 ★ → V2 升级.

**Why**: 连续 2 个 release (v0.2.15 + v0.2.16) 都因为 e2e test 跟产品代码不同步导致 CI 失败 + chore commit 修. V1 仅 grep 自有 BEM 漏了第三方组件库 internal class.

**How to apply**: Task 1 grep 阶段 audit 5 类 selector (BEM + 第三方 internal + UI text + data-* + role/aria), 不仅自有 BEM.

## 升级核心: Task 1 grep 必须扩展到所有 selector 类型

### Spec 阶段 grep 必加项 (不再只 grep 自有 BEM):

```bash
# 1. 自有 BEM class (项目自有)
Get-ChildItem -Recurse frontend/tests/e2e -Include *.ts |
  Select-String -Pattern "(待删 component name|UI text|.cs__|.gs__|.es__)"

# 2. 第三方 internal class (NaiveUI / 别的组件库)
Get-ChildItem -Recurse frontend/tests/e2e -Include *.ts |
  Select-String -Pattern "(\.n-data-table-tr|\.n-data-table|\.n-button|\.n-input|\.n-modal)"

# 3. UI text (hasText)
Get-ChildItem -Recurse frontend/tests/e2e -Include *.ts |
  Select-String -Pattern "(hasText.*'调整顺序'|hasText.*'删除组件名')"

# 4. data-* attributes (如有)
Get-ChildItem -Recurse frontend/tests/e2e -Include *.ts |
  Select-String -Pattern "(data-test|data-cy|aria-label)"

# 5. role / aria selectors
Get-ChildItem -Recurse frontend/tests/e2e -Include *.ts |
  Select-String -Pattern "(role=|aria-)"
```

### 处理选项 (按情况):

- 自有 BEM 引用 → 同 commit 删 (跟 v0.2.15 06-cards-sort-modal 模式)
- 第三方 internal class → ★ 同 commit swap 自有 BEM ★ (跟 v0.2.16 .n-data-table-tr swap 模式)
- UI text → 同 commit 删 (e.g. "调整顺序" 按钮删时)
- data-* → 看情况 (产品改动是否动 attribute)
- role / aria → 通常不需删 (语义 selector 稳定)

## v0.2.16 Patch B 实例 (验证 V2)

v0.2.16 删 NDataTable 后:
- phase-4b.spec.ts (StatefulInput state machine): .n-data-table-tr → .engines-list__item
- phase-3c-2.spec.ts (site-settings list/edit): .n-data-table-tr → .engines-list__item
- 单独 chore commit aa7743d, paths 命中 .ts 触发新 CI
- 新 CI success 1m51s ✓

## v0.2.15 + v0.2.16 累积模式 (避免再次重蹈)

- v0.2.15: 删 CardsSortModal → 06-cards-sort-modal e2e fail → Patch A
- v0.2.16: 删 NDataTable → 10 tests fail (.n-data-table-tr stale) → Patch B

★ 模式 ★: 删组件 → CI fail → chore 修 selector → success

V2 教训核心: 不让这模式再重复 → Spec Task 1 grep 阶段彻底 audit 所有 selector 类型.

## 不要重犯
- 不要假设 e2e test 仅用自有 BEM (实际 NaiveUI internal 也常用)
- 不要 amend 主 commit 修 e2e (混淆 product 改动)
- 不要在主 commit 后立刻 push tag (CI 跑完才知)
