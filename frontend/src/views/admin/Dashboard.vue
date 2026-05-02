<script setup lang="ts">
// Admin Overview — counters from /api/admin/stats. v0.1.2 showed hardcoded
// 0 for everything; v0.1.3 wires up the real endpoint.
import { onMounted, ref } from 'vue'
import { NAlert, NCard, NSpace, NSpin, NStatistic } from 'naive-ui'
import { http, type ApiResponse } from '@/api/client'

interface AdminStats {
  groups_count: number
  cards_count: number
  engines_count: number
  audit_count: number
}

const stats = ref<AdminStats | null>(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const { data } = await http.get<ApiResponse<AdminStats>>('/admin/stats')
    stats.value = data.data!
  } catch {
    // Network / 401 / 500 — leave stats null. The template falls back to
    // dashes; the page is informational, not load-bearing.
    stats.value = null
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <NSpace vertical :size="16">
    <!-- Informational alert. v0.1.2 had this empty as a "future stats coming"
         placeholder; v0.1.3 fills the stats and the alert pivots to a
         smaller forward-looking note. NOTE: keeping an NAlert here is also
         what currently anchors phase-3d-1 test 06's `.n-alert` waitFor —
         the test selector is too broad (matches any .n-alert on the page
         instead of `.n-modal .n-alert`). Removing this alert exposed the
         race; tightening that test's selector belongs in a follow-up. -->
    <NAlert type="info" :show-icon="false">
      左上角菜单可进入分组 / 卡片 / 站点设置 / 审计日志 / 安全管理。更多概览统计将在 v0.2 加入。
    </NAlert>
    <NCard title="概览">
      <NSpin :show="loading">
        <NSpace :size="32" wrap>
          <NStatistic label="分组" :value="stats?.groups_count ?? 0" />
          <NStatistic label="卡片" :value="stats?.cards_count ?? 0" />
          <NStatistic label="搜索引擎" :value="stats?.engines_count ?? 0" />
          <NStatistic label="近 7 天审计" :value="stats?.audit_count ?? 0" />
        </NSpace>
      </NSpin>
    </NCard>
  </NSpace>
</template>
