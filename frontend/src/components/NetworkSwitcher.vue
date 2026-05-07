<script setup lang="ts">
import { computed, h, type VNode } from 'vue'
import { NButton, NDropdown, NSelect, type DropdownOption, type SelectOption } from 'naive-ui'
import { useNetworkStore, type NetworkMode } from '@/stores/network'

const network = useNetworkStore()

const options: SelectOption[] = [
  { label: '自动 · 跟随各卡默认', value: 'auto' },
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
  <NSelect
    v-model:value="value"
    :options="options"
    size="small"
    class="network-switcher network-switcher--wide"
    data-testid="network-switcher-wide"
  />
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
</template>

<style scoped>
.network-switcher--wide {
  width: 200px;
}
.network-switcher--narrow {
  display: none !important;
}

@media (max-width: 768px) {
  .network-switcher--wide {
    display: none !important;
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
