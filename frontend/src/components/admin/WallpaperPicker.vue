<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NPopconfirm,
  NSlider,
  NSpace,
  NSpin,
  useMessage,
} from 'naive-ui'
import { ApiError } from '@/api/client'
import {
  type UploadedWallpaper,
  compressWallpaperToWebp,
  deleteWallpaper,
  listWallpapers,
  uploadWallpaper,
} from '@/api/wallpaper'
import { useUIStore } from '@/stores/ui'

const ui = useUIStore()
const message = useMessage()

const uploads = ref<UploadedWallpaper[]>([])
const loading = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

// Live blur during slider drag — store keeps the optimistic value, real PUT
// is debounced to ~350 ms after the last update so we don't spam
// /admin/settings on every pixel of slider motion. Using a debounce instead
// of NSlider's @dragend keeps the behavior consistent for keyboard (arrow
// keys), touch, and mouse — all funnel through @update:value.
const blurDraft = ref(ui.blur)
let blurSaveTimer: ReturnType<typeof setTimeout> | null = null

const builtinItems = computed(() =>
  ui.builtins.map((id) => ({
    id,
    spec: `builtin:${id}`,
    url: `/assets/wallpapers/${id}.svg`,
    label: builtinLabel(id),
  })),
)

function builtinLabel(id: string): string {
  switch (id) {
    case 'night': return '夜空'
    case 'aurora': return '极光'
    case 'graphite': return '石墨'
    // v0.2.0 additions
    case 'galaxy': return '银河'
    case 'ocean': return '海洋'
    case 'sunset': return '日落'
    case 'mountain': return '雪山'
    default: return id
  }
}

const currentSpec = computed(() => ui.wallpaper)

async function refresh() {
  loading.value = true
  try {
    uploads.value = await listWallpapers()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载壁纸列表失败')
  } finally {
    loading.value = false
  }
}

async function pick(spec: string | null) {
  try {
    await ui.setWallpaper(spec)
    message.success(spec === null ? '已清除壁纸' : '已应用')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '应用失败')
  }
}

function onSliderUpdate(value: number) {
  blurDraft.value = value
  ui.previewBlur(value)
  if (blurSaveTimer) clearTimeout(blurSaveTimer)
  blurSaveTimer = setTimeout(() => {
    ui.setBlur(value).catch((e) => {
      message.error(e instanceof ApiError ? e.message : '保存模糊度失败')
    })
  }, 350)
}

function triggerUpload() {
  fileInput.value?.click()
}

async function onFileChosen(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = '' // reset so same file can re-pick later
  if (!file) return

  // Sanity: 20 MiB hard cap on the original (canvas can't load bigger comfortably).
  if (file.size > 20 * 1024 * 1024) {
    message.error('源文件过大（>20MB）—— 请先在本地裁剪压缩')
    return
  }
  if (!file.type.startsWith('image/')) {
    message.error('请选择图片文件')
    return
  }

  uploading.value = true
  try {
    const blob = await compressWallpaperToWebp(file)
    const result = await uploadWallpaper(blob)
    if (result.deduped) {
      message.info('该壁纸已存在，复用现有文件')
    } else {
      message.success(`已上传（${(result.size / 1024).toFixed(0)} KB）`)
    }
    await refresh()
    await ui.setWallpaper(result.wallpaper)
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : (e as Error).message || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function removeUpload(item: UploadedWallpaper) {
  try {
    await deleteWallpaper(item.hash)
    // If the deleted file was currently active, fall back to no wallpaper.
    if (currentSpec.value === item.wallpaper) {
      await ui.setWallpaper(null)
    }
    await refresh()
    message.success('已删除')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

onMounted(refresh)
</script>

<template>
  <NSpace vertical :size="16">
    <NAlert type="info" :show-icon="false">
      内置 3 张轻量 SVG 壁纸（任意分辨率清晰），或上传自己的图片（自动压到 1920×1080 webp）。
      <br />
      模糊度 0-20px：用于让前景内容更突出。重置 = 不应用壁纸。
    </NAlert>

    <!-- Builtin grid -->
    <div class="wp__section">
      <div class="wp__label">内置</div>
      <div class="wp__grid">
        <button
          v-for="b in builtinItems"
          :key="b.id"
          type="button"
          class="wp__tile"
          :class="{ 'wp__tile--active': currentSpec === b.spec }"
          :style="{ backgroundImage: `url(${b.url})` }"
          :title="b.label"
          @click="pick(b.spec)"
        >
          <span class="wp__tile-label">{{ b.label }}</span>
        </button>
      </div>
    </div>

    <!-- User uploads -->
    <div class="wp__section">
      <div class="wp__row">
        <div class="wp__label">已上传</div>
        <NButton size="small" type="primary" :loading="uploading" @click="triggerUpload">
          上传图片
        </NButton>
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          style="display:none"
          @change="onFileChosen"
        />
      </div>
      <NSpin :show="loading">
        <div v-if="uploads.length === 0" class="wp__empty">
          还没有上传过壁纸。点"上传图片"添加。
        </div>
        <div v-else class="wp__grid">
          <div
            v-for="u in uploads"
            :key="u.hash"
            class="wp__tile-wrap"
          >
            <button
              type="button"
              class="wp__tile"
              :class="{ 'wp__tile--active': currentSpec === u.wallpaper }"
              :style="{ backgroundImage: `url(${u.url})` }"
              :title="`${(u.size / 1024).toFixed(0)} KB`"
              @click="pick(u.wallpaper)"
            />
            <NPopconfirm
              :positive-text="'删除'"
              :negative-text="'取消'"
              @positive-click="removeUpload(u)"
            >
              <template #trigger>
                <button type="button" class="wp__delete" title="删除">×</button>
              </template>
              确认删除该壁纸？
              <span v-if="currentSpec === u.wallpaper">（当前正在使用，删除后将清除壁纸）</span>
            </NPopconfirm>
          </div>
        </div>
      </NSpin>
    </div>

    <!-- v0.2.0: blur slider hidden. v0.1.7 removed CSS filter on
         wallpaper-layer (root cause of global continuous repaint, see
         App.vue notes), so this slider has no visible effect anymore.
         The ui.wallpaper_blur setting + backend column are kept to
         avoid a db migration; a future release can decide whether to
         bake blur into the wallpaper at upload time, then remove the
         field. Until then, the slider is just hidden. -->

    <!-- Reset -->
    <div class="wp__section">
      <NSpace>
        <NButton :disabled="!currentSpec" tertiary @click="pick(null)">
          清除壁纸
        </NButton>
      </NSpace>
    </div>
  </NSpace>
</template>

<style scoped>
.wp__section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.wp__row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.wp__label {
  font-size: 0.85rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.78);
  flex: 1;
}
.wp__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}
.wp__tile-wrap {
  position: relative;
}
.wp__tile {
  width: 100%;
  aspect-ratio: 16 / 9;
  border: 2px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  background-size: cover;
  background-position: center center;
  cursor: pointer;
  position: relative;
  padding: 0;
  display: flex;
  align-items: flex-end;
  justify-content: flex-start;
  overflow: hidden;
  transition: border-color 0.15s, transform 0.1s;
}
.wp__tile:hover {
  border-color: rgba(255, 255, 255, 0.4);
  transform: translateY(-1px);
}
.wp__tile--active {
  border-color: var(--primary, #5b8def);
  box-shadow: 0 0 0 2px rgba(91, 141, 239, 0.25);
}
.wp__tile-label {
  font-size: 0.75rem;
  color: rgba(255, 255, 255, 0.85);
  background: rgba(0, 0, 0, 0.45);
  padding: 2px 8px;
  border-radius: 0 6px 0 0;
  user-select: none;
}
.wp__delete {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
}
.wp__tile-wrap:hover .wp__delete {
  opacity: 1;
}
.wp__delete:hover {
  background: #d33;
}
.wp__empty {
  font-size: 0.85rem;
  opacity: 0.55;
  padding: 12px 0;
}
.wp__blur-value {
  font-family: monospace;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.65);
}
.wp__hint {
  font-size: 0.78rem;
  opacity: 0.55;
}
</style>
