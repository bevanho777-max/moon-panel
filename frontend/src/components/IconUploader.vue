<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { compressImage } from '@/utils/imageCompress'
import { uploadIcon } from '@/api/icon'
import LucideIcon from '@/components/LucideIcon.vue'

const props = defineProps<{
  /** Currently associated icon ref (e.g. "upload:public/icons/abc.webp") to show as preview */
  current?: string
}>()

const emit = defineEmits<{
  /** Fired after successful upload. iconRef is ready to write into card.icon. */
  (e: 'uploaded', payload: { iconRef: string; url: string; deduped: boolean }): void
}>()

const ACCEPT = 'image/png,image/jpeg,image/webp,image/gif'

const inputRef = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
const uploading = ref(false)
const previewUrl = ref<string | null>(null)
const message = useMessage()

function trigger() {
  if (uploading.value) return
  inputRef.value?.click()
}

function onSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // allow re-selecting same file later
  if (file) handleFile(file)
}

function onDrop(e: DragEvent) {
  dragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) handleFile(file)
}

async function handleFile(file: File) {
  if (!file.type.startsWith('image/')) {
    message.error('请选择图片文件（PNG / JPEG / WebP / GIF）')
    return
  }
  // Pre-flight size check (server enforces 1MB but compression typically gets
  // far below that — warn early on truly huge inputs to fail fast)
  if (file.size > 10 * 1024 * 1024) {
    message.error('图片过大（>10MB），请先在系统中压缩')
    return
  }

  uploading.value = true
  try {
    const compressed = await compressImage(file)
    const result = await uploadIcon(compressed.blob, compressed.filename)
    previewUrl.value = result.url
    emit('uploaded', {
      iconRef: result.icon,
      url: result.url,
      deduped: result.deduped,
    })
    const ratio = compressed.originalSize > 0
      ? Math.round((compressed.compressedSize / compressed.originalSize) * 100)
      : 100
    if (result.deduped) {
      message.success('已存在相同图标，已复用')
    } else {
      message.success(
        `上传成功（压缩到 ${formatSize(compressed.compressedSize)} · ${ratio}% of 原图）`,
      )
    }
  } catch (e) {
    if (e instanceof ApiError) {
      message.error(`上传失败：${e.message}`)
    } else if (e instanceof Error) {
      message.error(`处理失败：${e.message}`)
    } else {
      message.error('上传失败')
    }
  } finally {
    uploading.value = false
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

// Three preview modes based on the current icon ref:
//   1. Fresh URL from this session's upload → show <img>
//   2. props.current = "upload:..." → show <img> from /uploads/...
//   3. props.current = "lucide:..." → show <LucideIcon>
//   4. props.current = "https://..." → show <img> with onerror fallback (Phase 3a-2)
//   5. otherwise → empty drop zone
type PreviewMode = 'image' | 'lucide' | 'none'

const previewMode = computed<PreviewMode>(() => {
  if (previewUrl.value) return 'image'
  if (!props.current) return 'none'
  if (props.current.startsWith('upload:')) return 'image'
  if (props.current.startsWith('lucide:')) return 'lucide'
  if (/^https?:\/\//i.test(props.current)) return 'image'
  return 'none'
})

const previewImageUrl = computed(() => {
  if (previewUrl.value) return previewUrl.value
  if (!props.current) return ''
  if (props.current.startsWith('upload:')) {
    return '/uploads/' + props.current.slice('upload:'.length)
  }
  if (/^https?:\/\//i.test(props.current)) return props.current
  return ''
})

const previewLucideName = computed(() =>
  props.current?.startsWith('lucide:') ? props.current.slice('lucide:'.length) : '',
)
</script>

<template>
  <div
    class="icon-uploader"
    :class="{
      'icon-uploader--dragging': dragging,
      'icon-uploader--uploading': uploading,
    }"
    @click="trigger"
    @dragover.prevent="dragging = true"
    @dragleave.prevent="dragging = false"
    @drop.prevent="onDrop"
  >
    <input
      ref="inputRef"
      type="file"
      :accept="ACCEPT"
      style="display: none"
      @change="onSelect"
    />
    <div v-if="uploading" class="icon-uploader__state">
      <span class="icon-uploader__hint">压缩 + 上传中…</span>
    </div>
    <template v-else>
      <div v-if="previewMode === 'image'" class="icon-uploader__state icon-uploader__preview">
        <img :src="previewImageUrl" alt="icon preview" />
        <span class="icon-uploader__hint">点击或拖拽以替换</span>
      </div>
      <div v-else-if="previewMode === 'lucide'" class="icon-uploader__state icon-uploader__preview">
        <span class="icon-uploader__lucide-box">
          <LucideIcon :name="previewLucideName" :size="32" />
        </span>
        <span class="icon-uploader__hint">Lucide: {{ previewLucideName }} · 点击或拖拽以替换</span>
      </div>
      <div v-else class="icon-uploader__state icon-uploader__empty">
        <span class="icon-uploader__main">点击或拖拽图片到此</span>
        <span class="icon-uploader__sub">PNG / JPEG / WebP / GIF · 自动压缩到 256×256 WebP · 上限 10MB</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.icon-uploader {
  border: 1.5px dashed rgba(255, 255, 255, 0.15);
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
  transition: border-color 150ms ease, background 150ms ease;
  background: rgba(255, 255, 255, 0.02);
  user-select: none;
}
.icon-uploader:hover {
  border-color: rgba(91, 141, 239, 0.5);
  background: rgba(91, 141, 239, 0.04);
}
.icon-uploader--dragging {
  border-color: #5b8def;
  background: rgba(91, 141, 239, 0.1);
}
.icon-uploader--uploading {
  cursor: wait;
  opacity: 0.7;
}
.icon-uploader__state {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 56px;
}
.icon-uploader__preview img {
  width: 48px;
  height: 48px;
  border-radius: 6px;
  object-fit: contain;
  background: rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}
.icon-uploader__lucide-box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 6px;
  background: rgba(91, 141, 239, 0.15);
  color: #5b8def;
  flex-shrink: 0;
}
.icon-uploader__empty {
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 4px 0;
}
.icon-uploader__main {
  font-size: 0.95rem;
  font-weight: 500;
}
.icon-uploader__sub {
  font-size: 0.75rem;
  opacity: 0.55;
}
.icon-uploader__hint {
  font-size: 0.85rem;
  opacity: 0.7;
}
</style>
