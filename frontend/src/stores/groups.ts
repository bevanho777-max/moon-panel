import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/group'
import type { Group } from '@/api/group'

// Single source of truth for the groups list across pages.
// Cards.vue table reads group names by id; the editor's group dropdown reads
// the same data; mutations from Groups.vue should call refresh() afterwards.
export const useGroupsStore = defineStore('groups', () => {
  const items = ref<Group[]>([])
  const loaded = ref(false)
  const loading = ref(false)

  const byId = computed(() => {
    const m = new Map<number, Group>()
    for (const g of items.value) m.set(g.id, g)
    return m
  })

  function nameOf(id: number): string {
    return byId.value.get(id)?.name ?? `#${id}`
  }

  async function refresh() {
    loading.value = true
    try {
      items.value = await api.listGroups()
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  // Lazy load: only fetch the first time, unless caller forces.
  async function ensureLoaded() {
    if (!loaded.value && !loading.value) await refresh()
  }

  function reset() {
    items.value = []
    loaded.value = false
  }

  // Mark cache stale and re-fetch immediately. Used after Groups.vue saves a
  // change so dependent views (Cards editor dropdown) see the new state
  // without waiting for ensureLoaded's "first time only" gate.
  async function invalidate() {
    loaded.value = false
    await refresh()
  }

  return { items, loaded, loading, byId, nameOf, refresh, ensureLoaded, reset, invalidate }
})
