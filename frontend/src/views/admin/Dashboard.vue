<script setup lang="ts">
import { NCard, NSpace, NStatistic, NAlert } from 'naive-ui'
</script>

<template>
  <NSpace vertical size="large">
    <NAlert type="info" title="管理后台">
      左上角菜单可进入分组、卡片、站点设置、审计日志、安全管理。本页未来会显示卡片/分组/审计统计概览。
    </NAlert>
    <NCard title="概览">
      <NSpace>
        <NStatistic label="分组" :value="0" />
        <NStatistic label="卡片" :value="0" />
        <NStatistic label="搜索引擎" :value="0" />
      </NSpace>
    </NCard>
  </NSpace>
</template>
