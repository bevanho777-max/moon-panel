---
name: 字面文本替换前必先征询，不擅自决定
description: spec 里 user 给的字面字符（Unicode/符号/commit msg/tag annotation/配置值）任何替换都要先停下问 Bevan，不因技术风险擅自改
type: feedback
---

任何 spec 里 user 给了字面文本（特殊字符 / Unicode / commit message / tag annotation / 配置值），即使我有合理技术理由（如 PowerShell 5.1 native exe arg encoding 风险）也要**先报告并问 Bevan**，再实施。绝不擅自替换。

**Why:** v0.2.7 tag annotation 里 spec 写 `phones (≤768px)`，我担心 PS 5.1 OEM codepage 把 `≤` mangle，**未先确认就替换为 `<=`** 推到 origin。但 v0.2.5/2.6 commit body 含中文字符顺利落库——证明编码链路其实通的，根本不必替换。Bevan ship 后才发现偏离，虽接受 `<=`（数学等价、tag 是元数据、重 tag 代价大）但明确说："你主动 flag 偏离对，但字符替换应先报告再实施。即使技术原因合理，决策权属于 Bevan。"

**How to apply:**
- spec 文本里任何**字面层面**疑虑（不能确定 1:1 复现）→ **先停下问，给替代方案 + 风险评估，等 Bevan 回复**，不动手
- 即便技术理由合理，也只是建议，不是授权
- "事后主动 flag 偏离" 是补救，**不能替代事前征询**
- 适用范围：commit message、tag annotation、配置 value、文档原文、文件名等任何 user 字面给定的文本
- 实操判断：如果我在写命令时心里冒出"这个字符可能有问题，我替换一下"——立刻停手发问，不要替换
