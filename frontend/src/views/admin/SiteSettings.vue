<script setup lang="ts">
import { computed, h, onMounted, ref, type VNode } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NPopconfirm,
  NSpace,
  NSpin,
  NSwitch,
  useMessage,
} from 'naive-ui'
import { RouterLink } from 'vue-router'
import { Star } from 'lucide-vue-next'
import SortableTable from '@/components/SortableTable.vue'
import { ApiError } from '@/api/client'
import {
  type SearchEngine,
  type SearchEngineWritePayload,
  createSearchEngine,
  deleteSearchEngine,
  listSearchEngines,
  reorderSearchEngines,
  updateSearchEngine,
} from '@/api/searchEngine'
import { getSettings, updateSettings } from '@/api/setting'
import LucideIcon from '@/components/LucideIcon.vue'
import CityPicker from '@/components/admin/CityPicker.vue'
import TwoFactorEnrollModal from '@/components/admin/TwoFactorEnrollModal.vue'
import TwoFactorDisableModal from '@/components/admin/TwoFactorDisableModal.vue'
import TrustedNetworkModal from '@/components/admin/TrustedNetworkModal.vue'
import BackupRestoreModal from '@/components/admin/BackupRestoreModal.vue'
import WallpaperPicker from '@/components/admin/WallpaperPicker.vue'
import ThemeColorPicker from '@/components/admin/ThemeColorPicker.vue'
import StatefulInput from '@/components/StatefulInput.vue'
import { exportBackupJSON, exportBackupZip } from '@/api/backup'
import { showStatefulInputHintOnce } from '@/utils/statefulInputHint'
import { getMe } from '@/api/auth'
import { listTrustedIPs, deleteTrustedIP, type TrustedIPEntry } from '@/api/security'
import type { City } from '@/utils/citySearch'
import { useUIStore } from '@/stores/ui'
import draggable from 'vuedraggable'
import ThemePicker from '@/components/admin/ThemePicker.vue'

const message = useMessage()
const ui = useUIStore()

// v0.2.0: site title editor. Draft mirrors the live store value; save on
// blur or explicit click. Empty submission resets to backend default
// "Moon Panel" (the store's setSiteTitle handles that fallback).
const siteTitleDraft = ref(ui.siteTitle)
async function saveSiteTitle() {
  try {
    await ui.setSiteTitle(siteTitleDraft.value)
    message.success('站点名已保存')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  }
}

const engines = ref<SearchEngine[]>([])
const loading = ref(false)

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const editorForm = ref<Required<SearchEngineWritePayload>>(emptyForm())
const editorOriginal = ref({ name: '', url_template: '', icon: '' })
const submitting = ref(false)

const editorTitle = computed(() => (editorMode.value === 'create' ? '新建搜索引擎' : '编辑搜索引擎'))

function emptyForm(): Required<SearchEngineWritePayload> {
  return {
    name: '',
    url_template: '',
    icon: '',
    is_default: false,
    sort: 0,
  }
}

async function refresh() {
  loading.value = true
  try {
    engines.value = await listSearchEngines()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editorMode.value = 'create'
  editingId.value = null
  editorForm.value = emptyForm()
  editorOriginal.value = { name: '', url_template: '', icon: '' }
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

function openEdit(engine: SearchEngine) {
  editorMode.value = 'edit'
  editingId.value = engine.id
  editorForm.value = {
    name: engine.name,
    url_template: engine.url_template,
    icon: engine.icon,
    is_default: engine.is_default,
    sort: engine.sort,
  }
  editorOriginal.value = {
    name: engine.name,
    url_template: engine.url_template,
    icon: engine.icon,
  }
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

async function submit() {
  const f = editorForm.value
  if (!f.name.trim()) {
    message.warning('名称不能为空')
    return
  }
  if (!f.url_template.trim()) {
    message.warning('URL 模板不能为空')
    return
  }
  // Bug 2 fix: regex was theoretically correct but rejected {query} in
  // production. Switched to .includes() — unambiguous, simpler.
  const tpl = f.url_template.trim()
  if (!tpl.includes('{q}') && !tpl.includes('{query}')) {
    message.warning('URL 模板必须包含 {q} 或 {query} 占位符')
    return
  }
  submitting.value = true
  try {
    if (editorMode.value === 'create') {
      await createSearchEngine(f)
      message.success('已创建')
    } else if (editingId.value !== null) {
      await updateSearchEngine(editingId.value, f)
      message.success('已更新')
    }
    editorOpen.value = false
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function setAsDefault(engine: SearchEngine) {
  if (engine.is_default) return // already default
  try {
    await updateSearchEngine(engine.id, { is_default: true })
    message.success(`已将"${engine.name}"设为默认`)
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '设置默认失败')
  }
}

async function handleDelete(engine: SearchEngine) {
  try {
    await deleteSearchEngine(engine.id)
    message.success(`已删除"${engine.name}"`)
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// Bug 1 + Bug 3 fix: row icon renderer now handles all 4 prefix forms (http /
// upload / lucide / empty), trims input defensively (DB rows occasionally have
// trailing whitespace), and falls through to a clear "?" placeholder for
// unknown formats rather than silently failing.
// v0.2.14 P0 b: size 参数化 — PC NDataTable 默认 24, mobile 1 行卡片 22.
// LucideIcon 内 size 按比例 (~0.58 = 14/24) 缩, 跟 v0.2.13 renderIconThumb 同套路.
function renderEngineIcon(rawIcon: string, size = 24): VNode {
  const dim = `${size}px`
  const lucideSize = Math.round(size * 0.58)
  const baseStyle = `width:${dim};height:${dim};border-radius:4px;flex-shrink:0;display:inline-flex;align-items:center;justify-content:center;background:rgba(255,255,255,0.05)`
  const icon = (rawIcon ?? '').trim()
  if (!icon) {
    return h('div', { style: `${baseStyle};color:rgba(255,255,255,0.3)` }, '—')
  }
  if (/^https?:\/\//i.test(icon)) {
    return h('img', {
      src: icon,
      style: `${baseStyle};object-fit:contain`,
      onError: (e: Event) => {
        const img = e.target as HTMLImageElement
        img.style.display = 'none'
      },
    })
  }
  if (icon.startsWith('upload:')) {
    return h('img', {
      src: '/uploads/' + icon.slice('upload:'.length),
      style: `${baseStyle};object-fit:contain`,
      onError: (e: Event) => {
        const img = e.target as HTMLImageElement
        img.style.display = 'none'
      },
    })
  }
  if (icon.startsWith('lucide:')) {
    return h(
      'div',
      { style: `${baseStyle};color:#5b8def;background:rgba(91,141,239,0.15)`, title: icon },
      h(LucideIcon, { name: icon.slice('lucide:'.length), size: lucideSize }),
    )
  }
  return h('div', { style: `${baseStyle};color:rgba(255,193,77,0.7);font-size:11px`, title: icon }, '?')
}

// v0.2.18: SortableTable interface 包装 (single list -> [{id:0, name:'', items}]).
const enginesForSortable = computed(() => [
  { id: 0, name: '', items: engines.value },
])

// v0.2.16 P0 b: 拖拽结束 → 重算 sort = (i+1)*10, 立即 PUT (auto-save). 失败时
// reload (server state rollback). Mirrors v0.2.15 Cards.vue + v0.2.16 Groups.vue
// onCardReorder/onGroupReorder.
async function onEngineReorder() {
  const items = engines.value.map((e, i) => ({
    id: e.id,
    sort: (i + 1) * 10,
  }))
  if (items.length === 0) return
  try {
    await reorderSearchEngines(items)
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '排序保存失败, 已撤销')
    await refresh()
  }
}

// ───────────── Widget settings (cities + temp unit) ─────────────

const cities = ref<City[]>([])
const tempUnit = ref<'C' | 'F'>('C')
const cityPickerOpen = ref(false)
const widgetLoading = ref(false)
const MAX_CITIES = 5

async function loadWidgetSettings() {
  widgetLoading.value = true
  try {
    const settings = await getSettings()
    if (settings['widget.cities']) {
      try {
        const parsed = JSON.parse(settings['widget.cities'])
        if (Array.isArray(parsed)) cities.value = parsed
      } catch {
        cities.value = []
      }
    }
    if (settings['widget.temp_unit'] === 'F') tempUnit.value = 'F'
    else tempUnit.value = 'C'
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载设置失败')
  } finally {
    widgetLoading.value = false
  }
}

async function saveCities() {
  try {
    await updateSettings({ 'widget.cities': JSON.stringify(cities.value) })
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存城市失败')
  }
}

async function saveTempUnit(v: 'C' | 'F') {
  tempUnit.value = v
  try {
    await updateSettings({ 'widget.temp_unit': v })
    message.success(`已切换到 °${v}`)
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  }
}

// ───────────── Network probe URL (v0.2.23) ─────────────
// Empty = auto-sample from card internal URLs; explicit URL = probe that
// endpoint to decide LAN vs WAN. Backend validates http/https prefix +
// max length 200 — surface the failure inline if rejected.
const probeUrlDraft = ref('')
const probeUrlOriginal = ref('')
const probeUrlSaving = ref(false)

async function loadNetworkSettings() {
  try {
    const settings = await getSettings()
    probeUrlDraft.value = settings['network.probe_url'] ?? ''
    probeUrlOriginal.value = probeUrlDraft.value
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载网络设置失败')
  }
}

async function saveProbeUrl() {
  const v = probeUrlDraft.value.trim()
  probeUrlSaving.value = true
  try {
    await updateSettings({ 'network.probe_url': v })
    probeUrlOriginal.value = v
    probeUrlDraft.value = v
    message.success(v === '' ? '已清空，将自动从卡片内网 URL 取样' : '已保存')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    probeUrlSaving.value = false
  }
}

function handleCityPick(city: City) {
  if (cities.value.length >= MAX_CITIES) {
    message.warning(`最多 ${MAX_CITIES} 个城市，请先删除现有的再添加`)
    return
  }
  // Dedup by tz+lat+lon (city objects may have different name_cn but same place)
  const dup = cities.value.find(
    (c) => c.tz === city.tz && Math.abs(c.lat - city.lat) < 0.01 && Math.abs(c.lon - city.lon) < 0.01,
  )
  if (dup) {
    message.warning(`已经添加了「${dup.name_cn}」`)
    return
  }
  cities.value.push(city)
  saveCities()
  message.success(`已添加「${city.name_cn}」`)
}

function removeCity(idx: number) {
  const removed = cities.value.splice(idx, 1)[0]
  saveCities()
  message.success(`已移除「${removed.name_cn}」`)
}

// 2FA / TOTP state — fetched from /auth/me
const totpEnabled = ref(false)
const totpEnrollOpen = ref(false)
const totpDisableOpen = ref(false)

async function refreshTOTPState() {
  try {
    const me = await getMe()
    totpEnabled.value = !!me.totp_enabled
  } catch {
    // Silent — page already shows inline errors via other handlers.
  }
}

function on2FAEnabled() {
  totpEnabled.value = true
  message.success('两步验证已启用')
}

function on2FADisabled() {
  totpEnabled.value = false
}

// Trusted IP whitelist state
const trustedIPs = ref<TrustedIPEntry[]>([])
const trustedAddOpen = ref(false)
const trustedLoading = ref(false)

async function loadTrustedIPs() {
  trustedLoading.value = true
  try {
    trustedIPs.value = await listTrustedIPs()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载受信网络失败')
  } finally {
    trustedLoading.value = false
  }
}

async function removeTrustedIP(cidr: string) {
  try {
    await deleteTrustedIP(cidr)
    trustedIPs.value = trustedIPs.value.filter((e) => e.cidr !== cidr)
    message.success(`已移除 ${cidr}`)
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

function onTrustedAdded() {
  loadTrustedIPs()
}

const backupRestoreOpen = ref(false)
function onRestored() {
  // Force a full page reload so all stores re-fetch from the restored DB.
  // Simpler than invalidating each store individually; backups are rare events.
  window.location.reload()
}

onMounted(() => {
  refresh()
  loadWidgetSettings()
  loadNetworkSettings()
  refreshTOTPState()
  loadTrustedIPs()
})
</script>

<template>
  <NSpace vertical :size="24">
    <!-- v0.2.0: Section 0 — Site identity. Goes at top so the most
         immediately-visible customization is also the easiest to find. -->
    <NCard title="站点信息">
      <NSpace vertical :size="20" style="width: 100%">
        <NSpace align="center" :size="12" style="width: 100%">
          <span class="ws__label" style="min-width: 5em">站点名称</span>
          <NInput
            v-model:value="siteTitleDraft"
            placeholder="Moon Panel"
            maxlength="40"
            show-count
            style="max-width: 320px"
          />
          <NButton
            type="primary"
            :disabled="siteTitleDraft.trim() === ui.siteTitle"
            @click="saveSiteTitle"
          >保存</NButton>
        </NSpace>
        <div>
          <div class="ws__label" style="margin-bottom: 8px">主题预设</div>
          <ThemePicker />
        </div>
      </NSpace>
    </NCard>

    <!-- Section 1: Search Engines (active) -->
    <NCard>
      <template #header>
        <NSpace align="center" justify="space-between" style="width:100%">
          <span>搜索引擎</span>
          <NButton type="primary" @click="openCreate">新建引擎</NButton>
        </NSpace>
      </template>
      <NSpin :show="loading">
        <!-- v0.2.18: SortableTable 抽象 (Rule of Three). single list (engines 无 group
             concept), 包装. 保留 v0.2.14 ⭐/☆ Star icon (PC + Mobile 统一) +
             URL 模板 PC 显示 (mobile @media 隐藏). -->
        <SortableTable
          v-if="engines.length > 0"
          :groups="enginesForSortable"
          group-name="engines"
          :show-group-headers="false"
          @reorder="onEngineReorder"
        >
          <template #item="{ item: engine }">
            <component :is="renderEngineIcon(engine.icon, 22)" />
            <span class="engines-cell__title">{{ engine.name }}</span>
            <span class="engines-cell__url">{{ engine.url_template }}</span>
            <button
              type="button"
              class="engines-cell__star"
              :class="{ 'engines-cell__star--active': engine.is_default }"
              :title="engine.is_default ? '默认引擎' : '设为默认'"
              :disabled="engine.is_default"
              @click="setAsDefault(engine)"
            >
              <Star :size="18" :fill="engine.is_default ? 'currentColor' : 'none'" />
            </button>
            <span class="engines-cell__sort">{{ engine.sort }}</span>
            <div class="engines-cell__actions">
              <NButton size="small" @click="openEdit(engine)">编辑</NButton>
              <NPopconfirm
                :positive-text="'删除'"
                :negative-text="'取消'"
                @positive-click="handleDelete(engine)"
              >
                <template #trigger>
                  <NButton size="small" type="error" ghost>删除</NButton>
                </template>
                删除"{{ engine.name }}"？{{ engine.is_default ? '这是当前默认引擎。' : '' }}
              </NPopconfirm>
            </div>
          </template>
        </SortableTable>

        <NEmpty v-if="engines.length === 0" description="还没有搜索引擎" />
      </NSpin>
    </NCard>

    <!-- Section 2: Time display (active in Phase 3c) -->
    <NCard>
      <template #header>
        <NSpace align="center" justify="space-between" style="width:100%">
          <span>时间显示</span>
          <NButton
            type="primary"
            size="small"
            :disabled="cities.length >= MAX_CITIES"
            @click="cityPickerOpen = true"
          >
            添加城市（{{ cities.length }} / {{ MAX_CITIES }}）
          </NButton>
        </NSpace>
      </template>
      <NSpin :show="widgetLoading">
        <NSpace vertical :size="14">
          <div v-if="cities.length === 0" class="ws__empty">
            还没有城市。点右上角"添加城市"开始。主页 hero 区会展示这些城市的当地时间和气温。
          </div>
          <!-- v0.2.0: drag-to-reorder. Cities live as a JSON array in the
               widget.cities setting (single row, ordered), so persisting a
               new order just means re-stringifying after the drag — same
               saveCities() that ran on add/remove already. -->
          <div v-else class="ws__cities">
            <draggable
              v-model="cities"
              :item-key="(c: City) => `${c.tz}_${c.lat}_${c.lon}`"
              handle=".ws__drag"
              ghost-class="ws__city--ghost"
              animation="150"
              @end="saveCities"
            >
              <template #item="{ element: c, index: idx }">
                <div class="ws__city">
                  <span class="ws__drag" title="拖拽排序">☰</span>
                  <span class="ws__cn">{{ c.name_cn }}</span>
                  <span class="ws__en">{{ c.name_en }}</span>
                  <span class="ws__tz">{{ c.tz }}</span>
                  <span class="ws__coords">{{ c.lat.toFixed(2) }}, {{ c.lon.toFixed(2) }}</span>
                  <!-- v0.2.6: combined detail line for the mobile grid layout
                       (en + tz + coords on one ellipsis-clipped row). The
                       three desktop-only spans above stay rendered for ≥769px;
                       @media in scoped style toggles display per breakpoint. -->
                  <span class="ws__detail">
                    {{ c.name_en }} · {{ c.tz }} · {{ c.lat.toFixed(2) }}, {{ c.lon.toFixed(2) }}
                  </span>
                  <span class="ws__remove">
                    <NPopconfirm
                      :positive-text="'移除'"
                      :negative-text="'取消'"
                      @positive-click="removeCity(idx)"
                    >
                      <template #trigger>
                        <NButton size="tiny" type="error" tertiary>移除</NButton>
                      </template>
                      移除「{{ c.name_cn }}」？
                    </NPopconfirm>
                  </span>
                </div>
              </template>
            </draggable>
          </div>
          <div class="ws__temp-unit">
            <span>气温单位：</span>
            <NButton
              size="small"
              :type="tempUnit === 'C' ? 'primary' : 'default'"
              :secondary="tempUnit !== 'C'"
              @click="saveTempUnit('C')"
            >°C 摄氏</NButton>
            <NButton
              size="small"
              :type="tempUnit === 'F' ? 'primary' : 'default'"
              :secondary="tempUnit !== 'F'"
              @click="saveTempUnit('F')"
            >°F 华氏</NButton>
          </div>
        </NSpace>
      </NSpin>
    </NCard>

    <!-- Section 2.5 (v0.2.23): Network auto-detection probe URL -->
    <NCard title="网络检测">
      <NSpace vertical :size="12">
        <NSpace align="center" :size="12" style="width: 100%">
          <span class="ws__label" style="min-width: 7em">内网检测 URL</span>
          <NInput
            v-model:value="probeUrlDraft"
            placeholder="http://192.168.x.x:port （留空则自动从卡片内网 URL 取样）"
            :maxlength="200"
            clearable
            style="max-width: 460px"
            :disabled="probeUrlSaving"
          />
          <NButton
            type="primary"
            :disabled="probeUrlDraft.trim() === probeUrlOriginal.trim()"
            :loading="probeUrlSaving"
            @click="saveProbeUrl"
          >保存</NButton>
        </NSpace>
        <div class="ws__help">
          用于自动检测当前是否在本地局域网。设置后探测此 URL 是否可达；留空时自动从卡片内网 URL 池取样。
          探测使用 no-cors 模式，1.5s 超时，不会向页面回传任何数据。
        </div>
      </NSpace>
    </NCard>

    <!-- Section 3: Two-Factor Authentication -->
    <NCard title="两步验证 (2FA)">
      <NSpace vertical :size="12">
        <NAlert v-if="totpEnabled" type="success" :show-icon="false">
          <b>已启用</b> — 登录时密码后还需输入 Authenticator 应用的 6 位验证码（或备份码）。
        </NAlert>
        <NAlert v-else type="warning" :show-icon="false">
          <b>未启用</b> — 公网部署强烈建议开启。开启后即使密码泄漏，攻击者也无法登录。
        </NAlert>

        <NSpace>
          <NButton
            v-if="!totpEnabled"
            type="primary"
            @click="totpEnrollOpen = true"
          >
            启用两步验证
          </NButton>
          <NButton
            v-else
            type="error"
            secondary
            @click="totpDisableOpen = true"
          >
            禁用两步验证
          </NButton>
        </NSpace>

        <div class="ws__totp-tip" v-if="!totpEnabled">
          需要在手机或密码管理器（Google Authenticator / 1Password / Bitwarden 等）安装 TOTP 应用。
        </div>
      </NSpace>
    </NCard>

    <!-- Section 4: Trusted networks (login lockout bypass) -->
    <NCard title="受信网络（CIDR 白名单）">
      <NSpace vertical :size="12">
        <NAlert type="info" :show-icon="false">
          列表内的 IP 段在密码登录 / 2FA 失败时**不计入锁定**（仍记审计日志）。
          家庭局域网、固定办公出口、4G 漫游网段等可加入这里，避免同 IP 攻击者把你自己也锁出去。
        </NAlert>

        <div v-if="trustedIPs.length === 0" class="ws__trusted-empty">
          还没有受信网络。点右上角「添加 CIDR」加入第一条。
        </div>
        <div v-else>
          <div v-for="e in trustedIPs" :key="e.cidr" class="ws__trusted-row">
            <code class="ws__trusted-cidr">{{ e.cidr }}</code>
            <span class="ws__trusted-note">{{ e.note || '—' }}</span>
            <NButton size="tiny" tertiary type="error" @click="removeTrustedIP(e.cidr)">
              删除
            </NButton>
          </div>
        </div>

        <NSpace>
          <NButton size="small" @click="trustedAddOpen = true">添加 CIDR</NButton>
          <RouterLink :to="{ name: 'admin-security' }" custom v-slot="{ navigate }">
            <NButton size="small" tertiary @click="navigate">查看当前锁定 IP →</NButton>
          </RouterLink>
        </NSpace>
      </NSpace>
    </NCard>

    <!-- Section 5: Wallpaper (Phase 2.5c) -->
    <NCard title="背景壁纸">
      <WallpaperPicker />
    </NCard>

    <!-- Section 6: Theme color (Phase 2.5c) -->
    <NCard title="主题色">
      <ThemeColorPicker />
    </NCard>

    <!-- Section 7: Backup / restore -->
    <NCard title="备份与恢复">
      <NSpace vertical :size="12">
        <NAlert type="info" :show-icon="false">
          导出当前 panel 的全部分组、卡片、搜索引擎和设置。<b>不包含</b>登录密码 / 2FA 密钥 / 审计日志。
          <br />
          ZIP 导出额外打包 <code>uploads/</code> 目录下的所有上传文件。
        </NAlert>
        <NSpace>
          <NButton @click="exportBackupJSON">📄 导出 JSON</NButton>
          <NButton @click="exportBackupZip">📦 导出 ZIP（含上传文件）</NButton>
          <NButton type="warning" secondary @click="backupRestoreOpen = true">从备份恢复...</NButton>
        </NSpace>
      </NSpace>
    </NCard>
  </NSpace>

  <TwoFactorEnrollModal
    v-model:show="totpEnrollOpen"
    @enabled="on2FAEnabled"
  />
  <TwoFactorDisableModal
    v-model:show="totpDisableOpen"
    @disabled="on2FADisabled"
  />
  <TrustedNetworkModal
    v-model:show="trustedAddOpen"
    @added="onTrustedAdded"
  />
  <BackupRestoreModal
    v-model:show="backupRestoreOpen"
    @restored="onRestored"
  />

  <!-- Editor Modal -->
  <NModal
    v-model:show="editorOpen"
    preset="card"
    :title="editorTitle"
    style="max-width:560px"
    :mask-closable="!submitting"
  >
    <NForm @submit.prevent="submit">
      <NFormItem label="名称" required>
        <StatefulInput
          v-model="editorForm.name"
          :original-value="editorOriginal.name"
          placeholder="例如：Google / 必应 / Searxng"
          :maxlength="64"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="URL 模板" required>
        <StatefulInput
          v-model="editorForm.url_template"
          :original-value="editorOriginal.url_template"
          placeholder="https://example.com/search?q={q}"
          :disabled="submitting"
        />
      </NFormItem>
      <NAlert
        type="default"
        :show-icon="false"
        style="margin: -8px 0 12px; font-size: 0.8rem"
      >
        必须包含 <code>{q}</code> 或 <code>{query}</code> 占位符。两者等价，前端搜索时会替换成 URL 编码后的关键词。
      </NAlert>
      <NFormItem label="图标 URL">
        <StatefulInput
          v-model="editorForm.icon"
          :original-value="editorOriginal.icon"
          placeholder="https://example.com/favicon.ico（可选）"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="设为默认引擎">
        <NSwitch v-model:value="editorForm.is_default" :disabled="submitting" />
      </NFormItem>
      <NFormItem label="排序权重">
        <NInputNumber
          v-model:value="editorForm.sort"
          :step="10"
          :disabled="submitting"
          style="width:100%"
        />
      </NFormItem>
      <NSpace justify="end">
        <NButton @click="editorOpen = false" :disabled="submitting">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="submit">
          {{ editorMode === 'create' ? '创建' : '保存' }}
        </NButton>
      </NSpace>
    </NForm>
  </NModal>

  <CityPicker v-model:show="cityPickerOpen" @pick="handleCityPick" />
</template>

<style scoped>
.ws__empty {
  padding: 20px 4px;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.5);
}
.ws__totp-tip {
  font-size: 0.78rem;
  opacity: 0.55;
}
/* v0.2.23: probe URL help text — same dimming as .ws__totp-tip but kept
   as a distinct class so future copy/layout tweaks don't bleed across. */
.ws__help {
  font-size: 0.78rem;
  opacity: 0.55;
  line-height: 1.5;
}
.ws__trusted-empty {
  font-size: 0.85rem;
  opacity: 0.55;
  padding: 8px 0;
}
.ws__trusted-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.03);
  margin-bottom: 4px;
}
.ws__trusted-cidr {
  font-family: monospace;
  background: rgba(255, 255, 255, 0.06);
  padding: 2px 8px;
  border-radius: 3px;
  user-select: all;
  min-width: 140px;
}
.ws__trusted-note {
  flex: 1;
  font-size: 0.85rem;
  opacity: 0.75;
}
.ws__cities {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ws__city {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);
}
/* v0.2.0: drag handle for the cities list (Task E). */
.ws__drag {
  cursor: grab;
  color: rgba(255, 255, 255, 0.35);
  font-size: 1.05rem;
  user-select: none;
  flex-shrink: 0;
  transition: color 120ms ease;
}
.ws__drag:hover {
  color: rgba(255, 255, 255, 0.8);
}
.ws__drag:active {
  cursor: grabbing;
}
.ws__city--ghost {
  opacity: 0.4;
  background: rgba(91, 141, 239, 0.12);
}
.ws__cn {
  font-weight: 500;
  color: rgba(255, 255, 255, 0.92);
  min-width: 90px;
}
.ws__en {
  flex: 1;
  font-size: 0.85rem;
  opacity: 0.6;
  min-width: 0;
}
.ws__tz {
  font-family: monospace;
  font-size: 0.75rem;
  opacity: 0.55;
}
.ws__coords {
  font-family: monospace;
  font-size: 0.75rem;
  opacity: 0.4;
}
/* v0.2.6: combined "en · tz · lat,lon" detail line — mobile-only via @media. */
.ws__detail {
  display: none;
}
.ws__temp-unit {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

/* v0.2.18: cells-only styles (.sortable-table 共性已抽到 SortableTable.vue 内).
   Search Engines 独有 cell: title + URL (PC) + ⭐/☆ Star + sort (PC) + actions. */
.engines-cell__title {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--mp-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1 1 auto;
  min-width: 0;
}
.engines-cell__url {
  font-size: 0.85rem;
  color: var(--mp-text-secondary);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1 1 auto;
  min-width: 0;
}
.engines-cell__star {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--mp-text-secondary);
  flex-shrink: 0;
  border-radius: 4px;
  transition: color 0.15s;
}
.engines-cell__star:hover:not(:disabled) {
  color: var(--mp-brand-primary);
}
.engines-cell__star--active {
  color: #f59e0b;
  cursor: default;
}
.engines-cell__sort {
  font-size: 0.85rem;
  color: var(--mp-text-secondary);
  width: 32px;
  text-align: center;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.engines-cell__actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

/* v0.2.6: cities row swaps from a single-line flex to a 2-row CSS Grid on
   phones. Drag handle + remove button span both rows (vertically centered),
   while name + detail stack between them. The 14px gap on desktop is kept
   intact — only the @media block below redefines layout for ≤768px. */
@media (max-width: 768px) {
  .ws__city {
    display: grid;
    grid-template-columns: 24px 1fr auto;
    grid-template-areas:
      "drag name remove"
      "drag detail remove";
    column-gap: 10px;
    row-gap: 2px;
    padding: 10px 12px;
  }
  .ws__drag {
    grid-area: drag;
    align-self: center;
  }
  .ws__cn {
    grid-area: name;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ws__en,
  .ws__tz,
  .ws__coords {
    display: none;
  }
  .ws__detail {
    grid-area: detail;
    display: block;
    min-width: 0;
    font-size: 0.72rem;
    opacity: 0.55;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-family: monospace;
  }
  .ws__remove {
    grid-area: remove;
    align-self: center;
  }

  /* v0.2.18: engines-cell mobile rules — URL 模板 + sort 数字 mobile 隐藏
     (跟 v0.2.14 决策一致, 进编辑表单看 URL). title font 缩小. */
  .engines-cell__url {
    display: none;
  }
  .engines-cell__sort {
    display: none;
  }
  .engines-cell__title {
    font-size: 0.85rem;
    line-height: 1.2;
  }
}
</style>
