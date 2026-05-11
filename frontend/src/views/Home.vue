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
import { Settings } from 'lucide-vue-next'
import { ApiError } from '@/api/client'
import { getPanel, type PanelData } from '@/api/panel'
import NetworkSwitcher from '@/components/NetworkSwitcher.vue'
import CardItem from '@/components/CardItem.vue'
import HeaderSearchBox from '@/components/HeaderSearchBox.vue'
import HomeHero from '@/components/HomeHero.vue'
import VersionBadge from '@/components/VersionBadge.vue'
import StatusBar from '@/components/StatusBar.vue'
import { useUIStore } from '@/stores/ui'

const ui = useUIStore()

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
        <span class="home-header__title">{{ ui.siteTitle || 'Moon Panel' }}</span>
      </div>
      <div class="home-header__spacer" />
      <NSpace align="center" :size="12">
        <HeaderSearchBox
          :engines="panel?.search_engines ?? []"
          @update:query="searchQuery = $event"
        />
        <NetworkSwitcher />
        <NButton
          class="hh-icon-button"
          size="small"
          circle
          title="管理后台"
          aria-label="管理后台"
          @click="router.push('/admin')"
        >
          <template #icon>
            <Settings :size="18" />
          </template>
        </NButton>
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
    <VersionBadge />
    <StatusBar />
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
  /* v0.2.2: moon icon glyph colour follows the theme accent so risen's
     golden palette doesn't get a stray blue glyph in the corner. */
  color: var(--mp-brand-accent);
}
.home-header__title {
  /* v0.2.1+/v0.2.2: brand vars from main.css [data-theme]. The token
     values for moon literally equal what was previously hardcoded
     (1.05rem / 600 / 0.01em / rgba(255,255,255,0.96)); risen swaps
     to a serif + larger + golden palette. */
  font-family: var(--mp-brand-font);
  font-weight: var(--mp-brand-font-weight);
  font-size: var(--mp-brand-font-size);
  letter-spacing: var(--mp-brand-letter-spacing);
  color: var(--mp-brand-primary);
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

/* v0.2.21: group 容器扁平化 (Bevan daily UX 反馈 "AI流之类的标题不需要再外框
   和背景, 这样看起来感觉会更舒服和整洁、直观"). 删 bg/border/border-radius/
   padding, 仅保留 margin-bottom 作 group 间距. 跟 v0.2.13 HomeCard 扁平化
   + v0.2.20 Weather 删 acrylic 同理念 (去除多余视觉层级). .home-group__title
   (含 folder icon + group name + border-bottom divider) 保留作分组识别.
   --mp-group-* token 保留在 main.css (备用给未来组件). */
.home-group {
  margin-bottom: 32px;
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
  color: var(--mp-group-title);
  border-bottom: 1px solid var(--mp-group-divider);
}
.home-group__icon {
  color: var(--mp-group-icon);
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

/* v0.2.11: Mobile-only 44x44 box mirror admin/Layout.vue
   .admin-nav-mobile-home style. PC keeps NaiveUI default (small circle,
   ~28x28). Same border + theme tokens + hover for visual unity with
   admin's hamburger + view-home buttons. */

.hh-icon-button {
  /* PC default: don't override NaiveUI small circle (28x28) */
}

@media (max-width: 768px) {
  .hh-icon-button {
    width: 44px;
    height: 44px;
    border: 1px solid var(--mp-card-border);
    border-radius: 8px;
    background: transparent;
    transition: background 0.15s;
  }
  .hh-icon-button:hover {
    background: var(--mp-card-bg-hover);
  }
}

@media (max-width: 768px) {
  .home-header {
    padding: 0 1rem;
    gap: 0.5rem;
  }
  /* v0.2.11: Mobile brand title displayed at 12px (was display: none in
     v0.2.10). Combined with v0.2.11 P0 c (44x44 icon buttons) + P0 d
     (SearchBox 100px), total mobile top bar width ~346px < 375 viewport,
     no wrap. Reverses v0.2.10 Task 2.16 brand-hide. */
  .home-header__title {
    font-size: 12px;
  }
  /* v0.2.10: Force NSpace items to inline-flex with center alignment.
     NSpace v2 in useGap mode renders no .n-space-item wrapper (Space.mjs:128
     itemClass undefined; wrapper div omitted). Direct-child selector > * is
     the reliable target — same pattern as v0.2.7 AuditLog. Fixes rectangular
     NInput (HeaderSearchBox) misaligning with circle buttons (NetworkSwitcher
     + Settings) in cramped mobile header. */
  .home-header :deep(.n-space > *) {
    display: inline-flex;
    align-items: center;
  }
  .home-content {
    padding: 1.25rem 1rem 2rem;
  }
  /* v0.2.21: .home-group mobile padding/border-radius 删 (跟 PC 容器扁平化
     一致). 仅保留 margin-bottom 调整 (mobile 间距更紧凑). */
  .home-group {
    margin-bottom: 20px;
  }
  .home-group__cards {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 8px;
  }
}
@media (max-width: 480px) {
  /* v0.2.21: .home-group ≤480px padding 也删 (跟 ≤768 + PC 一致). */
  .home-group__cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
