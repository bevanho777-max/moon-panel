---
name: Chore commit 路径策略 (v0.2.13/14/15 累积)
description: chore commit 不 amend 已 push 主 commit, 单独 commit 紧跟 feat; tag 指向 HEAD 含 chore; paths-ignore 决定是否触发 CI
type: feedback
---

# Chore commit 路径策略 (v0.2.15 累积 + Bevan 决策)

## 教训
Chore commit (memory / e2e test fix / docs) 不 amend 已 push 的主 commit,
而是单独 commit 紧跟在主 feat commit 后, 历史清晰且不需 force push.

**Why**: v0.2.13/14/15 累积 Bevan 一致决策"新 commit 不 amend" — 不改写历史 + 干净 + 单一职责. amend 需要 force push 风险高, 合并到 v0.X+1 ship 又混淆改动归属.

**How to apply**: chore commit 单独 push 紧跟主 feat; paths-ignore 命中 (memory/**, **/*.md, docs/**) 不触发 CI; 不命中 (tests/e2e/*.ts, src/) 触发 CI 必须等绿; tag 指向 HEAD 含所有 chore.

## 决策 (Bevan v0.2.14/2.15 累积)
- v0.2.13: chore(memory) 单独 commit 4f457e1 (paths-ignore 不触发 CI)
- v0.2.15 Patch A: chore(test) 单独 commit 74a1a90 (paths 命中 .ts 触发 CI)

Bevan 明确选项 (v0.2.14 决策):
- "新 commit (不 amend)" 推荐
- 理由: 不改写历史 + 干净 + 单一职责
- 不选: amend (force push 风险) / v0.X+1 ship 时合并 (混淆)

## 应用纪律

### 决策矩阵:

| 改动类型 | 文件路径 | paths-ignore 命中? | 触发 CI? | commit 时机 |
|----------|----------|-------------------|----------|-------------|
| memory .md | memory/**/*.md | ✓ 命中 | ✗ 不触发 | 立刻 push, 不影响 release |
| docs .md | docs/**/*.md | ✓ 命中 | ✗ 不触发 | 立刻 push |
| e2e test .ts/.js | tests/e2e/**/*.ts | ✗ 不命中 | ✓ 触发 | 必须 CI 全绿才 release |
| product code | frontend/src/**, backend/** | ✗ 不命中 | ✓ 触发 | feat / fix commit |

### chore commit msg 格式:
- `chore(memory): <topic>` — memory 整理
- `chore(test): <topic>` — e2e test 维护
- `chore(docs): <topic>` — 文档
- 不要混入 feat/fix prefix

### tag 策略:
- tag 指向 HEAD (含 chore commit)
- 跟主 feat commit + chore commit 都属于同一 release
- release.yml build 用 HEAD commit (含 chore 修复内容)

## 应用历史
- v0.2.13 commit 顺序: feat(v0.2.13) f920abe → chore(memory) 6106e74 → tag v0.2.13 指向 chore commit
- v0.2.14 commit 顺序: feat(v0.2.14) 8216179 → tag v0.2.14 指向 8216179 (无 chore)
- v0.2.15 commit 顺序: feat(v0.2.15) b7fc394 → chore(test) 74a1a90 → tag v0.2.15 指向 74a1a90

## 不要重犯
- 不要 amend 已 push commit (除非 CI 内部 fix, 跟 force push 谨慎)
- 不要把 chore 改动合并到 feat commit (混淆单一职责)
- 不要 tag 主 feat commit 跳过 chore (会失 chore 修复)
