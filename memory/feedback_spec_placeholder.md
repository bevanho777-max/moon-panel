---
name: spec 边界值必须用真实预期 (不要 placeholder)
description: spec 涉及 form 默认值 / 状态比较 / 空值判断时, 必须 grep 验证实际 emptyForm / API default / DB default 值, 不靠记忆推测. 否则 watch / 验证 / 边界条件实施后立刻触发 bug, 跟决策直接矛盾.
type: feedback
---

# spec 边界值必须用真实预期 (不要 placeholder)

## 背景

v0.2.24 Phase B Flag #33 — Claude 写 NRadioGroup watch handler spec 时, 假设
"URL 为空" 边界是字面空字符串 (''). 实际 Cards.vue emptyForm 预填 `'http://'` /
`'https://'` protocol prefix (新表单 UX, v0.2.21 P4a 引入). Task 0 grep 时主动
flag — 否则 watch 在新表单打开瞬间立刻 trigger, 自动选 internal, 直接破 D1
"双空时 radio 不选" 决策.

## 错误模式

```ts
// ❌ 错: 假设 emptyForm.url_internal === ''
watch(() => editorForm.value.url_internal, (v) => {
  if (v) {                              // 'http://' 也算 truthy
    editorForm.value.url_default = 'internal'
  }
})
```

新表单一打开, watch 立刻 trigger, url_default 被强制设 'internal' — 跟"双空时
不选"决策矛盾.

## 正确做法

复用 codebase 已有判断逻辑 (Cards.vue:293-298 submit handler 内 "实质空" 判断
式), 不重新发明边界:

```ts
// ✓ 对: 复用 isUrlSubstantiallyEmpty
const isUrlSubstantiallyEmpty = (v: string): boolean => {
  const trimmed = (v || '').trim()
  return trimmed === '' || trimmed === 'http://' || trimmed === 'https://'
}

watch(() => editorForm.value.url_internal, (newVal) => {
  if (userPickedUrlDefault.value) return
  const intEmpty = isUrlSubstantiallyEmpty(newVal)
  // ...
})
```

## 适用范围

- spec 涉及 form 默认值 / 状态比较 / 空值判断
- 任何 spec 假设 "字段值是 X" 时, 必须 grep `emptyForm` / API default / DB
  default 验证, 不靠记忆推测
- 触发关键词: 默认值, 边界, 空值, "应该是 ''", "应该跟 X 相等"

## 关联 reference

- @memory/feedback_grep_before_spec.md (forward link, 待累积新建)
- @memory/feedback_unverified_prod_work.md
- @memory/feedback_no_unilateral_substitution.md
