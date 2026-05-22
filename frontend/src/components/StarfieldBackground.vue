<script setup lang="ts">
// v0.2.26: ambient starfield background.
//
// Replaces v0.2.25's aurora/glow CSS animation. Real-machine review
// found the glow scale-pulse felt mechanical and the wide-area breathing
// was distracting. The starfield holds positions fixed and only pulses
// opacity, with phase-offset jitter so stars don't blink in lockstep.
// Occasional meteors sweep upper-right → lower-left at a relaxed pace.
//
// First project use of requestAnimationFrame + <canvas>. Cleanup is
// strict (RAF cancel + visibilitychange + resize + reduced-motion MQL
// listeners) so router churn or auth bounces don't leak handlers.
//
// 30fps throttle via timestamp delta — eye can't resolve faster for
// ambient motion and the battery savings on phones are real. DPR is
// capped at 2 so 3x Retina iPhones don't pay 9x fill rate.
//
// Theme colors come from CSS vars (--mp-star, --mp-star-bright,
// --mp-meteor) read through getComputedStyle. The watch on
// ui.themePreset re-reads after the data-theme attribute flips.

import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useUIStore } from '../stores/ui'

const ui = useUIStore()
const canvasRef = ref<HTMLCanvasElement | null>(null)

interface Star {
  // Normalized coords [0..1] — window resize updates cssW/cssH and the
  // next frame multiplies through, no rebuild needed.
  x: number
  y: number
  r: number
  base: number
  amp: number
  // Random phase 0..2π so 15s-period breathing doesn't sync across stars
  // (the v0.2.25 lockstep look was one of the things Bevan disliked).
  phase: number
}

interface Meteor {
  x: number
  y: number
  vx: number
  vy: number
  len: number
  life: number
  maxLife: number
}

const STAR_COUNT = 26
// v0.2.27: 15s → 8s. The 15s cycle felt static on real-machine review
// (Bevan called the stars "frozen in place"). Combined with the
// base/amp tweak below this gives a livelier twinkle without the
// scale-pulse that drove the v0.2.25 rework.
const BREATHE_PERIOD_MS = 8000
const ANGULAR_W = (2 * Math.PI) / BREATHE_PERIOD_MS
const FRAME_INTERVAL_MS = 1000 / 30
const METEOR_GAP_MIN_MS = 20000
const METEOR_GAP_RANGE_MS = 20000

let ctx: CanvasRenderingContext2D | null = null
let cssW = 0
let cssH = 0
let stars: Star[] = []
let meteor: Meteor | null = null
let nextMeteorAt = 0
let rafId = 0
let lastFrameTs = 0
let startTs = 0

let colorStar = 'rgba(205, 214, 255, 0.7)'
let colorStarBright = 'rgba(220, 230, 255, 0.95)'
let colorMeteor = 'rgba(220, 230, 255, 0.9)'

let mql: MediaQueryList | null = null

function readThemeColors() {
  const cs = getComputedStyle(document.documentElement)
  const s = cs.getPropertyValue('--mp-star').trim()
  const sb = cs.getPropertyValue('--mp-star-bright').trim()
  const m = cs.getPropertyValue('--mp-meteor').trim()
  if (s) colorStar = s
  if (sb) colorStarBright = sb
  if (m) colorMeteor = m
}

function buildStars() {
  stars = []
  for (let i = 0; i < STAR_COUNT; i++) {
    stars.push({
      x: Math.random(),
      y: Math.random(),
      r: 0.5 + Math.random() * 1.1,
      // v0.2.27: darker base + wider amp so each star dips closer to 0
      // (off) and peaks near 1 (bright). The alpha clip at the draw site
      // takes care of out-of-range values, so [base-amp, base+amp] is
      // allowed to exceed [0,1] — the truncation is exactly what creates
      // the off-then-on twinkle that v0.2.26's softer range lacked.
      base: 0.15 + Math.random() * 0.25,
      amp: 0.35 + Math.random() * 0.45,
      phase: Math.random() * Math.PI * 2,
    })
  }
}

function resizeCanvas() {
  const canvas = canvasRef.value
  if (!canvas || !ctx) return
  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  const rect = canvas.getBoundingClientRect()
  cssW = rect.width
  cssH = rect.height
  canvas.width = Math.max(1, Math.floor(cssW * dpr))
  canvas.height = Math.max(1, Math.floor(cssH * dpr))
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
}

function spawnMeteor() {
  // Right upper third → vx negative, vy positive, slow. Trail length
  // 80-140 CSS px so the streak reads even on phones. maxLife in frames
  // (30fps), so 200-280 ≈ 6.7-9.3s on screen.
  meteor = {
    x: cssW * (0.6 + Math.random() * 0.4),
    y: cssH * (Math.random() * 0.3),
    vx: -(1.8 + Math.random()),
    vy: 0.8 + Math.random() * 0.5,
    len: 80 + Math.random() * 60,
    life: 0,
    maxLife: 200 + Math.random() * 80,
  }
}

function drawFrame(t: number) {
  if (!ctx) return
  ctx.clearRect(0, 0, cssW, cssH)

  for (const s of stars) {
    const raw = s.base + s.amp * Math.sin(t * ANGULAR_W + s.phase)
    const a = raw < 0 ? 0 : raw > 1 ? 1 : raw
    ctx.globalAlpha = a
    ctx.fillStyle = a > 0.75 ? colorStarBright : colorStar
    ctx.beginPath()
    ctx.arc(s.x * cssW, s.y * cssH, s.r, 0, Math.PI * 2)
    ctx.fill()
  }
  ctx.globalAlpha = 1

  if (meteor) {
    meteor.x += meteor.vx
    meteor.y += meteor.vy
    meteor.life += 1
    const f = meteor.life / meteor.maxLife
    let fade = 1
    if (f < 0.15) fade = f / 0.15
    else if (f > 0.6) fade = (1 - f) / 0.4
    if (fade < 0) fade = 0

    const tx = meteor.x - meteor.vx * meteor.len * 0.5
    const ty = meteor.y - meteor.vy * meteor.len * 0.5
    const grad = ctx.createLinearGradient(meteor.x, meteor.y, tx, ty)
    grad.addColorStop(0, colorMeteor)
    grad.addColorStop(1, 'transparent')
    ctx.strokeStyle = grad
    ctx.globalAlpha = fade
    ctx.lineWidth = 1.4
    ctx.lineCap = 'round'
    ctx.beginPath()
    ctx.moveTo(meteor.x, meteor.y)
    ctx.lineTo(tx, ty)
    ctx.stroke()
    ctx.globalAlpha = 1

    if (
      meteor.life >= meteor.maxLife ||
      meteor.x < -meteor.len ||
      meteor.y > cssH + meteor.len
    ) {
      meteor = null
      nextMeteorAt = t + METEOR_GAP_MIN_MS + Math.random() * METEOR_GAP_RANGE_MS
    }
  } else if (t >= nextMeteorAt) {
    spawnMeteor()
  }
}

function drawStaticFrame() {
  if (!ctx) return
  ctx.clearRect(0, 0, cssW, cssH)
  for (const s of stars) {
    ctx.globalAlpha = s.base
    ctx.fillStyle = colorStar
    ctx.beginPath()
    ctx.arc(s.x * cssW, s.y * cssH, s.r, 0, Math.PI * 2)
    ctx.fill()
  }
  ctx.globalAlpha = 1
}

function loop(now: number) {
  rafId = requestAnimationFrame(loop)
  if (now - lastFrameTs < FRAME_INTERVAL_MS) return
  lastFrameTs = now
  drawFrame(now - startTs)
}

function start() {
  if (rafId) return
  startTs = performance.now()
  lastFrameTs = 0
  // First meteor 5-20s in — full 20-40s gap would leave a brand-new
  // visitor staring at static stars for too long.
  nextMeteorAt = 5000 + Math.random() * 15000
  rafId = requestAnimationFrame(loop)
}

function stop() {
  if (rafId) cancelAnimationFrame(rafId)
  rafId = 0
}

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function onVisibility() {
  if (document.hidden) stop()
  else if (!prefersReducedMotion()) start()
}

function onResize() {
  resizeCanvas()
  if (rafId === 0) drawStaticFrame()
}

function onReducedMotionChange() {
  if (prefersReducedMotion()) {
    stop()
    drawStaticFrame()
  } else {
    start()
  }
}

watch(
  () => ui.themePreset,
  () => {
    // App.vue writes data-theme on the documentElement inside its own
    // watch — defer one microtask so getComputedStyle picks up the new
    // attribute. If RAF is running it'll repaint with the new colors on
    // the next tick; otherwise repaint the static frame now.
    queueMicrotask(() => {
      readThemeColors()
      if (rafId === 0) drawStaticFrame()
    })
  },
)

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  ctx = canvas.getContext('2d')
  if (!ctx) return

  readThemeColors()
  resizeCanvas()
  buildStars()

  if (prefersReducedMotion()) drawStaticFrame()
  else start()

  window.addEventListener('resize', onResize)
  document.addEventListener('visibilitychange', onVisibility)
  mql = window.matchMedia('(prefers-reduced-motion: reduce)')
  mql.addEventListener('change', onReducedMotionChange)
})

onBeforeUnmount(() => {
  stop()
  window.removeEventListener('resize', onResize)
  document.removeEventListener('visibilitychange', onVisibility)
  if (mql) {
    mql.removeEventListener('change', onReducedMotionChange)
    mql = null
  }
  ctx = null
})
</script>

<template>
  <canvas ref="canvasRef" class="mp-starfield" aria-hidden="true"></canvas>
</template>

<style scoped>
.mp-starfield {
  position: fixed;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
  z-index: 0;
  /* Same isolation trick as v0.2.25 mp-dynamic-bg: keeps Login card +
     modal acrylic from re-sampling the canvas every animation frame. */
  isolation: isolate;
  pointer-events: none;
}
</style>
