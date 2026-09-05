<template>
  <!-- 整页布局下占满全高（flex 拉伸）：不再使用圆角，作为通高侧栏与顶部导航栏/内容区左右并排。
       h-full + min-h-0 + overflow-y-auto：菜单超高时可纵向滚动（App 壳层 overflow-hidden）。 -->
  <aside
    class="tenant-console-sidebar bg-white shadow-md p-4 h-full min-h-0 overflow-y-auto"
    :class="collapsed ? 'tenant-console-sidebar-collapsed' : ''"
  >
    <!-- 标题即缩窄开关：点击在 200px 完整宽度 ↔ 64px 图标宽度之间切换（状态持久化于 localStorage） -->
    <!-- 标题文案与导航项文字列对齐：pl-11 = px-3(12) + icon w-5(20) + gap-3(12) -->
    <h3
      class="text-base font-semibold text-text mb-3 leading-6 cursor-pointer select-none rounded-lg transition-colors hover:bg-primary/5"
      :class="collapsed ? 'flex h-9 items-center justify-center' : 'flex items-center pl-11 pr-3'"
      :title="collapsed ? '展开侧栏' : '收起侧栏'"
      @click="toggleSidebarCollapse"
    >
      <svg v-if="collapsed" class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7"></path>
      </svg>
      <template v-else>
        <span>控制台导航</span>
        <svg class="w-4 h-4 ml-auto shrink-0 text-text-light transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7"></path>
        </svg>
      </template>
    </h3>
    <nav class="space-y-1" aria-label="控制台导航">
      <!-- 项目列表导航项 -->
      <router-link
        v-if="showMenu('nav.projects')"
        :to="`${tenantPath}/projects`"
        class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group"
        :class="[$route.path.includes('projects/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/10 text-text group-hover:text-primary', collapsed ? 'justify-center px-0' : '']"
        :title="collapsed ? '项目列表' : undefined"
      >
        <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path>
        </svg>
        <span v-show="!collapsed" class="text-sm leading-5" :class="$route.path.includes('projects/') ? 'text-primary font-medium' : 'text-text group-hover:text-primary transition-colors'">项目列表</span>
      </router-link>

      <!-- 工作面板导航项 -->
      <router-link
        v-if="showMenu('nav.work_panel')"
        :to="`${tenantPath}/work-panel`"
        class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group"
        :class="[$route.path.includes('work-panel/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/10 text-text group-hover:text-primary', collapsed ? 'justify-center px-0' : '']"
        :title="collapsed ? '工作面板' : undefined"
      >
        <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"></path>
        </svg>
        <span v-show="!collapsed" class="text-sm leading-5" :class="$route.path.includes('work-panel/') ? 'text-primary font-medium' : 'text-text group-hover:text-primary transition-colors'">工作面板</span>
      </router-link>

      <!-- 镜像市场导航项 -->
      <router-link
        v-if="showMenu('nav.image_market')"
        :to="`${tenantPath}/image-market`"
        class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group"
        :class="[$route.path.includes('image-market') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/10 text-text group-hover:text-primary', collapsed ? 'justify-center px-0' : '']"
        :title="collapsed ? '镜像市场' : undefined"
      >
        <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V7c0-2-1-3-3-3H7c-2 0-3 1-3 3zm0 0l4-4m0 0l4 4m-4-4v8"></path>
        </svg>
        <span v-show="!collapsed" class="text-sm leading-5" :class="$route.path.includes('image-market') ? 'text-primary font-medium' : 'text-text group-hover:text-primary transition-colors'">镜像市场</span>
      </router-link>

      <TenantConsoleFeedbackNav
        v-if="showMenu('nav.feedback')"
        :tenant-id="activeTenantId"
        :collapsed="collapsed"
        @expand-sidebar="expandSidebar"
      />

      <!-- 人员管理：公司租户 + 任一 people.* 权限 -->
      <div v-if="isCompanyTenant && showPeopleSection">
        <div
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group cursor-pointer"
          :class="[$route.path.includes('/people/') ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/10', collapsed ? 'justify-center px-0' : '']"
          :title="collapsed ? '人员管理' : undefined"
          @click="toggleMenu('peopleMenu')"
        >
          <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
          </svg>
          <span v-show="!collapsed" class="text-sm leading-5" :class="$route.path.includes('/people/') ? 'text-primary font-medium' : 'text-text group-hover:text-primary'">人员管理</span>
          <svg v-show="!collapsed" class="w-4 h-4 ml-auto shrink-0 transition-transform" :class="{ 'rotate-90': menuOpen.peopleMenu }" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
          </svg>
        </div>

        <!-- 人员管理子菜单 -->
        <div v-if="menuOpen.peopleMenu && !collapsed" class="ml-11 mt-1 space-y-1">
          <router-link
            v-if="showMenu('people.invite')"
            :to="`${tenantPath}/people/invite/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/people/invite/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            邀请人
          </router-link>
          <router-link
            v-if="showMenu('people.manage')"
            :to="`${tenantPath}/people/manage/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/people/manage/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            管理人员
          </router-link>
          <router-link
            v-if="showMenu('people.groups')"
            :to="`${tenantPath}/people/groups/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/people/groups/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            管理分组
          </router-link>
          <router-link
            v-if="showMenu('people.access')"
            :to="`${tenantPath}/people/access/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/people/access/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            访问管理
          </router-link>
          <router-link
            v-if="showMenu('people.roles')"
            :to="`${tenantPath}/people/roles/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/people/roles/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            角色管理
          </router-link>
        </div>
      </div>

      <!-- 租户设置导航项 -->
      <div v-if="showSettingsSection">
        <div
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group cursor-pointer"
          :class="[($route.path.includes('/settings/') || $route.path.includes('/deliverable-systems/')) && !$route.path.includes('/workspace/') ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/10', collapsed ? 'justify-center px-0' : '']"
          :title="collapsed ? '设置' : undefined"
          @click="toggleMenu('settingsMenu')"
        >
          <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
          </svg>
          <span v-show="!collapsed" class="text-sm leading-5" :class="($route.path.includes('/settings/') || $route.path.includes('/deliverable-systems/')) && !$route.path.includes('/workspace/') ? 'text-primary font-medium' : 'text-text group-hover:text-primary'">设置</span>
          <svg v-show="!collapsed" class="w-4 h-4 ml-auto shrink-0 transition-transform" :class="{ 'rotate-90': menuOpen.settingsMenu }" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
          </svg>
        </div>

        <!-- 设置子菜单 -->
        <div v-if="menuOpen.settingsMenu && !collapsed" class="ml-11 mt-1 space-y-1">
          <router-link
            v-if="showMenu('settings.company')"
            :to="`${tenantPath}/settings/company/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/settings/company/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            公司设置
          </router-link>
          <router-link
            v-if="showMenu('settings.cloud')"
            :to="`${tenantPath}/settings/cloud-platform/`" 
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors" 
            :class="$route.path.includes('/settings/cloud-platform/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            云平台绑定
          </router-link>
          <router-link
            v-if="showMenu('settings.gitlab')"
            :to="`${tenantPath}/settings/gitlab-connection/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/settings/gitlab-connection/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            GitLab
          </router-link>
          <router-link
            v-if="showMenu('settings.task_panel')"
            :to="`${tenantPath}/settings/task-panel/`" 
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors" 
            :class="$route.path.includes('/settings/task-panel/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            工作空间管理
          </router-link>
          <router-link
            v-if="showMenu('settings.feature_params')"
            :to="`${tenantPath}/settings/feature-params/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/settings/feature-params/') && !$route.path.includes('/workspace/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            智能体资源配置
          </router-link>
          <router-link
            v-if="showMenu('settings.deliverable')"
            :to="`${tenantPath}/deliverable-systems/`" 
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors" 
            :class="$route.path.includes('deliverable-systems/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            交付物体系设置
          </router-link>
          <router-link
            v-if="showMenu('settings.status')"
            :to="`${tenantPath}/settings/status/`" 
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors" 
            :class="$route.path.includes('/settings/status/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            进度体系设置
          </router-link>
        </div>
      </div>
      
      <!-- 资源与订单导航项 -->
      <div v-if="showBillingSection">
        <div
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group cursor-pointer"
          :class="[$route.path.includes('/billing/') ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/10', collapsed ? 'justify-center px-0' : '']"
          :title="collapsed ? '资源与订单' : undefined"
          @click="toggleMenu('billingMenu')"
        >
          <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
          <span v-show="!collapsed" class="text-sm leading-5" :class="$route.path.includes('/billing/') ? 'text-primary font-medium' : 'text-text group-hover:text-primary'">资源与订单</span>
          <svg v-show="!collapsed" class="w-4 h-4 ml-auto shrink-0 transition-transform" :class="{ 'rotate-90': menuOpen.billingMenu }" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
          </svg>
        </div>

        <!-- 计费管理子菜单 -->
        <div v-if="menuOpen.billingMenu && !collapsed" class="ml-11 mt-1 space-y-1">
          <router-link
            v-if="showMenu('billing.overview')"
            :to="`${tenantPath}/billing/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path === `${tenantPath}/billing/` ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            概览
          </router-link>
          <router-link
            v-if="showMenu('billing.orders')"
            :to="`${tenantPath}/billing/orders/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors"
            :class="$route.path.includes('/billing/orders/') && !$route.path.includes('/billing/orders/create/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            订单列表
          </router-link>
          <router-link
            v-if="showMenu('billing.transactions')"
            :to="`${tenantPath}/billing/transactions/`"
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors" 
            :class="$route.path.includes('/billing/transactions/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            交易流水
          </router-link>
          <router-link
            v-if="showMenu('billing.usage')"
            :to="`${tenantPath}/billing/usage/`" 
            class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors" 
            :class="$route.path.includes('/billing/usage/') ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light hover:text-primary'"
          >
            使用明细
          </router-link>
        </div>
      </div>
    </nav>
  </aside>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { getStoredUserId } from '../utils/sessionUserIdUtils'
import { useTenantConsoleSidebarCollapse } from '../composables/useTenantConsoleSidebarCollapse'
import { clearStaleLastActiveTenantId } from '../utils/staleTenantRecovery.js'
import { usePermissions } from '../composables/usePermissions.js'
import { canSeeMenuKey, canSeeSection } from '../domain/auth/tenantConsoleNav.js'
import TenantConsoleFeedbackNav from './TenantConsoleFeedbackNav.vue'

const route = useRoute()
const perms = usePermissions()

// 侧栏缩窄（仅图标宽度）状态：点击"控制台导航"标题切换，持久化于 localStorage
const { collapsed, toggleCollapsed, expandSidebar } = useTenantConsoleSidebarCollapse()

// localStorage key: 持久化最后活跃租户 ID（与 Navbar.logic.vue 共享），
// 确保在非租户页面或 API 未返回时侧栏链接仍指向正确租户。
const LAST_TENANT_STORAGE_KEY = 'lastActiveTenantId'

// 添加响应式变量存储租户ID和公司租户状态
// 公司租户 = 公司活跃成员 / 管理员 / 创建者（companies/current 返回三标志）
const currentTenant = ref('')
const isCompanyTenant = ref(false)
const permsReady = ref(false)

const activeTenantId = computed(() => {
  if (route.params.tenant) return String(route.params.tenant)
  if (currentTenant.value) return String(currentTenant.value)
  try {
    const stored = localStorage.getItem(LAST_TENANT_STORAGE_KEY)
    if (stored) return String(stored)
  } catch (_) {}
  return ''
})

function hasCode(code) {
  const tid = activeTenantId.value
  if (!tid) return false
  return perms.hasPerm(tid, code)
}

function showMenu(key) {
  // 权限未就绪或无租户上下文：主导航三项兜底可见（链接仍由 tenantPath 决定）
  if (!permsReady.value || !activeTenantId.value) {
    return key === 'nav.projects' || key === 'nav.work_panel' || key === 'nav.image_market'
  }
  return canSeeMenuKey(key, hasCode)
}

const showPeopleSection = computed(() => permsReady.value && canSeeSection('people', hasCode))
const showSettingsSection = computed(() => permsReady.value && canSeeSection('settings', hasCode))
const showBillingSection = computed(() => permsReady.value && canSeeSection('billing', hasCode))

// 菜单展开状态管理
const menuOpen = ref({
  peopleMenu: false,
  settingsMenu: false,
  billingMenu: false
})

// 计算租户路径
const tenantPath = computed(() => {
  // 优先从路由中获取租户ID，确保导航链接与当前页面使用相同的租户ID
  const routeTenant = route.params.tenant
  if (routeTenant) {
    return `/tenant/${routeTenant}`
  }
  // 如果路由中没有租户ID，使用从API获取的租户ID
  const tenant = currentTenant.value
  if (tenant) {
    return `/tenant/${tenant}`
  }
  // 最终回退：从 localStorage 读取上次活跃租户 ID
  try {
    const stored = localStorage.getItem(LAST_TENANT_STORAGE_KEY)
    if (stored) return `/tenant/${stored}`
  } catch (_) {}
  return ''
})

// 切换菜单展开状态；缩窄态点击带子菜单的入口一步展开侧栏 + 展开对应子菜单
//（OPT-20260807-055：原需二次点击才见子菜单）
const toggleMenu = (menuName) => {
  if (collapsed.value) {
    expandSidebar()
    menuOpen.value[menuName] = true
    return
  }
  menuOpen.value[menuName] = !menuOpen.value[menuName]
}

// 标题点击：切换缩窄态；缩窄时收起所有子菜单，避免展开后残留展开状态
const toggleSidebarCollapse = () => {
  toggleCollapsed()
  if (collapsed.value) {
    menuOpen.value.peopleMenu = false
    menuOpen.value.settingsMenu = false
    menuOpen.value.billingMenu = false
  }
}

// 根据当前路由自动展开对应菜单
const updateMenuState = () => {
  const path = route.path
  
  // 重置所有菜单状态
  menuOpen.value.peopleMenu = false
  menuOpen.value.settingsMenu = false
  menuOpen.value.billingMenu = false
  
  // 根据当前路径设置菜单状态
  if (path.includes('/people/')) {
    menuOpen.value.peopleMenu = true
  } else if (path.includes('/settings/') || path.includes('/deliverable-systems/')) {
    menuOpen.value.settingsMenu = true
  } else if (path.includes('/billing/')) {
    menuOpen.value.billingMenu = true
  }
}

// 监听路由变化，更新菜单状态
watch(
  () => route.path,
  () => {
    updateMenuState()
  }
)

// 获取租户ID
const initData = async () => {
  try {
    // OPT-20260807-004 语义：localStorage currentUserId 为主存储，cookie 为兼容回退
    // （HttpOnly 签名 userId cookie 对 JS 不可读，仅 getCookie 会误判为未登录而早退，
    // 导致公司租户身份判定永不执行 — 2026-08-09 生产复现修复）
    const userId = getStoredUserId()
    if (!userId) {
      console.log('未找到当前用户标识（localStorage currentUserId / userId cookie 均为空），跳过API请求')
      return
    }
    
    const tenantParam = route.params.tenant ? `?tenant_id=${encodeURIComponent(route.params.tenant)}` : ''
    const response = await apiFetch(`/api/accounts/users/me/${tenantParam}`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (response.ok) {
      const userData = await response.json()
      // 获取租户ID — 优先使用路由中的租户ID
      let tenantId = ''
      const routeTenant = route.params.tenant
      if (routeTenant && userData.companies && userData.companies.length > 0) {
        const matchedCompany = userData.companies.find(c => String(c.id) === String(routeTenant))
        if (matchedCompany) {
          tenantId = matchedCompany.id
        }
      }
      if (!tenantId) {
        if (userData.current_company && userData.current_company.id) {
          tenantId = userData.current_company.id
        } else if (userData.companies && userData.companies.length > 0) {
          tenantId = userData.companies[0].id
        }
      }
      if (tenantId) {
        currentTenant.value = tenantId
        // Persist last active tenant for cross-page fallback
        try { localStorage.setItem(LAST_TENANT_STORAGE_KEY, tenantId) } catch (_) {}
        // 查询当前用户在该公司的租户身份（活跃成员/管理员/创建者任一即公司租户）
        try {
          const companyResp = await apiFetch(`/api/tenant/${tenantId}/accounts/companies/current/`, {
            credentials: 'include',
            headers: { 'Accept': 'application/json' }
          })
          if (companyResp.ok) {
            const companyData = await companyResp.json()
            isCompanyTenant.value = !!(companyData.member_is_active || companyData.member_is_admin || companyData.member_is_creator)
          } else if (companyResp.status === 404) {
            // 公司已被清库删除：勿继续用陈旧 id 污染导航回退
            clearStaleLastActiveTenantId(tenantId)
            isCompanyTenant.value = false
          }
        } catch (e) {
          console.error('获取公司租户身份失败:', e)
        }
      } else {
        // /me/ 无可用公司：清除清库前残留的 lastActiveTenantId
        try { localStorage.removeItem(LAST_TENANT_STORAGE_KEY) } catch (_) {}
      }
    } else {
      console.warn('[Sidebar] 获取用户信息失败，侧边栏租户上下文可能不完整')
    }
  } catch (error) {
    console.warn('[Sidebar] 获取用户信息出错:', error)
  }
}

// 组件挂载时初始化数据和菜单状态
onMounted(async () => {
  updateMenuState()
  await Promise.all([
    initData(),
    perms.load(apiFetch).finally(() => { permsReady.value = true }),
  ])
})
</script>
