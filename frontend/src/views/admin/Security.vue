<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NPopconfirm,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { listLockedIPs, unlockIP, type LockedIPItem } from '@/api/security'
import { ApiError } from '@/api/client'

const message = useMessage()

const items = ref<LockedIPItem[]>([])
const loading = ref(false)
let refreshTimer: number | null = null

async function refresh() {
  loading.value = true
  try {
    items.value = await listLockedIPs()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function handleUnlock(item: LockedIPItem) {
  try {
    await unlockIP(item.ip, item.source)
    message.success(`已解锁 ${item.ip}`)
    refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '解锁失败')
  }
}

function formatRemaining(seconds: number): string {
  if (seconds <= 0) return '已过期'
  if (seconds < 60) return `${seconds} 秒`
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return s === 0 ? `${m} 分钟` : `${m}m ${s}s`
}

function sourceLabel(src: 'login' | 'totp'): string {
  return src === 'login' ? '密码登录' : '2FA 验证'
}

const columns = computed<DataTableColumns<LockedIPItem>>(() => [
  {
    title: 'IP',
    key: 'ip',
    width: 180,
    render: (row) => h('code', { style: 'font-family:monospace; user-select:all' }, row.ip),
  },
  {
    title: '锁定来源',
    key: 'source',
    width: 130,
    render: (row) =>
      h(NTag, { size: 'small', type: row.source === 'login' ? 'warning' : 'info' }, () =>
        sourceLabel(row.source),
      ),
  },
  {
    title: '失败次数',
    key: 'failures',
    width: 100,
  },
  {
    title: '剩余时间',
    key: 'remaining_seconds',
    width: 130,
    render: (row) => formatRemaining(row.remaining_seconds),
  },
  {
    title: '解锁',
    key: 'actions',
    width: 110,
    render: (row) =>
      h(
        NPopconfirm,
        {
          onPositiveClick: () => handleUnlock(row),
        },
        {
          trigger: () => h(NButton, { size: 'small', type: 'primary', text: true }, () => '立即解锁'),
          default: () => `确认解锁 ${row.ip}（${sourceLabel(row.source)}）？`,
        },
      ),
  },
])

onMounted(() => {
  refresh()
  // Auto-refresh every 30s while the page is open — locked counters tick down,
  // and an admin watching the dashboard wants to see them update without
  // manual reload. clearInterval on unmount.
  refreshTimer = window.setInterval(refresh, 30_000)
})

onBeforeUnmount(() => {
  if (refreshTimer !== null) clearInterval(refreshTimer)
})
</script>

<template>
  <div class="sec">
    <NCard title="当前锁定的 IP">
      <template #header-extra>
        <NButton size="small" @click="refresh" :loading="loading">刷新</NButton>
      </template>

      <NAlert type="info" :show-icon="false" style="margin-bottom: 12px">
        密码登录或 2FA 验证连续失败的 IP 会进入锁定。家庭网络可在<b>站点设置 → 受信网络</b>添加 CIDR 永久跳过锁定。
        软锁回退：锁定期间输入正确密码会扣减剩余时间，连续多次正确即可恢复。
      </NAlert>

      <NDataTable
        v-if="items.length > 0"
        class="sec__table"
        :columns="columns"
        :data="items"
        :loading="loading"
        :bordered="false"
        size="small"
      />
      <NEmpty v-else description="当前没有被锁定的 IP" size="small" style="padding: 24px 0" />
    </NCard>
  </div>
</template>

<style scoped>
.sec {
  max-width: 1000px;
  margin: 0 auto;
}
</style>
