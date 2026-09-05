<template>
  <!-- h-full + min-h-0 + overflow-y-auto：菜单超高时可纵向滚动（App 壳层 overflow-hidden，与租户 Sidebar 一致） -->
  <aside class="w-64 bg-gray-50 border-r border-gray-200 h-full min-h-0 flex-shrink-0 overflow-y-auto">
    <nav class="p-4">
      <ul class="space-y-2">
        <!-- 系统总览导航组 -->
        <li>
          <button
            class="w-full flex items-center justify-between px-4 py-2 text-sm font-semibold text-gray-600 hover:text-gray-900 transition-colors cursor-pointer"
            @click="toggleGroup('overview')"
          >
            <span>系统总览</span>
            <svg
              class="w-4 h-4 transition-transform duration-200"
              :class="isExpanded('overview') ? 'rotate-90' : 'rotate-0'"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <ul
            v-show="isExpanded('overview')"
            class="mt-0.5 space-y-1 border-l-2 border-gray-200 ml-4 pl-2"
          >
            <li>
              <router-link
                to="/system-admin/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                概览
              </router-link>
            </li>
            <li>
              <a
                href="/system-admin/users/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="isUsersNavActive ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                用户管理
              </a>
            </li>
            <li>
              <router-link
                to="/system-admin/oidc-extension/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/oidc-extension/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                浏览器插件
              </router-link>
            </li>
          </ul>
        </li>

        <!-- 系统配置导航组 -->
        <li>
          <button
            class="w-full flex items-center justify-between px-4 py-2 text-sm font-semibold text-gray-600 hover:text-gray-900 transition-colors cursor-pointer"
            @click="toggleGroup('system')"
          >
            <span>系统配置</span>
            <svg
              class="w-4 h-4 transition-transform duration-200"
              :class="isExpanded('system') ? 'rotate-90' : 'rotate-0'"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <ul
            v-show="isExpanded('system')"
            class="mt-0.5 space-y-1 border-l-2 border-gray-200 ml-4 pl-2"
          >
            <li>
              <router-link
                to="/system-admin/deliverable-system"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/deliverable-system/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                交付物体系管理
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/default-column-management"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/default-column-management/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                进度体系设置
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/gitlab-resources"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/gitlab-resources/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                GitLab资源管理
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/step-full-cos"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/step-full-cos/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                执行日志 COS
              </router-link>
            </li>
          </ul>
        </li>

        <!-- 运营管理导航组 -->
        <li>
          <button
            class="w-full flex items-center justify-between px-4 py-2 text-sm font-semibold text-gray-600 hover:text-gray-900 transition-colors cursor-pointer"
            @click="toggleGroup('ops')"
          >
            <span>运营管理</span>
            <svg
              class="w-4 h-4 transition-transform duration-200"
              :class="isExpanded('ops') ? 'rotate-90' : 'rotate-0'"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <ul
            v-show="isExpanded('ops')"
            class="mt-0.5 space-y-1 border-l-2 border-gray-200 ml-4 pl-2"
          >
            <li>
              <router-link
                to="/system-admin/grant-points"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/grant-points/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                赠送资源
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/order-records"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/order-records/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                订单与退款
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/referral-management"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="isReferralManagementActive ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                推荐码管理
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/price-management"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="isPricePackagesActive ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                价格管理
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/price-management/?tab=consumption"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="isConsumptionActive ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                消费情况
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/feedback-links/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/feedback-links/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                意见与建议链接
              </router-link>
            </li>
          </ul>
        </li>

        <!-- AI供应商导航组 -->
        <li>
          <button
            class="w-full flex items-center justify-between px-4 py-2 text-sm font-semibold text-gray-600 hover:text-gray-900 transition-colors cursor-pointer"
            @click="toggleGroup('provider')"
          >
            <span>AI供应商</span>
            <svg
              class="w-4 h-4 transition-transform duration-200"
              :class="isExpanded('provider') ? 'rotate-90' : 'rotate-0'"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <ul
            v-show="isExpanded('provider')"
            class="mt-0.5 space-y-1 border-l-2 border-gray-200 ml-4 pl-2"
          >
            <li>
              <router-link
                to="/system-admin/recommended-llm-providers"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/recommended-llm-providers/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                推荐供应商列表
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/sub-token-providers"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/sub-token-providers/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                派生Token供应商
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/container-images"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/container-images/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                容器镜像列表
              </router-link>
            </li>
          </ul>
        </li>

        <!-- 协议管理导航组 -->
        <li>
          <button
            class="w-full flex items-center justify-between px-4 py-2 text-sm font-semibold text-gray-600 hover:text-gray-900 transition-colors cursor-pointer"
            @click="toggleGroup('agreement')"
          >
            <span>协议管理</span>
            <svg
              class="w-4 h-4 transition-transform duration-200"
              :class="isExpanded('agreement') ? 'rotate-90' : 'rotate-0'"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <ul
            v-show="isExpanded('agreement')"
            class="mt-0.5 space-y-1 border-l-2 border-gray-200 ml-4 pl-2"
          >
            <li>
              <router-link
                to="/system-admin/privacy-policy/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/privacy-policy/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                隐私条款管理
              </router-link>
            </li>
            <li>
              <router-link
                to="/system-admin/license-agreement/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/license-agreement/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                服务协议
              </router-link>
            </li>
          </ul>
        </li>

        <!-- 安全策略导航组 -->
        <li>
          <button
            class="w-full flex items-center justify-between px-4 py-2 text-sm font-semibold text-gray-600 hover:text-gray-900 transition-colors cursor-pointer"
            @click="toggleGroup('security')"
          >
            <span>安全策略</span>
            <svg
              class="w-4 h-4 transition-transform duration-200"
              :class="isExpanded('security') ? 'rotate-90' : 'rotate-0'"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <ul
            v-show="isExpanded('security')"
            class="mt-0.5 space-y-1 border-l-2 border-gray-200 ml-4 pl-2"
          >
            <li>
              <router-link
                to="/system-admin/login-payment-policy/"
                class="block px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
                :class="$route.path === '/system-admin/login-payment-policy/' ? 'text-primary bg-primary/10' : 'text-gray-700 hover:bg-gray-100'"
              >
                登录与支付策略
              </router-link>
            </li>
          </ul>
        </li>
      </ul>
    </nav>
  </aside>
</template>

<script setup>
import { computed, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

// Route path → group key mapping for auto-expand
const ROUTE_GROUP_MAP = {
  '/system-admin/': 'overview',
  '/system-admin/users/': 'overview',
  '/system-admin/oidc-extension/': 'overview',
  '/system-admin/deliverable-system/': 'system',
  '/system-admin/default-column-management/': 'system',
  '/system-admin/gitlab-resources/': 'system',
  '/system-admin/gitlab-resources': 'system',
  '/system-admin/step-full-cos/': 'system',
  '/system-admin/step-full-cos': 'system',
  '/system-admin/grant-points/': 'ops',
  '/system-admin/order-records/': 'ops',
  '/system-admin/refund-applications/': 'ops',
  '/system-admin/referral-management/': 'ops',
  '/system-admin/price-management/': 'ops',
  '/system-admin/price-management': 'ops',
  '/system-admin/feedback-links/': 'ops',
  '/system-admin/feedback-links': 'ops',
  '/system-admin/container-images/': 'provider',
  '/system-admin/container-images': 'provider',
  '/system-admin/recommended-llm-providers/': 'provider',
  '/system-admin/sub-token-providers/': 'provider',
  '/system-admin/privacy-policy/': 'agreement',
  '/system-admin/license-agreement/': 'agreement',
  '/system-admin/login-payment-policy/': 'security',
}

// All groups start expanded
const expanded = reactive({
  overview: true,
  system: true,
  ops: true,
  provider: true,
  agreement: true,
  security: true,
})

function isExpanded(key) {
  return expanded[key]
}

function toggleGroup(key) {
  expanded[key] = !expanded[key]
}

/**
 * Auto-expand the group that contains the current route.
 * Users can still manually collapse it after navigation.
 */
function autoExpandCurrentGroup() {
  let groupKey = ROUTE_GROUP_MAP[route.path]
  if (!groupKey && String(route.path || '').startsWith('/system-admin/tenants/')) {
    groupKey = 'overview'
  }
  if (groupKey) {
    expanded[groupKey] = true
  }
}

// Run on mount and on route change
autoExpandCurrentGroup()
watch(() => route.path, autoExpandCurrentGroup)

const isPriceManagementPath = computed(() => {
  const path = route.path.replace(/\/$/, '')
  return path === '/system-admin/price-management'
})

const isReferralManagementActive = computed(() => {
  const path = (route.path || '').replace(/\/$/, '')
  return path === '/system-admin/referral-management'
})

const isConsumptionActive = computed(() =>
  isPriceManagementPath.value && route.query.tab === 'consumption'
)

const isPricePackagesActive = computed(() =>
  isPriceManagementPath.value && route.query.tab !== 'consumption'
)

const isUsersNavActive = computed(() => {
  const path = String(route.path || '')
  return path === '/system-admin/users/' || path.startsWith('/system-admin/tenants/')
})
</script>
