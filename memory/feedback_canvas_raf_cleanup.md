---
name: Canvas + requestAnimationFrame cleanup pattern (工程首引)
description: v0.2.26 StarfieldBackground.vue 是工程首个 canvas 渲染组件 + 首引 RAF; 建立 cleanup pattern (RAF cancel + visibility pause + DPR + reduced-motion + 主题色) 供未来 canvas 动效组件复用
type: feedback
---

## 背景

v0.2.26 [StarfieldBackground.vue](../frontend/src/components/StarfieldBackground.vue)
是工程首个 `<canvas>` 渲染组件 + 首引 `requestAnimationFrame` (此前 canvas 仅
[imageCompress.ts](../frontend/src/utils/imageCompress.ts) +
[wallpaper.ts](../frontend/src/api/wallpaper.ts) 离屏图片处理). 建立此 pattern
供未来 canvas 动效组件复用.

## Pattern (StarfieldBackground.vue 实证)

### 1. RAF id ref + cancel (防泄漏)
```ts
let rafId = 0
function start() { rafId = requestAnimationFrame(loop) }
function stop() { if (rafId) cancelAnimationFrame(rafId); rafId = 0 }
onBeforeUnmount(() => stop())
```

### 2. document.hidden 暂停 (省 CPU / 电池)
```ts
function onVisibility() {
  if (document.hidden) stop()
  else if (!prefersReducedMotion()) start()
}
document.addEventListener('visibilitychange', onVisibility)
// onBeforeUnmount: removeEventListener
```
参考 [network.ts:236/245/253](../frontend/src/stores/network.ts) 既有 pattern.

### 3. 限帧 (时间戳 delta, 不用 setTimeout)
```ts
const FRAME_INTERVAL_MS = 1000 / 30  // 30fps
function loop(now: number) {
  rafId = requestAnimationFrame(loop)
  if (now - lastFrameTs < FRAME_INTERVAL_MS) return  // skip
  lastFrameTs = now
  drawFrame(now - startTs)
}
```

### 4. DPR (Retina 不糊)
```ts
const dpr = Math.min(window.devicePixelRatio || 1, 2)  // cap 2 防 9x 填充
canvas.width = Math.max(1, Math.floor(cssW * dpr))
canvas.height = Math.max(1, Math.floor(cssH * dpr))
ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
// resize listener 重算
```

### 5. reduced-motion (a11y)
```ts
function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}
// 启动判断: 真 → drawStaticFrame(), 假 → start()
// 加 MQL change listener 支持 OS 运行时切换:
mql.addEventListener('change', onReducedMotionChange)
```

### 6. 主题色 (canvas 不能用 CSS var)
```ts
function readThemeColors() {
  const cs = getComputedStyle(document.documentElement)
  colorStar = cs.getPropertyValue('--mp-star').trim() || fallback
  // ...
}
watch(
  () => ui.themePreset,
  () => {
    // App.vue 在 watch 里写 documentElement.dataset.theme — defer 等更新
    queueMicrotask(() => {
      readThemeColors()
      if (rafId === 0) drawStaticFrame()  // reduced-motion 时重画
    })
  },
)
```

### 7. CSS scoped style (acrylic 不采样 + a11y / e2e 不影响)
```css
.mp-canvas-bg {
  position: fixed; inset: 0; z-index: 0;
  isolation: isolate;   /* 关键: Login card / modal acrylic 不采样 canvas */
  pointer-events: none;
}
```
模板: `<canvas aria-hidden="true">` — 不进 a11y tree, e2e selector 全保留.

## 关键陷阱

- ★ canvas 读 CSS var 必须等 `dataset.theme` 更新 (`queueMicrotask`),
  否则读到旧主题色 ★
- ★ `onBeforeUnmount` 必须 cancel RAF + remove **所有** listener
  (`visibilitychange` + `resize` + `mql change`), 否则路由切换 / 私有模式 auth
  bounce 都会泄漏 handler ★
- `aria-hidden="true"` 让 canvas 不进 a11y tree → e2e selector 不受影响
  (v0.2.26 ci.yml e2e 0 regression 验证)
- DPR 不 cap (`Math.min(dpr, 2)`) 时, 3x Retina iPhone 付 9x 填充, 移动性能差
- 限帧用时间戳 delta, 不用 `setTimeout(() => RAF, 1000/30)` — 后者跟 RAF 调度
  不同步, 易导致掉帧 / 卡顿

## How to apply

未来加 canvas 动效组件 (粒子 / 波形 / WebGL 等) 时, 完整复用这 7 项 pattern.
跟 [[feedback_silent_render_failure]] 配合 — canvas 渲染失败也是 silent (画布
空白无 console error), 真机视觉门必看.
