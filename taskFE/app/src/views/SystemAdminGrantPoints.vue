<template>
  <transition name="fade-slide">
    <div
      v-if="successToastVisible"
      class="fixed top-6 right-6 z-50 max-w-md rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800 shadow-lg"
      role="status"
      aria-live="polite"
    >
      {{ successToastMessage }}
    </div>
  </transition>
  <div data-alias="view-system-admin-grant-points" class="p-6 max-w-2xl">
    <h2 class="text-xl font-semibold text-gray-900 mb-1">赠送资源</h2>
    <p class="text-sm text-gray-500 mb-6">
      向指定租户（公司）赠送资源配额，支持选择资源类型、数量、过期时间和赠送原因，可同时赠送多种资源。
      数量默认留空：只改 VIP 时请勿填写数量，否则会同时生成赠送订单。
    </p>

    <div class="bg-amber-50 border border-amber-200 text-amber-900 text-sm rounded-lg p-4 mb-6">
      请从列表中选择租户并核对公司名与 ID。误操作将影响该租户下所有使用资源计费的功能。
    </div>

    <form class="space-y-4 bg-white border border-gray-200 rounded-lg p-6" @submit.prevent="submit">
      <!-- 租户选择 -->
      <div class="relative">
        <label class="block text-sm font-medium text-gray-700 mb-1">租户（公司）</label>
        <input
          v-model="tenantSearch"
          type="text"
          autocomplete="off"
          class="w-full border border-gray-300 rounded-md px-3 py-2"
          placeholder="输入公司名称筛选，或点击下项选择"
          @focus="openDropdown"
          @blur="scheduleCloseDropdown"
          @input="onSearchInput"
        >
        <ul
          v-if="dropdownOpen && tenantOptions.length"
          class="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-md border border-gray-200 bg-white shadow-lg"
        >
          <li
            v-for="t in tenantOptions"
            :key="t.id"
            class="cursor-pointer px-3 py-2 text-sm hover:bg-primary/10"
            @mousedown.prevent="selectTenant(t)"
          >
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900 truncate">{{ t.name }}</span>
              <span class="text-gray-500 text-xs whitespace-nowrap">ID {{ t.id }}</span>
            </div>
            <div
              v-if="tenantContactLine(t)"
              data-testid="tenant-option-contact"
              class="mt-0.5 text-xs text-gray-500"
            >
              {{ tenantContactLine(t) }}
            </div>
          </li>
        </ul>
        <p v-if="selectedTenant" data-testid="tenant-selected-summary" class="mt-2 text-sm text-gray-600">
          已选：<span class="font-medium text-gray-900">{{ selectedTenant.name }}</span>
          <span class="text-gray-500">（{{ selectedTenant.id }}）</span>
          <span v-if="tenantContactLine(selectedTenant)" class="text-gray-500"> · {{ tenantContactLine(selectedTenant) }}</span>
        </p>
        <p v-else class="mt-1 text-xs text-gray-500">未选择租户时无法提交</p>
      </div>

      <GrantPointsResourceLines
        :resource-lines="resourceLines"
        :regions="activeGitlabRegions"
        :allow-remove-last="Boolean(membershipTier)"
        @add="addResourceLine"
        @remove="removeResourceLine"
      />

      <GrantPointsMembershipSection
        v-model="membershipTier"
        :current-tier="currentMembershipTier"
        :locked="currentMembershipLocked"
      />

      <p
        v-if="selectedTenant && canSubmit && submitPreview"
        data-testid="grant-submit-preview"
        class="text-sm text-gray-700 bg-gray-50 border border-gray-200 rounded-md px-3 py-2"
      >
        {{ submitPreview }}
      </p>
      <div class="flex gap-3 pt-2">
        <button
          type="submit"
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
          :disabled="submitting || !selectedTenant || !canSubmit"
          :aria-busy="submitting ? 'true' : 'false'"
        >
          {{ submitting ? '提交中…' : '确认赠送' }}
        </button>
      </div>
    </form>

    <div v-if="errorMessage" class="mt-4 p-4 bg-red-50 text-red-800 rounded-lg text-sm" :data-traceId="errorMessageTraceId || undefined">
      {{ errorMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { tenantContactLine } from '../utils/tenantOptionContact.js'
import { createClickGuard } from '../utils/clickGuard.js'
import GrantPointsMembershipSection from './GrantPointsMembershipSection.vue'
import GrantPointsResourceLines from './GrantPointsResourceLines.vue'

const grantGuard = createClickGuard()
const route = useRoute()

const tenantSearch = ref('')
const tenantOptions = ref([])
const selectedTenant = ref(null)
const dropdownOpen = ref(false)
const fetchTimer = ref(null)
let blurCloseTimer = null

const submitting = ref(false)
const errorMessage = ref('')
const errorMessageTraceId = ref('')
const successToastVisible = ref(false)
const successToastMessage = ref('')
let successToastTimer = null

const gitlabRegions = ref([])
const membershipTier = ref('')
const currentMembershipTier = ref('')
const currentMembershipLocked = ref(false)

const resourceTypes = [
  { value: 'task_post', label: '任务帖' },
  { value: 'gitlab_disk', label: 'GitLab 磁盘 (GB)' },
  { value: 'gitlab_traffic', label: 'GitLab 流量 (GB)' },
]

function needsGitlabRegion(rt) {
  return rt === 'gitlab_disk' || rt === 'gitlab_traffic'
}

const activeGitlabRegions = computed(() =>
  gitlabRegions.value.filter((r) => r && r.slug && r.is_active !== false && r.is_active !== 0),
)

let nextLineId = 0

/** 返回半年后日期，格式 YYYY-MM-DD，用于 input[type=date] 默认值 */
function defaultExpiresAt() {
  const d = new Date()
  d.setMonth(d.getMonth() + 6)
  return d.toISOString().slice(0, 10)
}

function freshLine() {
  return {
    _id: nextLineId++,
    resource_type: 'task_post',
    quantity: '',
    expires_at: defaultExpiresAt(),
    reason: '',
    region: '',
  }
}

const resourceLines = ref([freshLine()])

const hasValidResources = computed(() =>
  resourceLines.value.some(
    (r) =>
      Number.isFinite(Number(r.quantity)) &&
      Number(r.quantity) >= 1 &&
      (!needsGitlabRegion(r.resource_type) || String(r.region || '').trim()),
  ),
)

const canSubmit = computed(() => hasValidResources.value || Boolean(String(membershipTier.value || '').trim()))

/** 提交核对文案：仅改 VIP 时明示不会生成赠送订单 */
const submitPreview = computed(() => {
  if (!selectedTenant.value) return ''
  const validLines = resourceLines.value.filter(
    (r) => Number.isFinite(Number(r.quantity)) && Number(r.quantity) >= 1,
  )
  const tier = String(membershipTier.value || '').trim()
  if (!validLines.length && !tier) return ''
  const parts = validLines.map((r) => {
    const rt = resourceTypes.find((x) => x.value === r.resource_type)
    const label = rt?.label || r.resource_type
    const region = r.region ? `（${r.region}）` : ''
    return `${label}${region} × ${Math.floor(Number(r.quantity))}`
  })
  const memText = tier ? `会员等级 → ${tier === 'vip1' ? 'VIP1' : tier}` : ''
  if (!validLines.length) {
    return `仅修改${memText}，不会生成赠送订单。`
  }
  return `将生成赠送订单：${parts.join('、')}${memText ? `；${memText}` : ''}。`
})

function addResourceLine() {
  resourceLines.value.push(freshLine())
}

function removeResourceLine(idx) {
  if (resourceLines.value.length <= 1 && !membershipTier.value) return
  resourceLines.value.splice(idx, 1)
}

const showSuccessToast = (message) => {
  successToastMessage.value = message
  successToastVisible.value = true
  if (successToastTimer) {
    clearTimeout(successToastTimer)
  }
  successToastTimer = setTimeout(() => {
    successToastVisible.value = false
    successToastTimer = null
  }, 4000)
}

const fetchOptions = async (search) => {
  errorMessageTraceId.value = ''
  try {
    const q = new URLSearchParams()
    if (search && search.trim()) {
      q.set('search', search.trim())
    }
    const url = `/api/system-admin/accounts/admin/tenant-options/?${q.toString()}`
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => [])
    if (!response.ok) {
      errorMessage.value = data.detail || `加载租户列表失败（${response.status}）`
      errorMessageTraceId.value = extractTraceId(response) || ''
      tenantOptions.value = []
      return
    }
    tenantOptions.value = Array.isArray(data) ? data : []
  } catch (e) {
    errorMessage.value = e.message || '加载租户列表失败'
    errorMessageTraceId.value = extractTraceId(e) || ''
    tenantOptions.value = []
  }
}

const fetchGitlabRegions = async () => {
  try {
    const response = await apiFetch('/api/system-admin/gitlab-regions/', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      errorMessage.value = data.error || data.detail || `加载 GitLab 区域失败（${response.status}）`
      errorMessageTraceId.value = extractTraceId(response) || ''
      gitlabRegions.value = []
      return
    }
    gitlabRegions.value = Array.isArray(data.regions) ? data.regions : []
  } catch (e) {
    errorMessage.value = e.message || '加载 GitLab 区域失败'
    errorMessageTraceId.value = extractTraceId(e) || ''
    gitlabRegions.value = []
  }
}

const loadMembership = async (tid) => {
  currentMembershipTier.value = ''
  currentMembershipLocked.value = false
  if (!tid) return
  try {
    const response = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/membership/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      return
    }
    const mem = data.membership || {}
    currentMembershipTier.value = mem.tier || ''
    currentMembershipLocked.value = Boolean(mem.admin_tier_locked)
  } catch {
    currentMembershipTier.value = ''
    currentMembershipLocked.value = false
  }
}

const openDropdown = () => {
  if (blurCloseTimer) {
    clearTimeout(blurCloseTimer)
    blurCloseTimer = null
  }
  dropdownOpen.value = true
}

const scheduleCloseDropdown = () => {
  if (blurCloseTimer) {
    clearTimeout(blurCloseTimer)
  }
  blurCloseTimer = setTimeout(() => {
    dropdownOpen.value = false
    blurCloseTimer = null
  }, 200)
}

const onSearchInput = () => {
  openDropdown()
  if (fetchTimer.value) {
    clearTimeout(fetchTimer.value)
  }
  fetchTimer.value = setTimeout(() => {
    fetchOptions(tenantSearch.value)
  }, 280)
}

const selectTenant = (t) => {
  if (blurCloseTimer) {
    clearTimeout(blurCloseTimer)
    blurCloseTimer = null
  }
  selectedTenant.value = t
  tenantSearch.value = t.name
  dropdownOpen.value = false
  loadMembership(t.id)
}

/** OPT-20260825-026: 从 /system-admin/grant-points/?tenant_id= 深链自动选中目标租户 */
const autoSelectFromQuery = async () => {
  const tid = route?.query?.tenant_id
  if (!tid) return
  try {
    const q = new URLSearchParams()
    q.set('search', String(tid))
    const response = await apiFetch(`/api/system-admin/accounts/admin/tenant-options/?${q.toString()}`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => [])
    const opts = Array.isArray(data) ? data : []
    const found = opts.find((o) => String(o.id) === String(tid))
    if (found) selectTenant(found)
  } catch (e) {
    // 深链预选失败不阻塞页面，运营仍可用下拉搜索
  }
}

onMounted(() => {
  fetchOptions('')
  fetchGitlabRegions()
  autoSelectFromQuery()
})

onBeforeUnmount(() => {
  if (fetchTimer.value) {
    clearTimeout(fetchTimer.value)
  }
  if (blurCloseTimer) {
    clearTimeout(blurCloseTimer)
  }
  if (successToastTimer) {
    clearTimeout(successToastTimer)
    successToastTimer = null
  }
})

const submit = async () => {
  errorMessage.value = ''
  errorMessageTraceId.value = ''
  const tid = selectedTenant.value?.id
  if (!tid) {
    errorMessage.value = '请选择租户。'
    return
  }

  const validLines = resourceLines.value.filter(
    (r) => Number.isFinite(Number(r.quantity)) && Number(r.quantity) >= 1,
  )
  const tier = String(membershipTier.value || '').trim()
  if (validLines.length === 0 && !tier) {
    errorMessage.value = '请至少填写一项有效的资源数量（≥1），或选择要修改的 VIP 等级。'
    return
  }
  if (validLines.some((r) => needsGitlabRegion(r.resource_type) && !String(r.region || '').trim())) {
    errorMessage.value = '请选择 GitLab 区域。'
    return
  }

  const resources = validLines.map((r) => {
    const item = {
      resource_type: r.resource_type,
      quantity: Math.floor(Number(r.quantity)),
      expires_at: r.expires_at ? r.expires_at + ' 23:59:59.000000' : '',
      reason: r.reason || '',
    }
    if (needsGitlabRegion(r.resource_type)) {
      item.region = String(r.region || '').trim()
    }
    return item
  })

  const payload = { resources }
  if (tier) {
    payload.membership_tier = tier
  }

  submitting.value = true
  try {
    const url = `/api/tenant/${encodeURIComponent(tid)}/billing/accounts/admin_grant_points/`
    // 点击防重：防抖 + Idempotency-Key（body 与 header 同值；后端双通道幂等）
    const outcome = await grantGuard.run(async ({ idempotencyKey, headers: ikHeaders }) => {
      const body = { ...payload }
      body.idempotency_key = idempotencyKey
      return apiFetch(url, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          ...ikHeaders,
        },
        body: JSON.stringify(body),
      })
    })
    if (outcome.skipped) return
    const response = outcome.result

    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      errorMessage.value =
        data.error || data.detail || Object.values(data).flat().join(' ') || `请求失败（${response.status}）`
      errorMessageTraceId.value = extractTraceId(response) || ''
      return
    }

    const grants = Array.isArray(data.grants) ? data.grants : []
    const mem = data.membership
    const memLabel = mem && mem.tier === 'vip1' ? 'VIP1' : mem && mem.tier === 'normal' ? '普通会员' : ''
    if (grants.length) {
      const parts = grants.map((g) => {
        const rt = resourceTypes.find((x) => x.value === g.resource_type)
        const label = rt?.label || g.resource_type
        const region = g.region ? ` · ${g.region}` : ''
        return `${label}${region} × ${g.quantity}`
      })
      const extra = memLabel ? `；会员等级：${memLabel}` : ''
      showSuccessToast(`赠送成功！${parts.join('、')}${extra}`)
    } else if (memLabel) {
      showSuccessToast(`会员等级已设为${memLabel}。`)
    } else {
      const quotaAfter = data.task_post_quota_after ?? data.points ?? ''
      showSuccessToast(`赠送成功！当前该租户任务帖配额：${quotaAfter}。`)
    }

    resourceLines.value = [freshLine()]
    membershipTier.value = ''
    if (tid) loadMembership(tid)
  } catch (e) {
    errorMessage.value = e.message || '网络错误'
    errorMessageTraceId.value = extractTraceId(e) || ''
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.2s ease;
}

.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
