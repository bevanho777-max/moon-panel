---
name: Spec 重发主动识别教训 (v0.2.13 Patch 3)
description: 收到 spec 时先 grep 验证当前状态, 不盲目重跑. 如发现已实施完整, 主动报告等指令
type: feedback
---

# Spec 重发主动识别教训 (v0.2.13 Patch 3)

## 教训
收到 spec 时, 必须先 grep / view 验证当前状态, 而不是盲目按 spec 重跑.
如果发现 spec 已实施完整, 主动报告"等明确指令", 不擅自重跑.

**Why**: v0.2.13 Patch 3 实例: Bevan 因消息混乱重发 Patch 3 spec (与上一轮 100% 一致). Claude Code 主动 grep 7 处关键改动 (renderIconThumb size 参数 / lucideSize / mobile 调用 / 4 处 CSS), 确认全在, 主动报告"上一轮已实施完整, 等明确指令". 节省冗余 build/IO + 防止误改风险.

**How to apply**: 收到任何 spec 第一步先 grep 关键改动点; 全一致 → 报告"已实施"; 部分一致 → 询问"是 spec 修订还是补做?"; 全新 → 正常实施.

## 根因 (v0.2.13 Patch 3 实例)
- Bevan 因消息混乱重发 Patch 3 spec (跟上一轮 100% 一致)
- 如果盲目按 spec 重跑: 浪费 build/IO + 临时文件 + 多余 git diff noise
- Claude Code 主动 grep 7 处关键改动 (renderIconThumb size 参数 / lucideSize / mobile 调用 / 4 处 CSS),
  确认全在, 主动报告"上一轮已实施完整, 等明确指令"

## 收益
- 节省冗余 build (~7-8s × 1 = 8s, 累积多次更可观)
- 节省 robocopy /MIR (1-2s)
- 节省 type-check (~3s)
- 主要收益: 防止误改风险 (盲目重跑可能 unstable, 比如 Cards.vue 接口已扩展 size 参数, 重跑可能覆盖)
- 工程师精神体现: 不擅自动作, 等明确指令

## 应用纪律

收到 spec 后第一步 (任何 spec):
1. grep 关键改动点 (e.g. CSS class / function 名 / 数值变化)
2. 如果 spec 内容跟当前状态 100% 一致 → 主动报告"已实施完整"
3. 如果 spec 部分一致 → 报告"X 处已 Y 处未, 是 spec 修订还是补做?"
4. 如果 spec 全新 → 正常实施

## 不要重犯
- 不要假设 spec 一定是新工作
- 不要因为 Bevan 重发就一定要"做点什么"
- "状态识别 + 主动报告" 是工程师职责
