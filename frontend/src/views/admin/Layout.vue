<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NSpace,
  NButton,
  NDropdown,
  useMessage,
  type DropdownOption,
} from 'naive-ui'
import { Menu as MenuIcon } from 'lucide-vue-next'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import ChangePasswordModal from '@/components/admin/ChangePasswordModal.vue'
import VersionBadge from '@/components/VersionBadge.vue'
import StatusBar from '@/components/StatusBar.vue'

const auth = useAuthStore()
const ui = useUIStore()
const router = useRouter()
const route = useRoute()
const message = useMessage()

const showPasswordModal = ref(false)
const showMobileMenu = ref(false)
const mobileMenuRef = ref<HTMLElement | null>(null)
const hamburgerRef = ref<HTMLElement | null>(null)

interface NavItem {
  name: string
  label: string
}

const navItems: NavItem[] = [
  { name: 'admin-dashboard', label: '概览' },
  { name: 'admin-groups', label: '分组' },
  { name: 'admin-cards', label: '卡片' },
  { name: 'admin-settings', label: '站点设置' },
  { name: 'admin-audit-logs', label: '审计日志' },
  { name: 'admin-security', label: '安全管理' },
]

function isActive(itemName: string) {
  return (route.name as string) === itemName
}

function toggleMobileMenu() {
  showMobileMenu.value = !showMobileMenu.value
}
function closeMobileMenu() {
  showMobileMenu.value = false
}

watch(() => route.path, () => closeMobileMenu())

function handleClickOutside(event: MouseEvent) {
  if (!showMobileMenu.value) return
  const target = event.target as Node | null
  if (!target) return
  if (mobileMenuRef.value?.contains(target)) return
  if (hamburgerRef.value?.contains(target)) return
  closeMobileMenu()
}

// Crossing back to desktop while the mobile dropdown is open would leave a
// stale `showMobileMenu = true` waiting for the next mobile resize. Cheap to
// reset on the breakpoint change so it never surprises a rotating tablet.
const desktopMq =
  typeof window !== 'undefined' ? window.matchMedia('(min-width: 769px)') : null
function handleBreakpointChange(e: MediaQueryListEvent) {
  if (e.matches) closeMobileMenu()
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  desktopMq?.addEventListener('change', handleBreakpointChange)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  desktopMq?.removeEventListener('change', handleBreakpointChange)
})

const userMenuOptions: DropdownOption[] = [
  { label: '修改密码', key: 'change-password' },
  { type: 'divider', key: 'd1' },
  { label: '退出登录', key: 'logout' },
]

async function handleLogout() {
  await auth.logout()
  message.success('已退出')
  router.replace('/login')
}

function handleUserMenu(key: string) {
  if (key === 'change-password') {
    showPasswordModal.value = true
  } else if (key === 'logout') {
    handleLogout()
  }
}
</script>

<template>
  <NLayout>
    <NLayoutHeader bordered class="admin-header">
      <div class="admin-header__title">{{ (ui.siteTitle || 'Moon Panel') }} · 管理后台</div>
      <nav class="admin-header__menu admin-nav-desktop" aria-label="后台导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="admin-nav-item"
          :class="{ 'admin-nav-item--active': isActive(item.name) }"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
      <NSpace>
        <button
          ref="hamburgerRef"
          type="button"
          class="admin-nav-hamburger"
          aria-label="切换导航菜单"
          :aria-expanded="showMobileMenu"
          data-testid="admin-nav-hamburger"
          @click="toggleMobileMenu"
        >
          <MenuIcon :size="20" />
        </button>
        <NButton size="small" @click="router.push('/')">查看主页</NButton>
        <NDropdown
          trigger="click"
          :options="userMenuOptions"
          @select="handleUserMenu"
        >
          <NButton size="small" data-testid="admin-user-menu">
            {{ auth.username || 'admin' }} ▾
          </NButton>
        </NDropdown>
      </NSpace>
    </NLayoutHeader>
    <Transition name="mp-mobile-menu">
      <div
        v-if="showMobileMenu"
        ref="mobileMenuRef"
        class="admin-nav-mobile-menu"
        role="menu"
      >
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="admin-nav-mobile-item"
          :class="{ 'admin-nav-mobile-item--active': isActive(item.name) }"
          role="menuitem"
          @click="closeMobileMenu"
        >
          {{ item.label }}
        </RouterLink>
      </div>
    </Transition>
    <NLayoutContent class="admin-content">
      <router-view />
    </NLayoutContent>

    <ChangePasswordModal v-model:show="showPasswordModal" />
    <VersionBadge />
    <StatusBar />
  </NLayout>
</template>

<style scoped>
.admin-header {
  display: flex;
  align-items: center;
  padding: 0 1.5rem;
  height: 56px;
  gap: 1.5rem;
}
.admin-header__title {
  /* v0.2.1+/v0.2.2: brand vars from main.css [data-theme]. moon resolves
     to v0.2.0-equivalent values; risen swaps to serif + larger + golden. */
  font-family: var(--mp-brand-font);
  font-weight: var(--mp-brand-font-weight);
  font-size: var(--mp-brand-font-size);
  letter-spacing: var(--mp-brand-letter-spacing);
  color: var(--mp-brand-primary);
  white-space: nowrap;
}

/* Desktop nav. v0.2.4 replaces NMenu(horizontal+responsive) which collapsed
   the whole bar at narrow widths. .admin-header__menu retained as the e2e
   selector anchor (phase-3d-2.spec.ts). */
.admin-header__menu {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.25rem;
  min-width: 0;
}
.admin-nav-item {
  display: inline-flex;
  align-items: center;
  height: 40px;
  padding: 0 14px;
  border-radius: 6px;
  color: var(--mp-text-secondary);
  text-decoration: none;
  font-size: 14px;
  white-space: nowrap;
  transition: color 0.15s ease, background-color 0.15s ease;
}
.admin-nav-item:hover {
  color: var(--mp-text-primary);
  background: var(--mp-card-bg-hover);
}
.admin-nav-item--active {
  color: var(--mp-brand-primary);
  font-weight: 600;
}

/* Hamburger + mobile dropdown are hidden by default, surfaced only at
   <=768px. Keeps PC behaviour pixel-equivalent to v0.2.3. */
.admin-nav-hamburger {
  display: none;
}
.admin-nav-mobile-menu {
  display: none;
}

.admin-content {
  padding: 2rem 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

@media (max-width: 768px) {
  .admin-header {
    gap: 0.75rem;
    padding: 0 1rem;
  }
  .admin-header__menu.admin-nav-desktop {
    display: none;
  }
  .admin-nav-hamburger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    background: transparent;
    border: 1px solid var(--mp-card-border);
    border-radius: 8px;
    color: var(--mp-text-primary);
    cursor: pointer;
    padding: 0;
  }
  .admin-nav-hamburger:hover,
  .admin-nav-hamburger:active {
    background: var(--mp-card-bg-hover);
  }
  .admin-nav-mobile-menu {
    display: block;
    position: fixed;
    top: 56px;
    left: 0;
    right: 0;
    background: var(--mp-card-bg);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border-bottom: 1px solid var(--mp-card-border);
    padding: 8px 0;
    z-index: 100;
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.35);
  }
  .admin-nav-mobile-item {
    display: block;
    min-height: 44px;
    padding: 12px 20px;
    margin: 0 8px;
    color: var(--mp-text-primary);
    font-size: 16px;
    line-height: 20px;
    text-decoration: none;
    border-radius: 6px;
  }
  .admin-nav-mobile-item:hover,
  .admin-nav-mobile-item:active {
    background: var(--mp-card-bg-hover);
  }
  .admin-nav-mobile-item--active {
    color: var(--mp-brand-primary);
    font-weight: 600;
  }
  .admin-content {
    padding: 1.5rem 1rem;
  }
}

.mp-mobile-menu-enter-active,
.mp-mobile-menu-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.mp-mobile-menu-enter-from,
.mp-mobile-menu-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
