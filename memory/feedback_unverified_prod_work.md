---
name: 不假设, 实际 grep / 跑 / 看 (spec 必须基于真实 codebase)
description: spec 起草前必须 grep / view 验证假设, 任何具体行号 / 路径 / 命名 / 行为必须 verbatim 引用. 凭记忆 / 推测 / "应该是" 写 spec 是 release 返工的主因. 累积纪律应用强制 Task 0 read-only grep 前置 + 实施前再 verify.
type: feedback
---

# 不假设, 实际 grep / 跑 / 看 (spec 必须基于真实 codebase)

## 背景

v0.2.24 本 release 累积 4 个 spec 假设错误 (全 Task 0 grep 阶段 surface):

- **#28**: spec 估 ci.yml 1 job, 实际 2 jobs 8 行
- **#29**: spec 假设 backend 有 `service/` 层 + `migration.go`, 实际无 (X2
  空字符串语义路线救场, 4 行 vs 30-50 行 + DB migration risk)
- **#32**: spec 写 `frontend/tests/cardUrl.spec.ts`, 实际
  `frontend/src/utils/__tests__/cardUrl.spec.ts`
- **#35**: spec 用 CSS single-line, codebase 用 multi-line (convention drift)

全部因为 Task 0 read-only grep 前置, 主动 flag → 0 擅自决定 → 0 实施返工.

## 错误模式

- 凭记忆写 spec ("应该有 service 层", "应该是 frontend/tests/", "应该是 1 行 CSS")
- 信任 "上次就是这样" / "其他项目就是这样" / "标准做法应该是"
- spec 引用具体行号但没验证 (例 "Cards.vue:115" 凭推测)
- 不 grep 直接锁定 type / 字段名 / API endpoint

```
❌ 错: "Cards.vue 加 mobile @media (推测 line 660)"
```

## 正确做法

1. ★ **spec 起草前** ★: 必须 grep / view 实际现状, 验证假设
2. ★ **spec 起草中** ★: 任何具体行号 / 路径 / 命名 必须 verbatim 引用
3. ★ **spec 起草后** ★: 任何 "我假设" 的边界必须 inline flag, 等 Bevan 拍板
4. ★ **实施前** ★: Task 0 read-only grep 前置 (强制), 主动 flag spec 假设错误
5. ★ **实施中** ★: 任何 spec 假设错误立刻 flag (累积 #X), 停止实施, 等拍板

```
✓ 对: "Cards.vue (Task 0.3 grep 验证 line 660-671) 现有 @media block 加 1 行"
```

## 适用范围

- 任何 spec 涉及现有文件 / 行号 / config / API / DB schema
- 任何 "我应该知道" 的本能, 必须二次 grep 验证
- 任何回忆模糊的细节, 必须 view 实际文件
- 触发关键词: 推测, 应该, 大概, 估计, 记得, 上次, 默认

## 关联 reference

- @memory/feedback_grep_before_spec.md (forward link, 待累积新建)
- @memory/feedback_no_unilateral_substitution.md
- @memory/feedback_patch_repeat_detection.md
