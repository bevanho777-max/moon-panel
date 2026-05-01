<script setup lang="ts">
import { computed, h, nextTick, onMounted, ref, type VNode } from 'vue'
import { NDropdown, type DropdownOption } from 'naive-ui'
import type { Card } from '@/api/card'
import { useNetworkStore } from '@/stores/network'
import { effectiveURL } from '@/utils/cardUrl'
import { useLongPress } from '@/composables/useLongPress'
import LucideIcon from '@/components/LucideIcon.vue'

const props = defineProps<{ card: Card }>()
const network = useNetworkStore()

// Plex-badge diagnostic (Phase 2.5a, per user request — F12 console reveals
// whether url_internal/external is whitespace-only, which would silently
// bypass the auto-badge logic). Logs once per card mount, only when actually
// suspicious (non-empty raw value but trim-empty).
onMounted(() => {
  for (const field of ['url_internal', 'url_external'] as const) {
    const v = props.card[field]
    if (v && v.trim() === '') {
      console.warn(
        `[Moon Panel] Card "${props.card.title}" (id=${props.card.id}) ${field} is whitespace-only — auto-badge logic treats this as set, you may want to clear it. Raw: ${JSON.stringify(v)}`,
      )
    }
  }
})

const url = computed(() =>
  effectiveURL(props.card, {
    global: network.global,
    overrides: network.overrides,
  }),
)

const showMenu = ref(false)
const menuX = ref(0)
const menuY = ref(0)

function openMenu(x: number, y: number) {
  showMenu.value = false
  menuX.value = x
  menuY.value = y
  nextTick(() => {
    showMenu.value = true
  })
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  openMenu(e.clientX, e.clientY)
}

const longPress = useLongPress((e) => {
  e.preventNativeMenu()
  openMenu(e.clientX, e.clientY)
})

const menuOptions = computed<DropdownOption[]>(() => {
  const ov = network.overrides[props.card.id]
  const checkmark = (active: boolean) => (active ? ' ✓' : '')
  return [
    {
      label: `自动（跟随全局）${checkmark(!ov)}`,
      key: 'auto',
    },
    {
      label: `固定走内网${checkmark(ov === 'internal')}`,
      key: 'internal',
      disabled: !props.card.url_internal.trim(),
    },
    {
      label: `固定走外网${checkmark(ov === 'external')}`,
      key: 'external',
      disabled: !props.card.url_external.trim(),
    },
    { type: 'divider', key: 'd1' },
    {
      label: url.value
        ? `当前走 ${url.value.side === 'internal' ? '内网' : '外网'}${url.value.fallback ? '（fallback）' : ''}`
        : '当前无可用 URL',
      key: 'info',
      disabled: true,
    },
  ]
})

function handleMenuSelect(key: string | number) {
  if (key === 'auto') network.clearOverride(props.card.id)
  else if (key === 'internal' || key === 'external') {
    network.setOverride(props.card.id, key)
  }
  showMenu.value = false
}

// Computed: which network sides are populated. Drives the auto "仅内网/仅外网"
// badge — both populated → no badge (default state, switcher chooses).
const internalSet = computed(() => props.card.url_internal.trim() !== '')
const externalSet = computed(() => props.card.url_external.trim() !== '')
const networkBadge = computed<'internal-only' | 'external-only' | null>(() => {
  if (internalSet.value && !externalSet.value) return 'internal-only'
  if (externalSet.value && !internalSet.value) return 'external-only'
  return null
})

function homeBadgeIcon(): VNode {
  return h(
    'svg',
    { width: 12, height: 12, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2.2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' },
    [
      h('path', { d: 'm3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z' }),
      h('polyline', { points: '9 22 9 12 15 12 15 22' }),
    ],
  )
}

function globeBadgeIcon(): VNode {
  return h(
    'svg',
    { width: 12, height: 12, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2.2, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' },
    [
      h('circle', { cx: 12, cy: 12, r: 10 }),
      h('path', { d: 'M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20' }),
      h('path', { d: 'M2 12h20' }),
    ],
  )
}

// Icon thumbnail rendering. URL icons can fail (404, CORS, offline) — track
// per-card load failure and swap to a lucide:image-off placeholder so the
// layout doesn't get a blank gap. See memory/feedback_external_resource_fallback.md.
const iconFailed = ref(false)
const isUrlIcon = computed(() => /^https?:\/\//i.test(props.card.icon))
const isLucideIcon = computed(() => props.card.icon.startsWith('lucide:'))
const isUploadIcon = computed(() => props.card.icon.startsWith('upload:'))
// upload:public/icons/abc.webp → /uploads/public/icons/abc.webp
const uploadUrl = computed(() =>
  isUploadIcon.value ? '/uploads/' + props.card.icon.slice('upload:'.length) : '',
)
// lucide:wrench → "wrench" (just the name, LucideIcon expects unprefixed)
const lucideName = computed(() =>
  isLucideIcon.value ? props.card.icon.slice('lucide:'.length) : '',
)

const cardClasses = computed(() => ({
  'card-item': true,
  'card-item--disabled': !url.value,
  // Phase 2.5c-acrylic: when body.has-wallpaper, the global rule overrides
  // .card-item's own bg/backdrop-filter to medium-tier acrylic. When no
  // wallpaper, the class is a no-op and the existing .card-item style
  // (rgba(255,255,255,0.06) + 8px blur from Phase 2.5b) keeps showing.
  'mp-acrylic-medium': true,
}))

const target = computed(() => (props.card.open_in_new_tab ? '_blank' : '_self'))
const rel = computed(() => (props.card.open_in_new_tab ? 'noopener noreferrer' : undefined))

const tooltipText = computed(() => {
  if (!url.value) return '未设置链接'
  const parts = [props.card.description]
  if (url.value.fallback) parts.push(`(fallback: ${url.value.side === 'internal' ? '内网空，走外网' : '外网空，走内网'})`)
  parts.push(url.value.url)
  return parts.filter(Boolean).join('\n')
})
</script>

<template>
  <a
    v-if="url"
    :class="cardClasses"
    :href="url.url"
    :target="target"
    :rel="rel"
    :title="tooltipText"
    @contextmenu="handleContextMenu"
    @touchstart="longPress.onTouchStart"
    @touchmove="longPress.onTouchMove"
    @touchend="longPress.onTouchEnd"
    @touchcancel="longPress.onTouchCancel"
  >
    <span class="card-item__icon">
      <img
        v-if="isUrlIcon && !iconFailed"
        :src="card.icon"
        class="card-item__icon-img"
        alt=""
        @error="iconFailed = true"
      />
      <img
        v-else-if="isUploadIcon && !iconFailed"
        :src="uploadUrl"
        class="card-item__icon-img"
        alt=""
        @error="iconFailed = true"
      />
      <span
        v-else-if="(isUrlIcon || isUploadIcon) && iconFailed"
        class="card-item__icon-box card-item__icon-box--failed"
        :title="`图片加载失败: ${card.icon}`"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <line x1="2" y1="2" x2="22" y2="22" />
          <path d="M10.41 10.41a2 2 0 1 1-2.83-2.83" />
          <line x1="13.5" y1="13.5" x2="6" y2="21" />
          <line x1="18" y1="12" x2="21" y2="15" />
          <path d="M3.59 3.59A1.99 1.99 0 0 0 3 5v14a2 2 0 0 0 2 2h14c.55 0 1.052-.22 1.41-.59" />
          <path d="M21 15V5a2 2 0 0 0-2-2H9" />
        </svg>
      </span>
      <span
        v-else-if="isLucideIcon"
        class="card-item__icon-box card-item__icon-box--lucide"
        :title="card.icon"
      >
        <LucideIcon :name="lucideName" :size="24" />
      </span>
      <span
        v-else-if="card.icon"
        class="card-item__icon-box card-item__icon-box--unknown"
        :title="card.icon"
      >?</span>
      <span
        v-else
        class="card-item__icon-box card-item__icon-box--empty"
      >—</span>
    </span>
    <div class="card-item__body">
      <div class="card-item__title-row">
        <span class="card-item__title">{{ card.title }}</span>
        <span
          v-if="networkBadge === 'internal-only'"
          class="card-item__badge card-item__badge--internal"
          title="此卡片只设置了内网地址"
        >
          <component :is="homeBadgeIcon()" />
          <span>仅内网</span>
        </span>
        <span
          v-else-if="networkBadge === 'external-only'"
          class="card-item__badge card-item__badge--external"
          title="此卡片只设置了外网地址"
        >
          <component :is="globeBadgeIcon()" />
          <span>仅外网</span>
        </span>
      </div>
      <div v-if="card.description" class="card-item__desc">{{ card.description }}</div>
    </div>
    <span v-if="url.fallback" class="card-item__fallback" title="主选 URL 为空，已 fallback 到另一边">↩</span>
  </a>
  <div
    v-else
    :class="cardClasses"
    :title="tooltipText"
    @contextmenu="handleContextMenu"
  >
    <span class="card-item__icon">
      <img
        v-if="isUrlIcon && !iconFailed"
        :src="card.icon"
        class="card-item__icon-img"
        alt=""
        @error="iconFailed = true"
      />
      <img
        v-else-if="isUploadIcon && !iconFailed"
        :src="uploadUrl"
        class="card-item__icon-img"
        alt=""
        @error="iconFailed = true"
      />
      <span
        v-else-if="(isUrlIcon || isUploadIcon) && iconFailed"
        class="card-item__icon-box card-item__icon-box--failed"
        :title="`图片加载失败: ${card.icon}`"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <line x1="2" y1="2" x2="22" y2="22" />
          <path d="M10.41 10.41a2 2 0 1 1-2.83-2.83" />
          <line x1="13.5" y1="13.5" x2="6" y2="21" />
          <line x1="18" y1="12" x2="21" y2="15" />
          <path d="M3.59 3.59A1.99 1.99 0 0 0 3 5v14a2 2 0 0 0 2 2h14c.55 0 1.052-.22 1.41-.59" />
          <path d="M21 15V5a2 2 0 0 0-2-2H9" />
        </svg>
      </span>
      <span
        v-else-if="isLucideIcon"
        class="card-item__icon-box card-item__icon-box--lucide"
        :title="card.icon"
      >
        <LucideIcon :name="lucideName" :size="24" />
      </span>
      <span
        v-else-if="card.icon"
        class="card-item__icon-box card-item__icon-box--unknown"
        :title="card.icon"
      >?</span>
      <span
        v-else
        class="card-item__icon-box card-item__icon-box--empty"
      >—</span>
    </span>
    <div class="card-item__body">
      <div class="card-item__title-row">
        <span class="card-item__title">{{ card.title }}</span>
      </div>
      <div class="card-item__desc">未设置链接</div>
    </div>
  </div>

  <NDropdown
    placement="bottom-start"
    trigger="manual"
    :show="showMenu"
    :x="menuX"
    :y="menuY"
    :options="menuOptions"
    @select="handleMenuSelect"
    @clickoutside="showMenu = false"
  />
</template>

<style scoped>
.card-item {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.06);
  /* Frosted glass: blur is invisible on pure dark bg but kicks in once a
     background image is set in Phase 2.5b. Saturate boosts color a bit. */
  -webkit-backdrop-filter: blur(8px) saturate(120%);
  backdrop-filter: blur(8px) saturate(120%);
  border: 1px solid rgba(255, 255, 255, 0.05);
  /* Hover uses translateY + box-shadow instead of scale, so the
     backdrop-filter region keeps a fixed size and the browser doesn't
     re-sample the wallpaper under each card on every frame. The previous
     scale(1.03) made backdrop blur ~3x more expensive per frame. */
  transition:
    box-shadow 180ms ease,
    transform 180ms ease,
    background-color 180ms ease;
  will-change: transform, box-shadow;
  cursor: pointer;
  min-width: 0;
  /* iOS: suppress system long-press menu */
  -webkit-touch-callout: none;
  -webkit-user-select: none;
  user-select: none;
  text-decoration: none;
  color: inherit;
}
.card-item:hover {
  /* Subtle white tint over the existing acrylic — half a notch above the
     idle 0.06 so the change registers without "everything turns white". */
  background-color: rgba(255, 255, 255, 0.08);
  /* Outer 1px ring + soft drop glow replace the old scale-up. Uses the
     existing brand blue (#5b8def family); dynamic theme primary isn't
     exposed as a CSS var today, so plumbing that could be a follow-up. */
  box-shadow:
    0 0 0 1px rgba(135, 165, 240, 0.3),
    0 6px 20px rgba(91, 141, 239, 0.15);
  transform: translateY(-1px);
}
.card-item--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.card-item--disabled:hover {
  background-color: rgba(255, 255, 255, 0.06);
  transform: none;
  box-shadow: none;
}
.card-item__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  padding: 8px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.04);
  box-sizing: border-box;
}
.card-item__icon-img,
.card-item__icon-box {
  width: 100%;
  height: 100%;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
}
.card-item__icon-img {
  object-fit: contain;
}
.card-item__icon-box--empty {
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.3);
}
.card-item__icon-box--lucide {
  background: rgba(91, 141, 239, 0.15);
  color: #5b8def;
}
.card-item__icon-box--upload {
  background: rgba(99, 226, 183, 0.15);
  color: #63e2b7;
}
.card-item__icon-box--unknown {
  background: rgba(255, 193, 77, 0.15);
  color: #ffc14d;
}
.card-item__icon-box--failed {
  background: rgba(255, 100, 100, 0.12);
  color: rgba(255, 130, 130, 0.7);
}
.card-item__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.card-item__title-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.card-item__badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 500;
  letter-spacing: 0.02em;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.55);
  white-space: nowrap;
}
.card-item__badge--internal {
  /* same neutral grey as external — both are "limitation" hints, not status */
}
.card-item__title {
  /* min-width: 0 lets the flex item shrink below its content width so the
     trailing badge stays visible — without it, long titles push badges out. */
  min-width: 0;
  flex: 1 1 auto;
  font-size: 14px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.92);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.card-item__desc {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.card-item__fallback {
  font-size: 0.7rem;
  opacity: 0.5;
  flex-shrink: 0;
}

@media (max-width: 480px) {
  .card-item {
    padding: 10px;
    gap: 10px;
    border-radius: 10px;
  }
  .card-item__icon {
    width: 40px;
    height: 40px;
    padding: 6px;
    border-radius: 8px;
  }
  .card-item__title {
    font-size: 13px;
  }
  .card-item__desc {
    font-size: 11px;
  }
}
</style>
