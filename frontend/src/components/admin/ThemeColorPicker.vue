<script setup lang="ts">
import { ref, watch } from 'vue'
import { NAlert, NButton, NColorPicker, NSpace, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { THEME_PRESETS, useUIStore } from '@/stores/ui'

const ui = useUIStore()
const message = useMessage()

// Local copy for live preview while the picker swatch is being dragged. The
// store keeps the optimistic value via previewThemePrimary; only on close /
// confirm do we PUT to /admin/settings via setThemePrimary.
const draft = ref<string>(ui.themePrimary ?? THEME_PRESETS[0].hex)
const dirty = ref(false)

// Keep draft in sync if another tab / restore changes the store value.
watch(
  () => ui.themePrimary,
  (v) => {
    if (!dirty.value) draft.value = v ?? THEME_PRESETS[0].hex
  },
)

async function applyPreset(hex: string) {
  draft.value = hex
  dirty.value = false
  try {
    await ui.setThemePrimary(hex)
    message.success(`已应用 ${hex}`)
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  }
}

function onPickerUpdate(hex: string) {
  // Strip alpha if NColorPicker emits an 8-digit hex; we only persist RGB.
  const rgb = hex.length > 7 ? hex.slice(0, 7) : hex
  draft.value = rgb
  dirty.value = true
  ui.previewThemePrimary(rgb)
}

async function commitCustom() {
  try {
    await ui.setThemePrimary(draft.value)
    dirty.value = false
    message.success('已应用自定义色')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  }
}

async function reset() {
  try {
    await ui.setThemePrimary(null)
    draft.value = THEME_PRESETS[0].hex
    dirty.value = false
    message.success('已恢复默认蓝')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '重置失败')
  }
}
</script>

<template>
  <NSpace vertical :size="14">
    <NAlert type="info" :show-icon="false">
      调整 NaiveUI 主题主色 — 影响按钮 / 链接 / 选中态 / 顶栏菜单 active 等。
      <br />
      5 个预设色快速切换；下方取色器可微调。重置 = 恢复默认蓝。
    </NAlert>

    <!-- 5 preset buttons -->
    <div class="tc__presets">
      <button
        v-for="p in THEME_PRESETS"
        :key="p.hex"
        type="button"
        class="tc__preset"
        :class="{ 'tc__preset--active': ui.themePrimary?.toLowerCase() === p.hex.toLowerCase() }"
        :style="{ backgroundColor: p.hex }"
        :title="`${p.name} ${p.hex}`"
        @click="applyPreset(p.hex)"
      >
        <span>{{ p.name }}</span>
      </button>
    </div>

    <!-- Custom color picker -->
    <div class="tc__custom">
      <span class="tc__label">自定义</span>
      <NColorPicker
        :value="draft"
        :modes="['hex']"
        :show-alpha="false"
        :swatches="THEME_PRESETS.map((p) => p.hex)"
        style="width: 220px"
        @update:value="onPickerUpdate"
      />
      <NButton
        v-if="dirty"
        size="small"
        type="primary"
        @click="commitCustom"
      >
        应用
      </NButton>
    </div>

    <!-- Reset -->
    <NSpace>
      <NButton
        :disabled="ui.themePrimary === null"
        tertiary
        @click="reset"
      >
        恢复默认色
      </NButton>
    </NSpace>
  </NSpace>
</template>

<style scoped>
.tc__presets {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.tc__preset {
  width: 64px;
  height: 36px;
  border-radius: 6px;
  border: 2px solid rgba(255, 255, 255, 0.12);
  cursor: pointer;
  font-size: 0.78rem;
  color: #fff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.55);
  font-weight: 500;
  transition: border-color 0.15s, transform 0.1s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.tc__preset:hover {
  border-color: rgba(255, 255, 255, 0.55);
  transform: translateY(-1px);
}
.tc__preset--active {
  border-color: #fff;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.3);
}
.tc__custom {
  display: flex;
  align-items: center;
  gap: 12px;
}
.tc__label {
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.78);
}
</style>
