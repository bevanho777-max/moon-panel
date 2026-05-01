<script setup lang="ts">
import { computed, h, onMounted, ref, type VNode } from 'vue'
import {
  NButton,
  NEmpty,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NSpace,
  NSpin,
  useMessage,
} from 'naive-ui'
import { useRouter } from 'vue-router'
import { ApiError } from '@/api/client'
import { getPanel, type PanelData } from '@/api/panel'
import NetworkSwitcher from '@/components/NetworkSwitcher.vue'
import CardItem from '@/components/CardItem.vue'
import HeaderSearchBox from '@/components/HeaderSearchBox.vue'
import HomeHero from '@/components/HomeHero.vue'

const router = useRouter()
const message = useMessage()

const panel = ref<PanelData | null>(null)
const loading = ref(true)
const searchQuery = ref('')

// Live filter applied to groups + cards. Empty query → return original list.
// When filtering, hide groups whose cards array becomes empty.
const filteredGroups = computed(() => {
  if (!panel.value) return []
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return panel.value.groups ?? []
  return (panel.value.groups ?? [])
    .map((group) => {
      const groupNameMatch = group.name.toLowerCase().includes(q)
      const cards = (group.cards ?? []).filter((card) =>
        groupNameMatch ||
        card.title.toLowerCase().includes(q) ||
        card.description.toLowerCase().includes(q) ||
        card.url_internal.toLowerCase().includes(q) ||
        card.url_external.toLowerCase().includes(q),
      )
      return { ...group, cards }
    })
    .filter((g) => (g.cards?.length ?? 0) > 0)
})

const showFilterEmpty = computed(() => {
  if (!panel.value) return false
  if (!searchQuery.value.trim()) return false
  return filteredGroups.value.length === 0
})

async function load() {
  loading.value = true
  try {
    panel.value = await getPanel()
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      router.replace({ name: 'login', query: { redirect: '/' } })
      return
    }
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

// Inline Lucide-derived folder icon — Phase 3 will replace with proper icon lib.
function folderIcon(): VNode {
  return h(
    'svg',
    {
      width: 18,
      height: 18,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
      'aria-hidden': 'true',
    },
    [
      h('path', {
        d: 'M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z',
      }),
    ],
  )
}

function moonIcon(): VNode {
  return h(
    'svg',
    {
      width: 22,
      height: 22,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
      'aria-hidden': 'true',
    },
    [h('path', { d: 'M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z' })],
  )
}

onMounted(load)
</script>

<template>
  <NLayout class="home-layout">
    <NLayoutHeader bordered class="home-header">
      <div class="home-header__brand">
        <span class="home-header__logo">
          <component :is="moonIcon()" />
        </span>
        <span class="home-header__title">Moon Panel</span>
      </div>
      <div class="home-header__spacer" />
      <NSpace align="center" :size="12">
        <HeaderSearchBox
          :engines="panel?.search_engines ?? []"
          @update:query="searchQuery = $event"
        />
        <NetworkSwitcher />
        <NButton size="small" @click="router.push('/admin')">管理后台</NButton>
      </NSpace>
    </NLayoutHeader>

    <NLayoutContent class="home-content">
      <NSpin :show="loading" style="min-height: 240px">
        <template v-if="panel">
          <HomeHero
            :cities="panel.site.cities ?? []"
            :temp-unit="panel.site.temp_unit ?? 'C'"
          />

          <div v-if="panel.groups.length === 0" class="home-empty">
            <NEmpty description="还没有任何卡片" size="huge">
              <template #extra>
                <NButton
                  type="primary"
                  style="max-width: 200px"
                  @click="router.push('/admin/groups')"
                >
                  去管理后台添加
                </NButton>
              </template>
            </NEmpty>
          </div>

          <div v-if="showFilterEmpty" class="home-empty">
            <NEmpty :description="`没有匹配「${searchQuery}」的卡片`" />
          </div>

          <TransitionGroup name="home-group-fade" tag="div">
            <section
              v-for="group in filteredGroups"
              :key="group.id"
              class="home-group"
            >
              <h2 class="home-group__title">
                <component :is="folderIcon()" class="home-group__icon" />
                <span>{{ group.name }}</span>
              </h2>
              <div
                v-if="!group.cards || group.cards.length === 0"
                class="home-group__placeholder"
              >
                该分组下还没有卡片
              </div>
              <TransitionGroup
                v-else
                name="home-card-fade"
                tag="div"
                class="home-group__cards"
              >
                <CardItem
                  v-for="card in group.cards"
                  :key="card.id"
                  :card="card"
                />
              </TransitionGroup>
            </section>
          </TransitionGroup>
        </template>
      </NSpin>
    </NLayoutContent>
  </NLayout>
</template>

<style scoped>
.home-layout {
  min-height: 100vh;
}

.home-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0 1.5rem;
  height: 56px;
}
.home-header__brand {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}
.home-header__logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #5b8def;
}
.home-header__title {
  font-weight: 600;
  font-size: 1.05rem;
  letter-spacing: 0.01em;
  white-space: nowrap;
}
.home-header__spacer {
  flex: 1;
}

.home-content {
  padding: 2rem 1.5rem 3rem;
  max-width: 1500px;
  margin: 0 auto;
}

.home-empty {
  min-height: 50vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.home-group {
  margin-bottom: 32px;
  padding: 24px;
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 16px;
  /* Frosted-glass for groups too — same degraded-on-dark, lit-up-on-bgimage
     behavior as cards. Saturate on bg image lifts color; on dark bg it's a no-op. */
  -webkit-backdrop-filter: blur(6px);
  backdrop-filter: blur(6px);
}
.home-group:last-child {
  margin-bottom: 0;
}
.home-group__title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0 0 1.1rem;
  padding-bottom: 0.7rem;
  font-size: 1.1rem;
  font-weight: 600;
  letter-spacing: 0.01em;
  color: rgba(255, 255, 255, 0.92);
  border-bottom: 1px solid rgba(255, 255, 255, 0.15);
}
.home-group__icon {
  color: #5b8def;
  flex-shrink: 0;
}
.home-group__placeholder {
  padding: 1.5rem 0;
  font-size: 0.85rem;
  opacity: 0.5;
}
.home-group__cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 0.75rem;
}

/* Filter transition: gentle fade with no layout jitter. Cards leaving fade
   to opacity 0 first, then collapse out. CSS Grid auto-flow re-packs the
   remaining cards smoothly. */
.home-card-fade-enter-active,
.home-card-fade-leave-active {
  transition: opacity 200ms ease, transform 200ms ease;
}
.home-card-fade-enter-from,
.home-card-fade-leave-to {
  opacity: 0;
  transform: scale(0.96);
}
.home-card-fade-leave-active {
  position: absolute;
}
.home-group-fade-enter-active,
.home-group-fade-leave-active {
  transition: opacity 250ms ease;
}
.home-group-fade-enter-from,
.home-group-fade-leave-to {
  opacity: 0;
}

@media (max-width: 768px) {
  .home-header {
    padding: 0 1rem;
    gap: 0.5rem;
  }
  .home-content {
    padding: 1.25rem 1rem 2rem;
  }
  .home-group {
    padding: 16px;
    margin-bottom: 20px;
    border-radius: 12px;
  }
  .home-group__cards {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 8px;
  }
}
@media (max-width: 480px) {
  .home-group {
    padding: 14px;
  }
  .home-group__cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
