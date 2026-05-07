---
name: Mobile flex 容器多元素 wrap 诊断要量化总宽
description: Mobile 顶栏多元素布局问题 (vertical 偏低 / 元素位置乱) 应先量化总宽 vs viewport, 不能只看截图局部. 多元素 flex 容器 wrap 通常是总宽超 viewport 导致, 不是局部 alignment 问题
type: feedback
---

# Mobile Flex 容器 Wrap 诊断方法论

v0.2.10 Task 2.13 实战误诊: 看截图说 "Settings 偏低 5px" → 实际是 wrap 第 2 行.
误诊导致修了无关问题 (.n-button__content 内部 alignment), 反而引入新 bug 
(NSpace 子项总宽变化, 强化 wrap).

后由 Claude Code Task 2.16 量化总宽 ~370px > 375px viewport, 真修法是隐藏 brand 文字节省 ~100px.

## 教训

★ Mobile 顶栏多元素 wrap 诊断, 先量化总宽, 别只看截图局部 ★

视觉问题分两类:

1. **Wrap 类**: 元素掉到下一行 (横向空间不够)
   - 看 DevTools 元素 inspector 的 box outline
   - 看 NSpace / flex 容器的 wrap 行为
   - 量化: 元素总宽 vs viewport 宽
   - 修法: 减少元素 / 缩小元素 / 改 flex 布局

2. **Alignment 类**: 元素位置偏移 (单维度对齐不齐)
   - 看 vertical-align / baseline / line-height
   - 量化: 元素 baseline 间像素差
   - 修法: 改 align-items / vertical-align / 容器 baseline

★ 关键 ★: 看截图觉得"偏低 5px"时, 先确认是 Alignment 类 (在第 1 行偏离), 
还是 Wrap 类 (掉到第 2 行 visual 看起来"低"). 这两种修法不同.

## 量化方法

Mobile flex 顶栏 wrap 诊断步骤:

### 1. 列每个子项 width (含 padding + border + margin)

| 元素 | 自身 width | outer (含 box-sizing) |
|------|-----------|----------------------|
| brand title text | 100px | 100px |
| brand logo (emoji) | 22px | 22px |
| SearchBox NInput | 160px | 160px (mobile @media 已设) |
| NetworkSwitcher button | 28px | 28px (circle) |
| admin Settings button | 28px | 28px (circle) |

### 2. 加 NSpace gap × (子项数 - 1)

NSpace gap=12 × 4 子项 = 12 × 3 = 36px

### 3. 加容器 padding (左右)

`.home-header` padding 0 1rem (16px) × 2 = 32px

### 4. 求总宽 vs mobile viewport

```
总宽 = brand(100+22) + SearchBox(160) + Network(28) + Settings(28) + gap(36) + padding(32)
     = 122 + 160 + 28 + 28 + 36 + 32
     = 406px

Mobile viewport = 375px (iPhone SE / 大部分主流 mobile)

406 > 375 → wrap (NSpace 默认 wrap=true 把最右元素挤下行)
```

### 5. 决定修法

修法选项 (按可行度):

A. **减少元素**: 隐藏次要元素 (brand title text -100 → 总宽 306 < 375 ✓)
B. **缩小元素**: 缩 SearchBox / 缩字号 (有上限, 太小不可用)
C. **改布局**: 拆 2 行 / 折叠 button (工作量大)
D. **改 viewport**: 加 zoom < 1 (UX 差, 不推荐)

★ 优先 A, 然后 B, 最后 C ★

## 案例: v0.2.10 Task 2.13 + 2.14 + 2.16

### Task 2.13 误诊 (反面)

看截图认为 "Settings 偏低 5px" (Alignment 类), 修法:

```css
.home-header :deep(.n-button.n-button--circle .n-button__content) {
  display: inline-flex;
  align-items: center;
  line-height: 1;
}
```

实际副作用: 改 button 内部 box, outer width 变化, 强化 wrap. 视觉更糟.

### Task 2.14 撤销

撤销 Task 2.13 加的规则.

### Task 2.16 真修 (正面)

Claude Code 主动量化总宽:

```
之前: ~402px > 375 → wrap
patch (隐藏 brand 文字): ~302px < 375 → fit ✓
```

修法 (mobile @media):

```css
.home-header__title {
  display: none;   /* 隐藏 brand 文字, 节省 ~100px */
}
```

PC 不动 (零回归).

## 适用范围

任何 mobile flex 容器内多元素 layout 问题:
- 顶栏 (HomeHeader / admin Layout / etc.)
- 卡片网格 (CardItem / CityWidget / etc.)
- 工具栏 (Toolbar / 等)

诊断步骤:
1. 截图看是 Wrap 类还是 Alignment 类 (常被误诊)
2. 如 Wrap → 量化总宽 vs viewport
3. 如总宽 > viewport → 减少 / 缩小元素
4. 如总宽 < viewport 但仍乱 → Alignment 类, 改 align/baseline

## 跟其他 memory 的关系

- `feedback_naiveui_deep_override.md`: 改 NaiveUI :deep() box 属性会影响 outer width
- 本文 (`feedback_mobile_layout_total_width_audit.md`): wrap 诊断量化总宽
- 一起用: 量化总宽时考虑 :deep() override 对 outer width 的潜在影响
- 改 :deep() 后, 重新量化总宽确认 layout 不被引发 wrap
