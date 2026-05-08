---
name: E2E 测试避免第三方 internal class (v0.2.16 Patch B 反思)
description: e2e test 优先项目自有 BEM, 不依赖 NaiveUI internal class (.n-*); 写新测试 + 改测试 + 维护时机纪律
type: feedback
---

# E2E 测试避免第三方 internal class (v0.2.16 Patch B 反思)

## 教训
E2E test 应该用 ★ 项目自有 BEM class ★, 避免依赖第三方组件库 internal class (.n-data-table-tr / .n-button / .n-input 等 NaiveUI 内部 class).

**Why**: 第三方 internal class 是组件库实现细节, 升级 NaiveUI 版本 / 替换组件库时, internal class 可能变 / 删. e2e test 跟第三方 internal class 耦合 = 跟组件库版本耦合. 自有 BEM (.cards-list__item / .groups-list__item / .engines-list__item) 是项目稳定 contract.

**How to apply**: 写新 e2e test 优先自有 BEM. 改 e2e test 看到 .n-* 时检查产品代码自有 BEM 包裹, 没有则加 BEM (产品改 + e2e 改同 commit). 维护时机不强制 — 按风险/工作量 trade-off.

## 根因
- 第三方 internal class 是组件库实现细节
- 升级 NaiveUI 版本 / 替换组件库时, internal class 可能变 / 删
- e2e test 跟第三方 internal class 耦合 = 跟组件库版本耦合
- 自有 BEM (跟 v0.2.15+ 累积一致) 是项目 ★ 稳定 contract ★

## 修法纪律

### 写新 e2e test 时:
- 优先用项目自有 BEM (.cards-list__item / .groups-list__item / .engines-list__item)
- 不得已用第三方 internal class 时, 加 comment 说明 + 加候选自有 selector

### 改 e2e test 时:
- 看到 .n-data-table-tr / .n-* 等第三方 internal class
- 先看产品代码是否有自有 BEM 包裹 → 用自有 BEM
- 没有自有 BEM 包裹 → 加 BEM (产品改 + e2e 改同 commit)

### 维护时机:
- audit-logs/security/audit/etc 7 处 .n-data-table-tr 仍引用 (v0.2.16 ship 时严守不动 scope)
- v0.2.17+ 可单独 chore 迁移 (跟 v0.2.16 P0 一起或独立 P0 候选)
- 不强制必须立刻迁移 (按风险/工作量 trade-off)

## 实例反思

v0.2.16 Patch B fix:
- phase-4b/3c-2 .n-data-table-tr → .engines-list__item (✓ 修了 v0.2.16 影响范围)
- audit-logs/security 7 处 .n-data-table-tr 不动 (v0.2.16 影响范围外, 严守)

下次 v0.2.17+ 类似情况:
- 不要扩大 scope (跟 v0.2.16 严守教训一致)
- 但累积"未来可迁移"清单 (e.g. memory/migration_candidate_e2e_third_party_class.md)

## 应用历史
- v0.2.16 Patch B 主动严守 scope (audit-logs/security 7 处不动)
- v0.2.17 chore memory (本次) 把教训内化

## 不要重犯
- 不要预先迁移所有第三方 internal class (扩 scope, 跟当前 release 无关)
- 不要忽视 .n-* 引用 (Task 1 grep 必须 audit, 应用 V2 教训)
- 不要写新 e2e test 时随意用 .n-* (优先自有 BEM)
