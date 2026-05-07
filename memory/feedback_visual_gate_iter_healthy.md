---
name: 视觉门 patch 迭代健康模式 (v0.2.13 累积)
description: 视觉门 patch 必须每个对应 Bevan 真实反馈, 不能完美追求. 健康范围 1-5 patches; 6+ 必有 spec 误判
type: feedback
---

# 视觉门 patch 迭代健康模式 (v0.2.13 累积)

## 教训
视觉门 patch 必须每个对应 Bevan 真实反馈, 不能"完美追求".
Patch 数量本身不是问题, 关键是每个 patch 都有具体反馈驱动.

**Why**: v0.2.13 视觉门 3 patches 全 Bevan 反馈驱动 (上下空白 → 一行 → 缩小), 无回退无 scope 蔓延. v0.2.10 7 patches 含 spec 误判的 wrap-vs-alignment 回退, 是反例.

**How to apply**: 视觉门反馈处理时, 每个 Bevan 反馈先量化 (高度/字号/像素), 给 2-4 候选 + 推荐, 不擅自加额外 patch. 累积 4+ patches 时停下 audit 是否 scope 蔓延.

## 健康模式 (v0.2.13 实例)

v0.2.13 视觉门 3 patches:
- Patch 1: Bevan 反馈"上下空白多, 显得臃肿" → 扁平化 (admin Cards mobile + PC HomeCard)
- Patch 2: Bevan 反馈"可以直接一行吗?" → 1 行布局 (mobile 极致紧凑)
- Patch 3: Bevan 反馈"图标和字样均可以再缩小一些" → icon 22 + font 0.85rem

★ 每个 patch 都对应明确反馈 ★, 没有"完美追求" 浪费

## 反例对比 (v0.2.10 教训)

v0.2.10 ship 时 7 patches, 含 spec 误判 (wrap-vs-alignment) 的回退 patch.
跟 v0.2.13 对比:
- v0.2.10: 7 patches (含 spec 误判 + 反复推翻)
- v0.2.13: 3 patches (全 Bevan 反馈驱动, 无回退)

## 数量参考 (v0.2.x 累积)

| 版本 | patches | 类型 |
|------|---------|------|
| v0.2.6 | 1 | 简单 mobile UX |
| v0.2.7 | 0 | 简单 hotfix |
| v0.2.8-2.11 | 1-2 | 中等 scope |
| v0.2.10 | 7 | ★ 失控反例 ★ (spec 误判 + 回退) |
| v0.2.12 | 5 | 健康 (marquee 优化迭代) |
| v0.2.13 | 3 | 健康 (视觉密度 3 轮收紧) |

★ 健康范围: 1-5 patches (反馈驱动) ★
★ 失控阈值: 6+ patches 必有 spec 误判 / 不必要 patch ★

## 应用纪律

视觉门反馈处理:
1. 每个 Bevan 反馈先量化 (e.g. 高度 ~50px → ~40px), 不主观判断
2. 给 2-4 候选方案 + 推荐, 不擅自决定
3. 实施前确认 spec 跟反馈一一对应, 没有 scope 蔓延
4. 不主动加额外 patch (除非真有 bug)
5. 累积 4+ patches 时停下 audit, 是否 spec 误判 / scope 蔓延

## 不要重犯
- 不要"完美追求" (e.g. "顺手再缩小点" 不在反馈范围)
- 不要 spec 误判 → patch 推翻 (v0.2.10 教训)
- 不要因为"上次也加了"就主动加 patch
