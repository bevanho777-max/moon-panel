<script setup lang="ts">
import { computed, h, ref } from 'vue'
import {
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NSpace,
  NButton,
  NDropdown,
  NMenu,
  useMessage,
  type DropdownOption,
  type MenuOption,
} from 'naive-ui'
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

const menuOptions: MenuOption[] = [
  {
    label: () => h(RouterLink, { to: { name: 'admin-dashboard' } }, () => '概览'),
    key: 'admin-dashboard',
  },
  {
    label: () => h(RouterLink, { to: { name: 'admin-groups' } }, () => '分组'),
    key: 'admin-groups',
  },
  {
    label: () => h(RouterLink, { to: { name: 'admin-cards' } }, () => '卡片'),
    key: 'admin-cards',
  },
  {
    label: () => h(RouterLink, { to: { name: 'admin-settings' } }, () => '站点设置'),
    key: 'admin-settings',
  },
  {
    label: () => h(RouterLink, { to: { name: 'admin-audit-logs' } }, () => '审计日志'),
    key: 'admin-audit-logs',
  },
  {
    label: () => h(RouterLink, { to: { name: 'admin-security' } }, () => '安全管理'),
    key: 'admin-security',
  },
]

const activeKey = computed(() => (route.name as string) || 'admin-dashboard')

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
      <NMenu
        mode="horizontal"
        :options="menuOptions"
        :value="activeKey"
        responsive
        class="admin-header__menu"
      />
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
    </NLayoutHeader>
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
  /* v0.2.1: brand vars from main.css [data-theme]. moon equals previous
     hardcoded weight 600 / inherited size; risen resolves to serif. */
  font-family: var(--mp-brand-font);
  font-weight: var(--mp-brand-font-weight);
  font-size: var(--mp-brand-font-size);
  letter-spacing: var(--mp-brand-letter-spacing);
  color: var(--mp-brand-color);
  white-space: nowrap;
}
.admin-header__menu {
  flex: 1;
}
.admin-content {
  padding: 2rem 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}
</style>
