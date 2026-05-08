---
name: Spec 决策 default 保守倾向教训 (v0.2.15 Patch 1)
description: Task 1 候选给 Bevan 时不 default 推保守, 让 Bevan 从 daily use UX 视角拍板; v0.2.15 P0 a B.1a→B.1b 修正实例
type: feedback
---

# Spec 决策 default 保守倾向教训 (v0.2.15 Patch 1)

## 教训
Task 1 候选给 Bevan 拍板时, 工程师推荐倾向于"保守 / scope 不扩"的选项,
但 Bevan daily use 期望可能跟"保守"选项有差距, 导致视觉门多 1 patch.

**Why**: v0.2.15 P0 a B.1a (推荐, 保守 per-group 拖) → Bevan 视觉门反馈"卡片要从一个分组直接拖到另一个分组" → Patch 1 修正为 B.1b (跨分组). 同样 v0.2.13 P0 b 也有类似 default-bias 教训.

**How to apply**: 给候选时不仅给"推荐 + 工程理由", 还要明确"各候选对 daily use 的实际影响 + workaround 是否便利", 让 Bevan 从用户视角直接选.

## 根因 (v0.2.15 Task 1 B 决策实例)

Task 1 P0 a 候选:
- B.1a (推荐): per-group 拖, 跨分组用编辑表单 (跟 v0.2.13 CardsSortModal 一致, 保守)
- B.1b: 跨分组拖 (backend 已支持, UX 复杂)

我 (工程师) 推荐 B.1a 的理由:
- 跟 v0.2.13 一致 (累积模式)
- scope 不扩 (~30-50 行 vs B.1b ~80-100 行)
- 视觉门 patches 预算可控

Bevan 真实期望 (视觉门反馈):
- 卡片应可直接从一个分组拖到另一个分组 (B.1b)
- 编辑表单切分组太麻烦 (我假设的 workaround 实际不便)

结果: Task 2 实施 B.1a → 视觉门反馈 → Patch 1 修正为 B.1b (~30 min Claude Code 时间)

## 应用纪律 (Task 1 候选阶段)

### 候选给 Bevan 时:
- 不仅给 "推荐" + 理由, 还要明确说 ★ 各候选对 daily use 的实际影响 ★
- 不假设 Bevan 偏好 "保守 / scope 控制"
- 让 Bevan 从 daily use 视角直接选 (UX 期望优先)

### 推荐措辞:
- 旧: "B.1a 推荐 (跟 v0.2.13 一致)"
- 新: "B.1a (保守, 跨分组用编辑表单切换) vs B.1b (直接拖, +30-50 行 +1-2 patches), 你 daily use 期望选哪个?"

### 不"默认推荐保守" 的判断维度:
1. 工作量差距 (B.1a vs B.1b): 30-50 行 ≈ 等价 (保守不省多少)
2. patches 预算 (1-2 个 vs 0): 健康范围内
3. daily use UX 差距: ★ 这个最重要 ★ (workaround vs 一步到位)
4. backend 已支持? (v0.2.15 reorderCards group_id? 已支持) → 不是技术债

当 1+2+4 都不阻塞时, 优先 ★ daily use UX 期望 ★ 选项.

## 应用历史
- v0.2.13 P0 b 类似教训 (我假设"输入数字"指 sort modal, 实际指 NInputNumber)
- v0.2.15 P0 a 视觉门修正 (B.1a → B.1b)

## 不要重犯
- 不要因 "scope 控制" 默认推保守 (大部分 patches 都在视觉门修补这个 gap)
- 不要假设 Bevan 工程纪律倾向跟我一致 (Bevan 优先 daily use UX)
- 多用 "你 daily use 期望选哪个?" 让 Bevan 从用户视角拍板
