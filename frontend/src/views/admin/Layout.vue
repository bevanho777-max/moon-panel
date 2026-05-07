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
import { Home, Menu as MenuIcon } from 'lucide-vue-next'
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

// v0.2.6: mobile-only action handlers. Each closes the mobile dropdown after
// firing so the menu does not stay open over the destination view.
function handleMobileChangePassword() {
  closeMobileMenu()
  showPasswordModal.value = true
}
function handleMobileLogout() {
  closeMobileMenu()
  handleLogout()
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
        <!-- v0.2.10: mobile-only 顶栏快捷入口 "查看主页", 搬出 dropdown.
             Desktop 端隐藏 (default style display: none; mobile @media flips
             to inline-flex). Replaces the v0.2.6 dropdown-internal entry. -->
        <button
          type="button"
          class="admin-nav-mobile-home"
          title="查看主页"
          aria-label="查看主页"
          @click="router.push('/')"
        >
          <Home :size="18" />
        </button>
        <div class="admin-header__actions-desktop">
          <NSpace>
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
        </div>
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
        <!-- v0.2.6: actions absorbed from the right-side desktop NButton/NDropdown
             which are display:none on mobile. Hand-rolled <button> rather than
             nesting NDropdown-inside-mobile-menu (NDropdown's portal teleports
             out of .admin-nav-mobile-menu and breaks the click-outside logic). -->
        <div class="admin-nav-mobile-divider" role="separator" aria-hidden="true" />
        <button
          type="button"
          class="admin-nav-mobile-item admin-nav-mobile-action"
          role="menuitem"
          @click="handleMobileChangePassword"
        >
          修改密码
        </button>
        <button
          type="button"
          class="admin-nav-mobile-item admin-nav-mobile-action"
          role="menuitem"
          @click="handleMobileLogout"
        >
          退出登录
        </button>
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
/* v0.2.10: Mobile-only "查看主页" icon button, sits right of hamburger.
   Desktop hides this (admin-header__actions-desktop has the labeled
   "查看主页" button instead). Box style mirrors .admin-nav-hamburger
   (44x44, --mp-card-border, --mp-card-bg-hover) so the two icons read
   as a paired left-side action group — visual gate patch from initial
   "lighter style" mismatch. */
.admin-nav-mobile-home {
  display: none;
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
  transition: background 0.15s;
}
.admin-nav-mobile-home:hover,
.admin-nav-mobile-home:active {
  background: var(--mp-card-bg-hover);
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
    /* v0.2.14 P0 a: mobile ☰+🏠 漂到右侧. PC 用 .admin-header__menu flex:1
       充当 spacer; mobile 隐藏 nav-desktop 后只剩 [title]+[NSpace] 两 child,
       space-between 把 NSpace 推到右边 (Bevan v0.2.13 真机反馈). */
    justify-content: space-between;
  }
  /* v0.2.5: scoped scoped-CSS @media takes precedence over the desktop
     `.admin-header__title` rule above (same specificity, later in source).
     The token --mp-brand-font-size-mobile is theme-aware: moon → 1.0rem,
     risen → 1.15rem, both defined in main.css's :root[data-theme] blocks. */
  .admin-header__title {
    font-size: var(--mp-brand-font-size-mobile);
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
  /* v0.2.10 mobile-home: surface as inline-flex so it sits next to the
     hamburger button. Box style (44x44 / border / hover) is in the
     default rule above, mirroring .admin-nav-hamburger. */
  .admin-nav-mobile-home {
    display: inline-flex;
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
  /* v0.2.6: desktop right-side NButton ("查看主页") + NDropdown ("admin▾") are
     wrapped in this div on mobile so a single rule hides both — their
     equivalents are absorbed into the hamburger menu below. The hamburger
     <button> sits as a sibling outside this wrapper and stays visible. */
  .admin-header__actions-desktop {
    display: none;
  }
  /* v0.2.6: action items in mobile dropdown (查看主页 / 修改密码 / 退出登录).
     Reuses .admin-nav-mobile-item base style; overrides only the <button>
     resets so they read pixel-equivalent to the RouterLink rows above. */
  .admin-nav-mobile-action {
    width: calc(100% - 16px);
    text-align: left;
    background: transparent;
    border: none;
    cursor: pointer;
    font: inherit;
  }
  .admin-nav-mobile-divider {
    height: 1px;
    margin: 6px 12px;
    background: var(--mp-card-border);
  }
  .admin-content {
    padding: 1.5rem 1rem;
    /* v0.2.6: bottom-pad is theme-aware. Moon hides StatusBar (24px breathing
       room only); risen shows StatusBar (~30-34px on mobile) so we add ~46px
       extra to keep the last NCard from being clipped/overlapped. Token
       defined in main.css's :root[data-theme] blocks. */
    padding-bottom: var(--mp-content-bottom-pad-mobile);
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
