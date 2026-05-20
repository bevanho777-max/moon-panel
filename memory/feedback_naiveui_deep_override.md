---
name: NaiveUI :deep() override 改组件内部 BEM 的副作用
description: 改 .n-button__content 等 NaiveUI 内部 BEM 的 box 计算属性 (display, line-height) 会改变 outer width, 在 flex 父容器中触发 wrap. 安全做法: 在组件根类自身加 align, 不深入内部 BEM
type: feedback
---

# NaiveUI :deep() Override 副作用

v0.2.10 Task 2.13 实战发现: 给 NaiveUI 组件内部 BEM (`.n-button__content`)
加 `display: inline-flex; line-height: 1` 想居中 icon, 实际改了 NaiveUI button
内部 box 计算, 导致 outer width 微变, 在 NSpace flex 父容器内触发 wrap
(子项总宽超 viewport, 最右元素被挤下行).

## 现象

v0.2.10 Task 2.13 加规则 (mobile @media):
```css
.home-header :deep(.n-button.n-button--circle .n-button__content) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}
```

期望: NButton circle 内 icon vertical center.
实际: NSpace 内 Settings circle button 被 wrap 到第 2 行 (本来 4 元素第 1 行 OK).

## 根因

NaiveUI NButton 内部结构:
```html
<button class="n-button n-button--circle">
  <span class="n-button__content">
    <Settings />
  </span>
</button>
```

改 `.n-button__content` display + line-height:
- 默认 inline + 默认 line-height 让 button 整体高度跟其他元素 baseline 一致
- 改 inline-flex + line-height: 1 后 button 视觉高度变化 (微), 影响 box 计算
- NSpace 子项总宽计算时, button outer width 略增
- 总宽度从勉强 fit (294px ≤ 375 viewport) 变成超出, NSpace wrap=true 自然 wrap

## 安全做法

★ 不要在 :deep() 内改 NaiveUI 组件内部 BEM 的 box 属性 ★
(display, width, height, line-height, padding, margin, etc.)

★ 推荐做法 (按场景):

1. **改对齐**: 在 NaiveUI 组件根类自身改 (.n-button { align-items: center; })
   - 不影响内部 box 计算
   - 跟 NaiveUI 默认 box 兼容

2. **改外观**: 在 :deep() 内改 visual 属性 (color, background, border, border-radius)
   - 不影响 box 计算
   - 不会引发 layout 副作用

3. **改 layout 容器**: 在外层容器改, 不深入 NaiveUI 内部
   - 例: NSpace 加 align="center" / .n-space-item :deep() inline-flex (v0.2.10 Task 2.10)
   - 影响 NSpace 子项, 不动 NaiveUI 组件内部

## 反面 vs 正面案例

### 反面: v0.2.10 Task 2.13 (撤销)

改 `.n-button__content` display + line-height:
- 引发 NSpace wrap
- Task 2.14 撤销

### 正面: v0.2.7 AuditLog .n-card-header 改 flex-direction (安全)

改 `.n-card-header` flex-direction (不是内部 BEM, 是 NCard header 顶层布局容器):
```css
:deep(.n-card-header) {
  flex-direction: column;
}
```

NCard header 是顶层布局容器, 改它的 flex-direction 不影响内部 box width:
- header 内子项重新排列 (vertical), 但每个子项 outer width 不变
- NCard 整体 width 不变
- 父 flex 容器 (admin Layout) 不受影响

## 适用范围

任何用 :deep() override NaiveUI 组件内部 BEM 的场景:
- NButton (.n-button__content / .n-button__icon / etc.)
- NInput (.n-input__input-el / .n-input__placeholder / etc.)
- NCard (.n-card-header / .n-card__content / etc.)
- 任何 NaiveUI 组件内部 BEM (.n-XXX__YYY)

修改前自检:
1. 是否改的是 box 计算属性 (display/width/height/line-height/padding/margin)?
2. 父容器是否是 flex 且空间紧?
3. 改后会不会让 outer width 微变, 触发 wrap?

如果 yes → 不改, 在组件根类或外层容器改.
如果 no (仅改 visual) → 可改, 风险低.

## 跟其他 memory 的关系

- `feedback_no_unilateral_substitution.md`: 字面/字符不擅自替换
- `feedback_ps_native_arg_quoting.md`: PS 5.1 + ASCII " 编码不可靠
- 本文 (`feedback_naiveui_deep_override.md`): NaiveUI :deep() box 副作用
- `feedback_mobile_layout_total_width_audit.md` (待新建): Mobile flex 容器 wrap 诊断方法论

互补关系:
- 本文 = "改什么 + 怎么改"
- mobile_layout_audit = "诊断 wrap 应该量化总宽"
- 一起用: 改之前先量化, 改 box 属性前先想 outer width 影响

## NSpace v2 .n-space-item baseline alignment

### 背景

v0.2.10 Home.vue mobile header 反馈: SearchBox (NInput 矩形) + NetworkSwitcher
(NButton circle) + Settings (NButton circle) 横排, mobile 下垂直对齐错位.

Root cause: NSpace v2 默认子项 wrapper `.n-space-item` 用 `inline-block`, 元素
按 baseline 对齐. 圆 button 跟矩形 NInput baseline 不同, 视觉错位 (圆 button
看起来 "偏低"). v0.2.7 AuditLog 已碰到同模式, 同修法.

### 错误模式

```vue
<!-- ❌ 错: 期待 NSpace 默认就居中 -->
<NSpace>
  <HeaderSearchBox />
  <NetworkSwitcher />
  <NButton circle>...</NButton>
</NSpace>
```

视觉表现: 圆 button 跟 NInput baseline 不齐, 圆 button "偏低".

### 正确做法

```css
/* ✓ 对: 强制 .n-space-item 用 inline-flex + center */
.home-header :deep(.n-space-item) {
  display: inline-flex;
  align-items: center;
}
```

注意:
- selector 是 `.n-space-item` (无下划线, NaiveUI v2)
- 用 `:deep()` 跨 NSpace 内部 BEM
- 仅 mobile @media 内加 (避免 PC 回归)

### 适用范围

- 任何 NSpace 容器内混排 矩形元素 + 圆元素 + 不同 baseline 元素
- 任何 mobile / 紧凑布局发现元素 "略偏低 / 略偏高" 的视觉问题
- 任何 NSpace 子项视觉对齐失败 (尤其 NInput + NButton circle 混排)

### 关联 reference

- 本文上方 "## 跟其他 memory 的关系" section (mobile_layout_audit cross-ref)
- 本文上方 "## 安全做法" 第 3 点 (改 layout 容器, v0.2.10 Task 2.10 提到 `.n-space-item :deep() inline-flex`)
