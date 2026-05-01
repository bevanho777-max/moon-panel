<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NInputNumber,
  NModal,
  NPopconfirm,
  NSpace,
  NSpin,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { ApiError } from '@/api/client'
import { createGroup, deleteGroup, listGroups, updateGroup, type Group } from '@/api/group'
import { useGroupsStore } from '@/stores/groups'
import GroupsSortModal from '@/components/admin/GroupsSortModal.vue'
import StatefulInput from '@/components/StatefulInput.vue'
import IconAutoComplete from '@/components/admin/IconAutoComplete.vue'
import { showStatefulInputHintOnce } from '@/utils/statefulInputHint'

const message = useMessage()
const groupsStore = useGroupsStore()

const groups = ref<Group[]>([])
const loading = ref(false)
const sortOpen = ref(false)
// Snapshot of editorForm at modal open time — fed into StatefulInput so it
// can compare current vs original and drive its 4-state UX.
const editorOriginal = ref({ name: '', icon: '' })

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const editorForm = ref({ name: '', icon: '', sort: null as number | null })
const submitting = ref(false)

const editorTitle = computed(() => (editorMode.value === 'create' ? '新建分组' : '编辑分组'))

async function refresh() {
  loading.value = true
  try {
    groups.value = await listGroups()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editorMode.value = 'create'
  editingId.value = null
  editorForm.value = { name: '', icon: '', sort: null }
  editorOriginal.value = { name: '', icon: '' }
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

function openEdit(g: Group) {
  editorMode.value = 'edit'
  editingId.value = g.id
  editorForm.value = { name: g.name, icon: g.icon, sort: g.sort }
  editorOriginal.value = { name: g.name, icon: g.icon }
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

async function submit() {
  const name = editorForm.value.name.trim()
  if (!name) {
    message.warning('分组名不能为空')
    return
  }
  submitting.value = true
  try {
    const payload = {
      name,
      icon: editorForm.value.icon,
      ...(editorForm.value.sort !== null ? { sort: editorForm.value.sort } : {}),
    }
    if (editorMode.value === 'create') {
      await createGroup(payload)
      message.success('已创建')
    } else if (editingId.value !== null) {
      await updateGroup(editingId.value, payload)
      message.success('已更新')
    }
    editorOpen.value = false
    await refresh()
    // Invalidate the shared groups store so Cards.vue dropdown shows the
    // new name/sort immediately (4b polish — fixes the "rename group, then
    // open card editor → still shows old name" stale-cache bug).
    await groupsStore.invalidate()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(g: Group) {
  try {
    await deleteGroup(g.id)
    message.success(`已删除"${g.name}"`)
    await refresh()
    await groupsStore.invalidate()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

const columns = computed<DataTableColumns<Group>>(() => [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name', minWidth: 160 },
  {
    title: '图标',
    key: 'icon',
    width: 200,
    render: (row) => row.icon || h('span', { style: { opacity: 0.4 } }, '—'),
  },
  { title: '排序', key: 'sort', width: 80 },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, { size: 'small', onClick: () => openEdit(row) }, () => '编辑'),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row),
            positiveText: '删除',
            negativeText: '取消',
          },
          {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, () => '删除'),
            default: () => `删除分组"${row.name}"？组内卡片会一并删除。`,
          },
        ),
      ]),
  },
])

onMounted(refresh)
</script>

<template>
  <NCard>
    <template #header>
      <NSpace align="center" justify="space-between" style="width: 100%">
        <span>分组管理</span>
        <NSpace>
          <NButton :disabled="groups.length < 2" @click="sortOpen = true">调整顺序</NButton>
          <NButton type="primary" @click="openCreate">新建分组</NButton>
        </NSpace>
      </NSpace>
    </template>

    <NSpin :show="loading">
      <NDataTable
        v-if="groups.length > 0"
        :columns="columns"
        :data="groups"
        :row-key="(row: Group) => row.id"
        :bordered="false"
      />
      <NEmpty v-else description="还没有分组，点右上角「新建分组」开始添加" />
    </NSpin>
  </NCard>

  <NModal
    v-model:show="editorOpen"
    preset="card"
    :title="editorTitle"
    style="max-width: 480px"
    :mask-closable="!submitting"
  >
    <NForm @submit.prevent="submit">
      <NFormItem label="名称" required>
        <StatefulInput
          v-model="editorForm.name"
          :original-value="editorOriginal.name"
          placeholder="例如：影音、工具、开发"
          :maxlength="128"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="图标">
        <IconAutoComplete
          v-model="editorForm.icon"
          :original-value="editorOriginal.icon"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="排序权重">
        <NInputNumber
          v-model:value="editorForm.sort"
          :step="10"
          placeholder="留空 = 自动追加到末尾"
          :disabled="submitting"
          style="width: 100%"
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

  <GroupsSortModal v-model:show="sortOpen" @saved="refresh" />
</template>
