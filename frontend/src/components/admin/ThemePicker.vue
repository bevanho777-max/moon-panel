<script setup lang="ts">
// v0.2.1: theme-preset picker. Shows two thumbnail tiles (moon / risen)
// for the admin to switch the panel's visual identity. Switch is
// persisted via the ui store, which optimistically flips
// document.documentElement.dataset.theme — visual feedback is instant
// (no save button, same UX as the wallpaper picker tiles).
//
// Thumbnails come from the backend at /assets/themes/{moon,risen}-preview.svg
// (embedded SVGs, ~600 B each) so there's no extra build dependency.

import { useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { useUIStore } from '@/stores/ui'

interface ThemeOption {
  id: 'moon' | 'risen'
  label: string
  blurb: string
  preview: string
}

const ui = useUIStore()
const message = useMessage()

const options: ThemeOption[] = [
  {
    id: 'moon',
    label: 'Moon',
    blurb: '蓝紫月色 · 简洁现代 (默认)',
    preview: '/assets/themes/moon-preview.svg',
  },
  {
    id: 'risen',
    label: 'Risen',
    blurb: '金棕奢华 · 衬线字体 + 状态条',
    preview: '/assets/themes/risen-preview.svg',
  },
]

async function pick(id: 'moon' | 'risen') {
  if (id === ui.themePreset) return
  try {
    await ui.setThemePreset(id)
    message.success(id === 'moon' ? '已切换到 Moon' : '已切换到 Risen')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '切换主题失败')
  }
}
</script>

<template>
  <div class="tp">
    <div class="tp__row">
      <button
        v-for="opt in options"
        :key="opt.id"
        type="button"
        class="tp__tile"
        :class="{ 'tp__tile--active': ui.themePreset === opt.id }"
        @click="pick(opt.id)"
      >
        <img :src="opt.preview" :alt="opt.label" class="tp__preview" />
        <div class="tp__meta">
          <div class="tp__label">{{ opt.label }}</div>
          <div class="tp__blurb">{{ opt.blurb }}</div>
        </div>
        <span v-if="ui.themePreset === opt.id" class="tp__check" aria-label="当前">✓</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.tp__row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.tp__tile {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px 8px 8px;
  border: 2px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.02);
  cursor: pointer;
  transition: border-color 150ms ease, background-color 150ms ease;
  min-width: 240px;
  text-align: left;
}
.tp__tile:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.2);
}
.tp__tile--active {
  border-color: rgba(91, 141, 239, 0.7);
  background: rgba(91, 141, 239, 0.08);
}
.tp__preview {
  width: 120px;
  height: 80px;
  border-radius: 4px;
  flex-shrink: 0;
  display: block;
}
.tp__meta {
  flex: 1;
  min-width: 0;
}
.tp__label {
  font-size: 0.95rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.92);
}
.tp__blurb {
  margin-top: 2px;
  font-size: 0.75rem;
  color: rgba(255, 255, 255, 0.55);
}
.tp__check {
  position: absolute;
  top: 6px;
  right: 8px;
  font-size: 0.85rem;
  color: rgba(91, 141, 239, 0.95);
}
</style>
