<template>
  <!-- 收起态：仅在当前页有租户「控制台导航」恢复入口时隐藏顶栏（系统管理等页始终显示） -->
  <nav v-if="!navbarCollapsed" data-alias="cmp-navbar-main" :class="{
    'bg-white shadow-md fixed w-full top-0 z-50 px-6': navbarType === 'system_admin',
    'bg-white/90 backdrop-blur-sm shadow-sm py-4 px-6 sticky top-0 z-50': navbarType !== 'system_admin'
  }">
    <div :class="{
      'max-w-7xl flex justify-between h-16': navbarType === 'system_admin',
      'max-w-7xl flex justify-between items-center': navbarType !== 'system_admin'
    }">
      <!-- 左侧部分：Logo、价格、代码仓库 -->
      <div class="flex items-center min-w-0 gap-4 sm:gap-6">
        <div class="flex-shrink-0 flex items-center">
          <a href="/" class="flex items-center gap-2 min-w-0" data-testid="navbar-brand">
            <img :src="brandFaviconUrl" alt="云端开发" class="w-8 h-8 rounded-lg shrink-0" />
            <span class="text-primary text-xl sm:text-2xl font-bold truncate">云端开发</span>
          </a>
        </div>
        <a
          :href="pricingHref"
          class="nav-link hover:text-primary transition-colors shrink-0"
          data-testid="nav-pricing"
        >
          价格
        </a>
        <NavbarGitServiceNav
          :is-user-authenticated="isUserAuthenticated"
          :git-resources="gitResources"
          :membership-tier="membershipTier"
          :current-page-href="currentPageHref"
          :pricing-href="pricingHref"
          :git-resources-status="gitResourcesStatus"
        />
      </div>

      <!-- 右侧部分：导航链接和用户操作 -->
      <div class="flex items-center gap-2 sm:gap-4 lg:gap-6 shrink-0">
        <!-- 已登录状态 -->
        <template v-if="isUserAuthenticated">
          <!-- 平台角色始终显示「系统管理」。管理员也可被邀请进公司，故租户导航与本入口并列，
               不再用 v-else 互斥掉公司下拉。判定兼容 isPlatformStaff / isSuperuser / platform_roles。 -->
          <template v-if="isPlatformStaffUser">
            <a href="/system-admin/" class="nav-link hover:text-primary transition-colors">系统管理</a>
          </template>

          <!-- 有租户上下文：工作面板或公司切换下拉（普通用户与已加入公司的平台角色共用） -->
          <template v-if="hasTenantContext">
            <!-- 单公司（或无公司但 localStorage 回退）：保持原「工作面板」链接 -->
            <a
              v-if="userCompanies.length <= 1"
              :href="workPanelPath"
              class="nav-link hover:text-primary transition-colors"
              data-testid="nav-work-panel"
            >工作面板</a>

            <!-- 多公司：工作面板位置渲染为公司切换下拉，用户在此下拉选择具体公司 -->
            <div v-else class="relative shrink-0" data-testid="nav-company-switcher">
              <button
                type="button"
                class="nav-link hover:text-primary transition-colors inline-flex items-center gap-1"
                data-testid="nav-work-panel"
                :aria-expanded="companyMenuOpen ? 'true' : 'false'"
                aria-haspopup="menu"
                @click="companyMenuOpen = !companyMenuOpen"
                @keydown.esc="companyMenuOpen = false"
              >
                <span class="max-w-[10rem] truncate">{{ currentCompanyName }}</span>
                <svg
                  class="w-3.5 h-3.5 shrink-0 transition-transform"
                  :class="{ 'rotate-180': companyMenuOpen }"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              <!-- 点击外部关闭：透明遮罩（z-40 低于菜单 z-50）。
                   独立 dropdown-click-outside-overlay 类自带 fixed+inset-0 铺满视口且无灰罩，不依赖全局模态样式 -->
              <div
                v-if="companyMenuOpen"
                class="dropdown-click-outside-overlay"
                data-testid="nav-company-menu-overlay"
                @click="companyMenuOpen = false"
              ></div>

              <div
                v-if="companyMenuOpen"
                class="absolute right-0 mt-2 w-56 rounded-lg bg-white shadow-lg ring-1 ring-black/5 border border-gray-100 py-1 z-50"
                role="menu"
                data-testid="nav-company-menu"
              >
                <!-- Anti-Replay-OK: navigation — 真实 a[href]，含当前公司也进工作面板 -->
                <a
                  v-for="company in userCompanies"
                  :key="company.id"
                  :href="companyWorkPanelHref(company.id)"
                  role="menuitem"
                  class="flex items-center justify-between w-full px-4 py-2 text-sm text-left hover:bg-gray-50 hover:text-primary no-underline"
                  :class="isCurrentCompany(company.id) ? 'text-primary font-medium' : 'text-gray-700'"
                  :data-current="isCurrentCompany(company.id) ? 'true' : 'false'"
                  @click="onCompanyItemClick(company.id)"
                >
                  <span class="truncate">{{ company.name }}</span>
                  <svg
                    v-if="isCurrentCompany(company.id)"
                    class="w-4 h-4 shrink-0 ml-2"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                </a>
              </div>
            </div>
          </template>

          <!-- 已认证、非平台角色、无租户：引导创建/加入公司（平台角色永不显示「开始使用」） -->
          <template v-else-if="!isPlatformStaffUser">
            <a
              :href="onboardingHref"
              class="nav-link hover:text-primary transition-colors"
              data-testid="nav-onboarding"
            >开始使用</a>
          </template>

          <!-- 公司切换入口已并入「工作面板」下拉（nav-company-switcher），不再渲染独立 select -->

          <!-- 账号下拉（已去掉多账号登录：无账号切换列表与「添加账号」入口） -->
          <AccountSwitcherDropdown
            :display-name="displayName"
            :avatar-src="avatarSrc"
            :profile-path="profilePath"
            @logout="$emit('logout')"
          />
          
          <!-- 退出按钮保留为快捷入口（与下拉「退出当前」一致） -->
          <button @click="$emit('logout')" class="btn-secondary">退出</button>
        </template>
        
        <!-- 未登录状态 -->
        <template v-else>
          <a href="/auth/login/" class="btn-primary px-3 py-2 sm:px-4 sm:py-2">登录</a>
          <a href="/auth/register/" class="btn-secondary hidden sm:inline-flex px-3 py-2 sm:px-4 sm:py-2">注册</a>
        </template>
      </div>
    </div>
  </nav>
</template>

<script setup>
/* @alias:cmp-navbar-main */
import { computed, ref } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import { initialsAvatarDataUri } from '../utils/initialsAvatarDataUri.js'
import AccountSwitcherDropdown from './AccountSwitcherDropdown.vue'
import NavbarGitServiceNav from './NavbarGitServiceNav.vue'
import { useTenantConsoleSidebarCollapse } from '../composables/useTenantConsoleSidebarCollapse'
import { shouldHideNavbarWhenCollapsed } from '../utils/navbarCollapsedVisibility.js'
import { buildCompanySwitchHref } from '../utils/companySwitchNavigation.js'

// src/public/img/icon128.png 站点根静态资源（源：app/static/img/icon128.png）：
// JS 字符串避免模板 transformAssetUrls 构建期解析
//（同 Login.vue 处理，OPT-20260824-012）。
const brandFaviconUrl = '/img/icon128.png'

/** 将 { path, query } 或字符串转为可放在 <a href> 的地址（禁止 router-link 点击拦截）。 */
function toHref(to) {
  if (typeof to === 'string') return to
  const path = String(to?.path || '/').trim() || '/'
  const query = to?.query && typeof to.query === 'object' ? to.query : null
  if (!query) return path
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(query)) {
    if (v == null || v === '') continue
    params.set(k, Array.isArray(v) ? String(v[0]) : String(v))
  }
  const s = params.toString()
  return s ? `${path}?${s}` : path
}

const props = defineProps({
  navbarType: {
    type: String,
    default: 'user'
  },
  user: {
    type: Object,
    default: () => ({
      isAuthenticated: false,
      isSuperuser: false,
      username: ''
    })
  },
  currentUser: {
    type: Object,
    default: () => ({
      isAuthenticated: false,
      isSuperuser: false,
      username: '',
      avatarUrl: null,
      userId: '',
    })
  },
  isUserAuthenticated: {
    type: Boolean,
    default: false
  },
  tenantPath: {
    type: String,
    default: ''
  },
  route: {
    type: Object,
    required: true
  },
  userCompanies: {
    type: Array,
    default: () => []
  },
  currentTenant: {
    type: String,
    default: ''
  },
  /** 当前租户会员等级（'vip1' | 'normal'），仅驱动 VIP1 角标，不再拦截跳转。
   *  空串 = 未知（未加载/拉取失败）→ 不显示角标。 */
  membershipTier: {
    type: String,
    default: ''
  },
  /** 当前租户已开通/获赠的 GitLab 区域（含 gitlab_web_url），驱动代码仓库入口。 */
  gitResources: {
    type: Array,
    default: () => []
  },
  /** ready | unknown | error — 空列表仅在 ready 时跳转价格页。 */
  gitResourcesStatus: {
    type: String,
    default: 'ready'
  }
})

// 定义组件事件
const emit = defineEmits(['logout', 'company-switched'])

// 顶部导航栏收起：与侧栏「控制台导航」共用同一份状态（模块级单例 + localStorage 持久化）。
// 仅当当前路由挂载租户控制台侧栏（有恢复入口）时才 v-if 隐藏；系统管理页路由
// 从不传 navbarType，默认 user，不能靠 system_admin 类型豁免。
const { collapsed } = useTenantConsoleSidebarCollapse()
const navbarCollapsed = computed(() => shouldHideNavbarWhenCollapsed({
  collapsed: collapsed.value,
  navbarType: props.navbarType,
  path: props.route?.path,
}))

const displayName = computed(() => {
  return props.currentUser?.username || '未设置昵称'
})

const tenantPathTrimmed = computed(() => String(props.tenantPath || '').trim())

const avatarSrc = computed(() => {
  const url = props.currentUser?.avatarUrl
  if (url) {
    return url
  }
  const seed = props.currentUser?.username || 'user'
  return initialsAvatarDataUri(seed)
})

const profilePath = computed(() => {
  const uid = String(props.currentUser?.userId || getCookie('userId') || '').trim()
  const accessCode = props.route?.query?.accessCode ? String(props.route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/',
    query: accessCode ? { accessCode } : {}
  }
})

const pricingHref = computed(() => {
  const accessCode = props.route?.query?.accessCode ? String(props.route.query.accessCode) : ''
  return toHref({
    path: '/pricing/',
    query: accessCode ? { accessCode } : {}
  })
})

/** 无仓库且列表未就绪/失败时「代码仓库」留在当前页（真实 <a href>）。 */
const currentPageHref = computed(() => {
  const full = String(props.route?.fullPath || props.route?.path || '/').trim()
  return full.startsWith('/') ? full : '/'
})

// ── 公司切换下拉（多公司场景，替代「工作面板」链接）──

/** 下拉菜单开合状态 */
const companyMenuOpen = ref(false)

/** 当前公司 id：优先路由租户（tenantPath），其次 currentTenant prop */
const currentCompanyId = computed(() => {
  const tp = tenantPathTrimmed.value
  const m = tp.match(/^\/tenant\/([^/]+)/)
  return m?.[1] || String(props.currentTenant || '').trim()
})

/** 当前公司名：下拉触发按钮文案；找不到匹配时回退列表首个公司，最终回退「工作面板」 */
const currentCompanyName = computed(() => {
  const id = currentCompanyId.value
  const found = props.userCompanies.find((c) => String(c.id) === id)
  return found?.name || props.userCompanies[0]?.name || '工作面板'
})

const isCurrentCompany = (companyId) => String(companyId) === currentCompanyId.value

const accessCodeQuery = computed(() =>
  props.route?.query?.accessCode ? String(props.route.query.accessCode) : '',
)

/**
 * 各公司工作面板真实 href。任务详情等页点击「当前公司」也必须能跳到工作面板，
 * 不得因 isCurrentCompany 做成 no-op。accessCode 优先从 fullPath 解析，否则用 query。
 */
const companyWorkPanelHref = (companyId) => {
  const full = String(props.route?.fullPath || '')
  const fromFull = buildCompanySwitchHref(companyId, full)
  if (fromFull && (!accessCodeQuery.value || fromFull.includes('accessCode='))) {
    return fromFull
  }
  const fromQuery = accessCodeQuery.value
    ? buildCompanySwitchHref(
      companyId,
      `http://local.invalid/?accessCode=${encodeURIComponent(accessCodeQuery.value)}`,
    )
    : null
  return fromQuery || fromFull || '#'
}

/**
 * 公司菜单项：真实 <a href> 负责跳转（禁止 preventDefault）。
 * 单击只关菜单并通知 logic 写入 lastActiveTenantId；Ctrl/中键由浏览器跟 href。
 * Anti-Replay-OK: navigation
 */
const onCompanyItemClick = (companyId) => {
  companyMenuOpen.value = false
  emit('company-switched', String(companyId))
}

/** 工作面板链接：须有租户前缀；与「价格」一致保留 accessCode query。
 *  使用真实 <a href>（禁止 router-link / preventDefault 点击拦截）。
 *  当 tenantPath 为空时（如非租户路由的用户 profile 页），直接回退到 currentTenant 构造路径，
 *  避免非租户页面上的「工作面板」链接错误指向首页 /。 */
const workPanelPath = computed(() => {
  const full = String(props.route?.fullPath || '')
  const accessCode = accessCodeQuery.value

  // 已解析出租户 id 时统一走 buildCompanySwitchHref（与多公司下拉同一套 href 构造），
  // 避免 accessCode / 无租户回退两条路径漂移。accessCode 优先从 fullPath 解析，
  // 否则用 route.query 合成一次，避免 query 有码但 fullPath 未含时丢失。
  const buildFromTenant = (tenantSlug) => {
    const viaHref = buildCompanySwitchHref(tenantSlug, full)
    if (viaHref && (!accessCode || viaHref.includes('accessCode='))) {
      return viaHref
    }
    return (
      buildCompanySwitchHref(
        tenantSlug,
        accessCode
          ? `http://local.invalid/?accessCode=${encodeURIComponent(accessCode)}`
          : '',
      ) || '/onboarding/'
    )
  }

  const tp = tenantPathTrimmed.value
  if (tp) {
    const m = tp.match(/^\/tenant\/([^/]+)/)
    if (m && m[1]) {
      return buildFromTenant(m[1])
    }
  }

  // 回退：非租户路由页面（如 /user/:id/profile/）上没有 tenantPath，
  // 直接使用 currentTenant prop 构造正确的工作面板路径。
  const fallbackTenant = String(props.currentTenant || '').trim()
  if (fallbackTenant) {
    return buildFromTenant(fallbackTenant)
  }

  // 最终回退：尝试 localStorage 中持久化的 lastActiveTenantId，
  // 确保在非租户页面（如 profile）上「工作面板」链接仍指向正确租户。
  try {
    const stored = localStorage.getItem('lastActiveTenantId')
    if (stored) return buildFromTenant(stored)
  } catch (_) {}

  // 安全网：与 resolveWorkPanelPathFromUser() 保持一致，
  // 无租户上下文时引导用户到 onboarding 而非空白首页。
  return '/onboarding/'
})

/** "开始使用"链接：引导无租户用户创建/加入公司。保留 accessCode query。真实 <a href>。 */
const onboardingHref = computed(() => {
  const accessCode = accessCodeQuery.value
  if (accessCode) {
    return toHref({ path: '/onboarding/', query: { accessCode } })
  }
  return '/onboarding/'
})

/**
 * 平台角色判定：决定导航栏是否显示「系统管理」。
 * 兼容三种数据来源（任一命中即视为平台角色）：
 *  1. isPlatformStaff — v63 Navbar.logic 从 /api/auth/user-roles/ 派生
 *  2. isSuperuser     — /me/ 遗留字段（auth_super_admin 表）
 *  3. platform_roles  — /me/ 原始载荷（["super_admin","employee"]）
 * 平台角色用户即使无公司也绝不显示「开始使用」引导菜单。
 */
const isPlatformStaffUser = computed(() => {
  const u = props.currentUser || {}
  if (u.isPlatformStaff) return true
  if (u.isSuperuser) return true
  const roles = Array.isArray(u.platform_roles) ? u.platform_roles : []
  return roles.some((r) => ['super_admin', 'employee'].includes(r))
})

/**
 * 租户上下文判断：用于决定导航栏显示「工作面板」还是「开始使用」。
 * 综合 API 返回的 companies 列表和 localStorage 回退，
 * 避免 transient API 错误导致错误隐藏「工作面板」链接。
 */
const hasTenantContext = computed(() => {
  // API 已返回公司列表
  if (props.userCompanies.length > 0) return true
  // localStorage 回退：用户之前活跃过某个租户
  try {
    if (localStorage.getItem('lastActiveTenantId')) return true
  } catch (_) {}
  return false
})
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>
