<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDivider,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSpace,
  NSpin,
  useMessage,
} from 'naive-ui'
import {
  type City,
  loadCityCatalog,
  searchCities,
  validateCustomCity,
} from '@/utils/citySearch'

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'pick', city: City): void
}>()

const message = useMessage()

const catalog = ref<City[]>([])
const catalogLoading = ref(false)
const query = ref('')

// Custom-add fallback fields
const showCustom = ref(false)
const customForm = ref<Partial<City>>({
  name_cn: '',
  name_en: '',
  tz: '',
  lat: undefined,
  lon: undefined,
})

const candidates = computed(() => searchCities(query.value, catalog.value, 12))

onMounted(async () => {
  catalogLoading.value = true
  try {
    catalog.value = await loadCityCatalog()
  } finally {
    catalogLoading.value = false
  }
})

function close() {
  emit('update:show', false)
  // Reset state on close so next open is clean
  query.value = ''
  showCustom.value = false
  customForm.value = { name_cn: '', name_en: '', tz: '', lat: undefined, lon: undefined }
}

function pick(c: City) {
  // Defensive shape-strip: candidates flowing in from searchCities() are
  // CitySearchHits (City + score). emit('pick', c) typed as City, but the
  // runtime object retains score, which then gets serialized into
  // widget.cities JSON. Strip back down to the City shape so saved data
  // stays minimal — score is a search-time computation, not user data.
  const clean: City = {
    name_cn: c.name_cn,
    name_en: c.name_en,
    tz: c.tz,
    lat: c.lat,
    lon: c.lon,
  }
  emit('pick', clean)
  close()
}

function submitCustom() {
  const err = validateCustomCity(customForm.value)
  if (err) {
    message.warning(err)
    return
  }
  pick({
    name_cn: customForm.value.name_cn?.trim() || customForm.value.name_en!.trim(),
    name_en: customForm.value.name_en?.trim() || customForm.value.name_cn!.trim(),
    tz: customForm.value.tz!.trim(),
    lat: customForm.value.lat!,
    lon: customForm.value.lon!,
  })
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="添加城市"
    style="max-width: 480px"
    @update:show="emit('update:show', $event)"
  >
    <NSpace vertical :size="12">
      <NInput
        v-model:value="query"
        placeholder="搜索城市名（北京 / Tokyo / New York...）"
        clearable
        :disabled="catalogLoading"
      />

      <NSpin :show="catalogLoading">
        <div v-if="!query.trim()" class="cp__hint">
          输入城市名搜索（支持中英文）。catalog 含 {{ catalog.length }} 个全球主要城市。
        </div>
        <div v-else-if="candidates.length === 0" class="cp__empty">
          <NAlert type="info" :show-icon="false" style="margin-bottom: 8px">
            找不到「{{ query }}」？下方手动添加 4 个字段（城市名 / 时区 / 经纬度）。
          </NAlert>
          <NButton size="small" @click="showCustom = true" :disabled="showCustom">
            手动添加
          </NButton>
        </div>
        <ul v-else class="cp__list">
          <li
            v-for="c in candidates"
            :key="c.name_en"
            class="cp__row"
            @click="pick(c)"
          >
            <span class="cp__cn">{{ c.name_cn }}</span>
            <span class="cp__en">{{ c.name_en }}</span>
            <span class="cp__tz">{{ c.tz }}</span>
          </li>
        </ul>
      </NSpin>

      <template v-if="showCustom">
        <NDivider style="margin: 8px 0">手动添加</NDivider>
        <NForm size="small" :show-feedback="false">
          <NFormItem label="中文名">
            <NInput v-model:value="customForm.name_cn" placeholder="例如：雷克雅未克" />
          </NFormItem>
          <NFormItem label="英文名">
            <NInput v-model:value="customForm.name_en" placeholder="例如：Reykjavik" />
          </NFormItem>
          <NFormItem label="时区">
            <NInput v-model:value="customForm.tz" placeholder="IANA 名，如 Atlantic/Reykjavik" />
          </NFormItem>
          <NFormItem label="纬度">
            <NInputNumber
              v-model:value="customForm.lat"
              :min="-90"
              :max="90"
              :precision="4"
              placeholder="-90 到 90"
              style="width: 100%"
            />
          </NFormItem>
          <NFormItem label="经度">
            <NInputNumber
              v-model:value="customForm.lon"
              :min="-180"
              :max="180"
              :precision="4"
              placeholder="-180 到 180"
              style="width: 100%"
            />
          </NFormItem>
          <NSpace justify="end">
            <NButton size="small" @click="showCustom = false">取消手动</NButton>
            <NButton size="small" type="primary" @click="submitCustom">添加</NButton>
          </NSpace>
        </NForm>
      </template>
    </NSpace>
  </NModal>
</template>

<style scoped>
.cp__hint,
.cp__empty {
  padding: 12px 4px;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.55);
}
.cp__list {
  margin: 0;
  padding: 0;
  list-style: none;
  max-height: 360px;
  overflow-y: auto;
}
.cp__row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 120ms ease;
}
.cp__row:hover {
  background: rgba(91, 141, 239, 0.1);
}
.cp__cn {
  font-weight: 500;
  color: rgba(255, 255, 255, 0.92);
  min-width: 80px;
}
.cp__en {
  font-size: 0.85rem;
  opacity: 0.65;
  flex: 1;
  min-width: 0;
}
.cp__tz {
  font-family: monospace;
  font-size: 0.75rem;
  opacity: 0.5;
  flex-shrink: 0;
}
</style>
