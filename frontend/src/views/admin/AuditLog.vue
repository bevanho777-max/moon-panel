<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
  NInput,
  NModal,
  NPagination,
  NSelect,
  NSpace,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { listAuditLogs, type AuditLogEntry } from '@/api/audit'
import { ApiError } from '@/api/client'

const message = useMessage()

const items = ref<AuditLogEntry[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

const filterAction = ref<string | null>(null)
const filterActor = ref('')

// Curated action filter — covers both explicit named actions (login_*,
// password_change, init_admin, logout) and the generic middleware ones by
// HTTP verb (PUT/POST/DELETE prefix).
const actionFilterOptions = [
  { label: '全部', value: '' },
  { label: '登录成功', value: 'login_success' },
  { label: '登录失败', value: 'login_failure' },
  { label: '修改密码', value: 'password_change' },
  { label: '退出登录', value: 'logout' },
  { label: '初始化管理员', value: 'init_admin' },
  { label: '2FA 启用/禁用', value: 'totp_e' }, // matches totp_enable + totp_enable_failed + totp_disable*
  { label: '2FA 验证', value: 'totp_verify' }, // matches totp_verify_success + totp_verify_failure
  { label: '备份码使用', value: 'backup_code_used' },
  { label: '受信网络变更', value: 'trusted_ip' },
  { label: '手动解锁', value: 'manual_unlock' },
  { label: '拖拽排序', value: 'reorder' },
  { label: '备份导出/恢复', value: 'backup_' },
  { label: '增删改（POST）', value: 'POST' },
  { label: '增删改（PUT）', value: 'PUT' },
  { label: '删除（DELETE）', value: 'DELETE' },
]

const detailEntry = ref<AuditLogEntry | null>(null)
const showDetail = computed(() => detailEntry.value !== null)

// Translate generic "<METHOD> /api/admin/cards/123" to friendly Chinese label.
// Falls back to the raw action for entries we don't recognize.
function friendlyAction(action: string): string {
  const named: Record<string, string> = {
    login_success: '登录成功',
    login_failure: '登录失败',
    login_password_ok_awaiting_2fa: '密码通过，等待 2FA',
    login_soft_unlock_progress: '软锁回退（密码正确但仍锁定）',
    login_soft_unlock_cleared: '软锁回退清除完成',
    logout: '退出登录',
    password_change: '修改密码',
    init_admin: '初始化管理员',
    totp_enable: '启用 2FA',
    totp_enable_failed: '启用 2FA 失败',
    totp_disable: '禁用 2FA',
    totp_disable_failed: '禁用 2FA 失败',
    totp_verify_success: '2FA 验证成功',
    totp_verify_failure: '2FA 验证失败',
    backup_code_used: '使用备份码登录',
    trusted_ip_add: '添加受信网络',
    trusted_ip_remove: '移除受信网络',
    manual_unlock: '管理员手动解锁',
    backup_export_json: '导出备份（JSON）',
    backup_export_zip: '导出备份（ZIP）',
    backup_restore: '从备份恢复',
    'PUT /api/admin/cards/reorder': '批量调整卡片顺序',
    'PUT /api/admin/groups/reorder': '批量调整分组顺序',
  }
  if (named[action]) return named[action]

  // Generic pattern: METHOD /api/admin/<resource>[/<id>]
  const m = /^(GET|POST|PUT|PATCH|DELETE)\s+\/api\/admin\/([^/]+)(?:\/(.+))?$/.exec(action)
  if (m) {
    const [, method, resource, rest] = m
    const verbCn: Record<string, string> = {
      POST: '创建',
      PUT: '修改',
      PATCH: '修改',
      DELETE: '删除',
    }
    const resCn: Record<string, string> = {
      cards: '卡片',
      groups: '分组',
      'search-engines': '搜索引擎',
      settings: '设置',
      icons: '图标',
    }
    const verb = verbCn[method] || method
    const res = resCn[resource] || resource
    return rest ? `${verb}${res} #${rest}` : `${verb}${res}`
  }
  return action
}

function statusColor(status: number): 'success' | 'warning' | 'error' | 'default' {
  if (status >= 200 && status < 300) return 'success'
  if (status === 401 || status === 429) return 'warning'
  if (status >= 400) return 'error'
  return 'default'
}

const columns: DataTableColumns<AuditLogEntry> = [
  {
    title: '时间',
    key: 'timestamp',
    width: 170,
    render: (row) => new Date(row.timestamp).toLocaleString('zh-CN', { hour12: false }),
  },
  {
    title: '操作者',
    key: 'actor',
    width: 100,
    render: (row) => row.actor || h('span', { style: 'opacity:0.5' }, '—'),
  },
  {
    title: '动作',
    key: 'action',
    render: (row) =>
      h('div', null, [
        h('div', { style: 'font-weight: 500' }, friendlyAction(row.action)),
        h(
          'div',
          { style: 'font-size: 0.75rem; opacity: 0.55; font-family: monospace' },
          row.action,
        ),
      ]),
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row) => h(NTag, { type: statusColor(row.status), size: 'small' }, () => row.status),
  },
  {
    title: 'IP',
    key: 'ip',
    width: 130,
    render: (row) => row.ip || h('span', { style: 'opacity:0.5' }, '—'),
  },
  {
    title: 'UA',
    key: 'user_agent',
    width: 60,
    render: (row) =>
      row.user_agent
        ? h(
            'span',
            { style: 'cursor: help; opacity: 0.6', title: row.user_agent },
            'hover',
          )
        : h('span', { style: 'opacity:0.5' }, '—'),
  },
  {
    title: '详情',
    key: 'details',
    width: 70,
    render: (row) =>
      h(
        NButton,
        { size: 'tiny', text: true, onClick: () => (detailEntry.value = row) },
        () => '查看',
      ),
  },
]

async function load() {
  loading.value = true
  try {
    const r = await listAuditLogs({
      page: page.value,
      size: pageSize.value,
      action: filterAction.value || undefined,
      actor: filterActor.value.trim() || undefined,
    })
    items.value = r.items
    total.value = r.total
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  page.value = 1
  load()
}

function resetFilters() {
  filterAction.value = null
  filterActor.value = ''
  page.value = 1
  load()
}

function prettyDetails(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

onMounted(load)
</script>

<template>
  <div class="al">
    <NCard title="审计日志" class="al__card">
      <template #header-extra>
        <NSpace>
          <NSelect
            v-model:value="filterAction"
            :options="actionFilterOptions"
            placeholder="筛选动作"
            clearable
            style="width: 180px"
            @update:value="applyFilters"
          />
          <NInput
            v-model:value="filterActor"
            placeholder="操作者"
            clearable
            style="width: 140px"
            @keyup.enter="applyFilters"
            @clear="applyFilters"
          />
          <NButton size="small" @click="applyFilters">查询</NButton>
          <NButton size="small" tertiary @click="resetFilters">重置</NButton>
        </NSpace>
      </template>

      <NDataTable
        :columns="columns"
        :data="items"
        :loading="loading"
        :bordered="false"
        size="small"
        class="al__table"
      />

      <div class="al__pagination">
        <NPagination
          v-model:page="page"
          :page-size="pageSize"
          :item-count="total"
          show-quick-jumper
          @update:page="load"
        />
        <span class="al__total">共 {{ total }} 条</span>
      </div>
    </NCard>

    <NModal
      :show="showDetail"
      preset="card"
      title="审计详情"
      style="max-width: 720px"
      @update:show="(v: boolean) => { if (!v) detailEntry = null }"
    >
      <template v-if="detailEntry">
        <div class="al__detail-meta">
          <div><b>动作：</b>{{ friendlyAction(detailEntry.action) }}</div>
          <div><b>动作 key：</b><code>{{ detailEntry.action }}</code></div>
          <div><b>时间：</b>{{ new Date(detailEntry.timestamp).toLocaleString('zh-CN', { hour12: false }) }}</div>
          <div><b>操作者：</b>{{ detailEntry.actor || '—' }}</div>
          <div><b>IP：</b>{{ detailEntry.ip || '—' }}</div>
          <div><b>状态码：</b>{{ detailEntry.status }}</div>
          <div><b>User-Agent：</b><code class="al__ua">{{ detailEntry.user_agent || '—' }}</code></div>
        </div>
        <pre class="al__detail-json">{{ prettyDetails(detailEntry.details) }}</pre>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.al {
  max-width: 1200px;
  margin: 0 auto;
}
.al__card {
  --header-extra-min-width: auto;
}
.al__table {
  margin-top: 8px;
}
.al__pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
}
.al__total {
  font-size: 0.85rem;
  opacity: 0.6;
}
.al__detail-meta {
  font-size: 0.85rem;
  display: grid;
  gap: 6px;
  margin-bottom: 12px;
}
.al__detail-meta code {
  font-family: monospace;
  font-size: 0.78rem;
  background: rgba(255, 255, 255, 0.06);
  padding: 1px 6px;
  border-radius: 3px;
}
.al__ua {
  word-break: break-all;
  white-space: normal;
  display: inline-block;
  max-width: 100%;
}
.al__detail-json {
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 12px;
  font-size: 0.78rem;
  font-family: monospace;
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
