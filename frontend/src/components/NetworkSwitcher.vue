<script setup lang="ts">
import { computed, h, type VNode } from 'vue'
import { NButton, NDropdown, NSelect, type DropdownOption, type SelectOption } from 'naive-ui'
import { Loader2 } from 'lucide-vue-next'
import { useNetworkStore, type NetworkMode } from '@/stores/network'

const network = useNetworkStore()

const options: SelectOption[] = [
  { label: '自动检测', value: 'auto' },
  { label: '强制内网', value: 'internal' },
  { label: '强制外网', value: 'external' },
]

const value = computed({
  get: () => network.global,
  set: (v) => network.setGlobal(v as NetworkMode),
})

// Mobile (< 768px) gets an icon-button trigger to save horizontal space.
// Same option list rendered via NDropdown so the menu UX stays consistent.
const dropdownOptions = computed<DropdownOption[]>(() =>
  options.map((o) => ({
    label: `${o.label as string}${network.global === o.value ? ' ✓' : ''}`,
    key: String(o.value),
  })),
)

function handleDropdownSelect(key: string | number) {
  network.setGlobal(String(key) as NetworkMode)
}

const currentLabel = computed(
  () => options.find((o) => o.value === network.global)?.label ?? '',
)

// v0.2.23: status dot tracks the auto-detection result. Only visible when
// global === 'auto' (other modes are user-forced — no detection runs).
//
// dotKind drives the colour modifier class; tooltip drives the title attr.
//   'lan'        → 绿点  (探测成功 + 当前内网可达)
//   'wan'        → 橙点  (探测失败/超时 + 走外网)
//   'detecting'  → 灰点  (首次探测中 / 'unknown' fallback)
// We collapse 'detecting' and 'unknown' into the same visual: both are
// "indeterminate" states from the user's POV — spinner does the heavy
// lifting for the actively-probing case.
type DotKind = 'lan' | 'wan' | 'detecting'

const dotKind = computed<DotKind>(() => {
  const d = network.detectedMode
  if (d === 'lan') return 'lan'
  if (d === 'wan') return 'wan'
  return 'detecting'
})

const indicatorTitle = computed(() => {
  if (network.probeStatus === 'probing') return '正在探测网络环境…'
  if (dotKind.value === 'lan') return '自动检测：当前在内网（探测成功）'
  if (dotKind.value === 'wan') return '自动检测：当前在外网（探测失败/超时）'
  return '尚未完成探测，将在 1.5s 内确定'
})

const switcherTooltip = '自动检测当前网络环境。临时手动切换将在刷新后恢复自动。'

function globeIcon(): VNode {
  return h(
    'svg',
    {
      width: 18,
      height: 18,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
      'aria-hidden': 'true',
    },
    [
      h('circle', { cx: 12, cy: 12, r: 10 }),
      h('path', { d: 'M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20' }),
      h('path', { d: 'M2 12h20' }),
    ],
  )
}
</script>

<template>
  <div class="network-switcher-wrap network-switcher-wrap--wide" :title="switcherTooltip">
    <NSelect
      v-model:value="value"
      :options="options"
      size="small"
      class="network-switcher network-switcher--wide"
      data-testid="network-switcher-wide"
    />
    <span
      v-if="network.global === 'auto'"
      class="network-switcher__indicator"
      :title="indicatorTitle"
      data-testid="network-switcher-indicator"
    >
      <span
        class="network-switcher__dot"
        :class="`network-switcher__dot--${dotKind}`"
        :data-detected="dotKind"
      />
      <Loader2
        v-if="network.probeStatus === 'probing'"
        :size="12"
        class="network-switcher__spinner"
        aria-hidden="true"
      />
    </span>
  </div>
  <div class="network-switcher__mobile-trigger-wrap">
    <NDropdown
      :options="dropdownOptions"
      placement="bottom-end"
      trigger="click"
      @select="handleDropdownSelect"
    >
      <NButton
        size="small"
        circle
        class="network-switcher network-switcher--narrow"
        :title="`网络模式：${currentLabel}`"
        data-testid="network-switcher-narrow"
      >
        <component :is="globeIcon()" />
      </NButton>
    </NDropdown>
    <!-- v0.2.23 B-patch.2: Mobile detection dot, overlaid at button corner.
         Mirrors PC dot semantics (lan/wan/detecting) but uses a notification-
         badge layout: 8px circle with a 2px ring (card-bg-tinted) to punch
         through the page background, regardless of wallpaper hue. -->
    <span
      v-if="network.global === 'auto'"
      class="network-switcher__mobile-dot"
      :class="{
        'network-switcher__mobile-dot--lan':       network.detectedMode === 'lan',
        'network-switcher__mobile-dot--wan':       network.detectedMode === 'wan',
        'network-switcher__mobile-dot--detecting': network.probeStatus === 'probing',
      }"
      :title="indicatorTitle"
      data-testid="network-switcher-mobile-dot"
    />
  </div>
</template>

<style scoped>
.network-switcher-wrap--wide {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.network-switcher--wide {
  width: 200px;
}
.network-switcher--narrow {
  display: none !important;
}

/* v0.2.23: auto-detection status indicator. Only visible in `global=auto`
   mode; sits to the right of the NSelect. Dot is a 8px circle with a state
   colour; spinner is 12px Loader2 that appears only while probeStatus is
   'probing'. Both share the same wrapper so they pack tightly with 4px gap. */
.network-switcher__indicator {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  /* 12px square box so the dot is vertically centered against the small
     NSelect (height ~28px). The dot itself is 8px; padding leaves room
     for the spinner without shifting layout when it appears. */
  height: 12px;
}
.network-switcher__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  display: inline-block;
}
.network-switcher__dot--lan {
  /* 绿: 内网可达 (探测成功). Same hue family as NaiveUI success token but
     uses raw hex so the colour stays consistent across moon/risen themes
     (status semantics shouldn't shift with the brand palette). */
  background: #22c55e;
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.18);
}
.network-switcher__dot--wan {
  /* 橙: 外网模式 (探测失败/超时 → 保守降级). */
  background: #f97316;
  box-shadow: 0 0 0 2px rgba(249, 115, 22, 0.18);
}
.network-switcher__dot--detecting {
  /* 灰 pulse: 探测中 / 未完成. Pulse animation keeps the indicator alive
     even when the spinner isn't showing (probe completes fast — animation
     gives the user something to lock onto during the 1.5s window). */
  background: rgba(156, 163, 175, 0.8);
  animation: network-dot-pulse 1.4s ease-in-out infinite;
}
.network-switcher__spinner {
  color: var(--mp-text-tertiary);
  animation: network-spinner-spin 0.8s linear infinite;
  flex-shrink: 0;
}

@keyframes network-dot-pulse {
  0%, 100% { opacity: 0.4; transform: scale(0.85); }
  50%      { opacity: 1;   transform: scale(1); }
}
@keyframes network-spinner-spin {
  to { transform: rotate(360deg); }
}

/* v0.2.23 B-patch.2: Mobile dot wrapper. position:relative anchors the
   absolutely-positioned dot to the 44x44 button. inline-block keeps it
   sized to its content (no extra width inherited from header flex). */
.network-switcher__mobile-trigger-wrap {
  position: relative;
  display: none;
}
.network-switcher__mobile-dot {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  /* Substitute for spec's nonexistent --mp-bg-elevated. card-bg approximates
     the "ring punching through page bg" effect across both moon/risen themes
     (semi-translucent dark/brown matches each theme's panel surface). */
  border: 2px solid var(--mp-card-bg);
  pointer-events: none;
  /* Default state (no modifier): keep the spot invisible during the brief
     'idle' window after mount, before probe() flips status to 'probing'. */
  background: transparent;
}
.network-switcher__mobile-dot--lan {
  background: #4ade80;
}
.network-switcher__mobile-dot--wan {
  background: #f97316;
}
.network-switcher__mobile-dot--detecting {
  background: var(--mp-text-tertiary);
  animation: network-dot-pulse 1.4s ease-in-out infinite;
}

@media (max-width: 768px) {
  .network-switcher-wrap--wide {
    display: none !important;
  }
  .network-switcher__mobile-trigger-wrap {
    display: inline-block;
  }
  /* v0.2.11: Mobile 44x44 box mirror admin/Layout.vue + Home Settings.
     Visual unity across all mobile top bar icon buttons. */
  .network-switcher--narrow {
    display: inline-flex !important;
    width: 44px;
    height: 44px;
    border: 1px solid var(--mp-card-border);
    border-radius: 8px;
    background: transparent;
    transition: background 0.15s;
  }
  .network-switcher--narrow:hover {
    background: var(--mp-card-bg-hover);
  }
}
</style>
