<script setup lang="ts" generic="T extends { id: number | string }">
// v0.2.18 P0: SortableTable 抽象组件 (Rule of Three 时机, v0.2.15+v0.2.16 累积 3
// 用例 — admin Cards / Groups / Search Engines).
//
// 共性 80% 抽象到本组件: handle / drag UX / animation / disabled / group-name /
// group-headers / .sortable-table BEM. 差异 20% 由 #item slot 调用方完全自己渲染.
//
// 数据接口统一为 [{id, name, items}] (per-group nested 或 single list 包装为
// [{id:0, name:'', items}]). 调用方 onReorder 自己处理 reorderXxx API + refresh().
import draggable from 'vuedraggable'
import { GripVertical } from 'lucide-vue-next'

interface SortableGroup {
  id: number | string
  name: string
  items: T[]
}

withDefaults(
  defineProps<{
    groups: SortableGroup[]
    groupName: string
    animation?: number
    disabled?: boolean
    showGroupHeaders?: boolean
    /** Optional item visibility filter — used by Cards.vue for search.
     *  When provided, items where `itemFilter(item) === false` are hidden
     *  via `v-show` (kept in DOM so vuedraggable indices stay stable).
     *  Recommended pairing: pass `disabled: true` while filter is active
     *  to prevent dragging hidden items (Cards.vue: `:disabled="isSearching"`). */
    itemFilter?: (item: T) => boolean
  }>(),
  {
    animation: 160,
    disabled: false,
    showGroupHeaders: false,
    itemFilter: undefined,
  },
)

const emit = defineEmits<{
  (e: 'reorder'): void
}>()

defineSlots<{
  item(props: { item: T; group: SortableGroup }): unknown
}>()

function onEnd() {
  emit('reorder')
}
</script>

<template>
  <div class="sortable-table">
    <div
      v-for="group in groups"
      :key="group.id"
      class="sortable-table__group"
    >
      <div
        v-if="showGroupHeaders"
        class="sortable-table__group-header"
      >
        <span class="sortable-table__group-name">{{ group.name }}</span>
        <span class="sortable-table__group-count">({{ group.items.length }})</span>
      </div>

      <draggable
        :list="group.items"
        :group="groupName"
        :animation="animation"
        :disabled="disabled"
        item-key="id"
        handle=".sortable-table__handle"
        class="sortable-table__items"
        @end="onEnd"
      >
        <template #item="{ element }">
          <div
            v-show="!itemFilter || itemFilter(element as T)"
            class="sortable-table__item"
          >
            <button
              type="button"
              class="sortable-table__handle"
              :class="{ 'sortable-table__handle--disabled': disabled }"
              :title="disabled ? '清空搜索后可拖动' : '拖动调整顺序'"
              :disabled="disabled"
            >
              <GripVertical :size="16" />
            </button>
            <slot name="item" :item="(element as T)" :group="(group as SortableGroup)" />
          </div>
        </template>
      </draggable>
    </div>
  </div>
</template>

<style scoped>
/* v0.2.18 P0: SortableTable BEM (跨 Cards / Groups / Search Engines 共性).
   各栏目 cell 内部 (.{module}-cell__*) 由调用方 scoped CSS 自己定义. */
.sortable-table {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.sortable-table__group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sortable-table__group-header {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 0 4px;
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--mp-text-primary);
}
.sortable-table__group-count {
  font-size: 0.8rem;
  color: var(--mp-text-secondary);
}
.sortable-table__items {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sortable-table__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--mp-card-bg);
  border: 1px solid var(--mp-card-border);
}
.sortable-table__handle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: none;
  cursor: grab;
  color: var(--mp-text-secondary);
  flex-shrink: 0;
  border-radius: 4px;
  transition: color 0.15s;
}
.sortable-table__handle:hover:not(:disabled) {
  color: var(--mp-brand-primary);
}
.sortable-table__handle:active:not(:disabled) {
  cursor: grabbing;
}
.sortable-table__handle--disabled {
  cursor: not-allowed;
  opacity: 0.3;
}
</style>
