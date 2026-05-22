---
name: Bevan 报告输出格式约定 (点击复制块)
description: 每个 Phase / 门 / Task 的报告末尾追加 ``` 代码围栏包住的精简版, 给 Bevan 点 1 下复制按钮整块拷给 Opus, 不用鼠标划选
type: feedback
---

## 约定 (Bevan 2026-05-22 长期工作约定)

每个 Phase / 门 / Task 的报告输出 = 2 层:

1. **正常详细报告** (Markdown 自由格式, 给 Bevan 看实施/验收过程)
2. **末尾追加点击复制块** (单个 ``` 代码围栏包整份精简报告)

## 格式模板

```
（正常报告 — 详细 verbatim / diff / 验收 / flag, 给 Bevan 看）

...

===给 Opus 的报告 (点击复制)===

` ` ` (此处替换为三个反引号开始)
# <Phase / 门 / Task 名> 报告

## 关键结论
- hash: <commit hash>
- 验收: <绿/红, 时长>
- flag: <#编号 或 无>
- 状态: <下一步等什么>

## diff stat (如有)
- ...

## 备注 (任何 Opus 必须知道的非显然项)
- ...
` ` ` (此处替换为三个反引号结束)
```

## Why

- Bevan 工作流: Claude Code 实施 → Bevan 点复制 → 贴 Opus 决策
- 鼠标划选选不到代码块尾 + 易选漏内容 + 桌面 / 移动通用差
- 代码块右上角自带"复制全文"按钮, 1 click 100% 精确
- 详细版给 Bevan 看, 精简版给 Opus 看 (两者各自最优)

## How to apply

- **每次** 给 Bevan 完整报告时都用此格式 (不仅 Phase 报告, 还含门 1/2/3 报告, Task 0 综合报告等)
- 代码围栏内**不能再嵌入 ```** (否则提前闭合), 嵌入代码示例改用缩进 4 空格或单反引号
- 精简版**只**含 Opus 决策需要的: hash, 验收, flag, 状态. **不**含 mockup, 思考过程, 详细 diff
- ★ "点击复制" 标题别变, Bevan 识别这个 anchor ★
- 短任务 (1-2 步小修) 不强制, 但凡报告超 3 段就加

## 反面案例

- ❌ 用 4 空格缩进代替围栏 (复制按钮不出)
- ❌ 围栏内有 ```typescript 嵌套围栏 (提前闭合)
- ❌ Opus 块跟详细报告内容 100% 重复 (浪费 Opus context, 应该是精简)
- ❌ 漏 "===给 Opus 的报告 (点击复制)===" 这个 anchor 行

## 跟其他 feedback 联动

跟 [[feedback_visual_gate_iter_healthy]] / [[feedback_chore_commit_path_strategy]] 互补:
报告内容 (验收 + flag + hash + 状态) 不变, 只是末尾多挂一个复制块.
