<template>
  <div data-alias="panel-system-admin-resource-pricing">
    <p class="text-sm text-gray-500 mb-2">
      直接管理各资源的<strong>单价</strong>。修改后<strong>即时生效</strong>，用户创建新订单时将按最新价格计算。
      不再使用「价格套餐」模式。
    </p>
    <div class="bg-blue-50 border border-blue-200 text-blue-950 text-sm rounded-lg p-3 mb-6">
      {{ policySummary }}
    </div>

    <!-- 当前定价（只读） -->
    <section
      v-if="currentPricing"
      class="mb-8 bg-white border border-gray-200 rounded-lg p-4"
      data-testid="admin-current-pricing-card"
    >
      <h3 class="text-lg font-medium text-gray-800 mb-1">当前资源定价</h3>
      <p class="text-xs text-gray-500 mb-3">
        来自 <code>GET /api/system-admin/resource-pricing/</code>（billing_unit 实时价格）
      </p>
      <dl class="grid grid-cols-2 gap-x-6 gap-y-1.5 text-sm items-baseline">
        <dt class="text-gray-500">任务（创建帖次数）</dt>
        <dd class="text-gray-900 font-medium tabular-nums text-right whitespace-nowrap">
          {{ currentPricing.task_post?.price_yuan || '—' }} 元/帖/12个月
          <span class="ml-2 text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-600">{{ tierLabel(currentPricing.task_post?.required_tier) }}</span>
        </dd>
        <dt class="text-gray-500">GitLab 磁盘</dt>
        <dd class="text-gray-900 font-medium tabular-nums text-right whitespace-nowrap">
          {{ currentPricing.gitlab_disk?.price_yuan || '—' }} 元/GB/月
          <span class="ml-2 text-xs px-1.5 py-0.5 rounded bg-amber-100 text-amber-800">{{ tierLabel(currentPricing.gitlab_disk?.required_tier) }}</span>
          <span
            v-if="currentPricing.gitlab_disk?.min_quantity"
            class="ml-2 text-xs text-gray-500 font-normal"
            data-testid="admin-gitlab-disk-min-gb"
          >起购 {{ currentPricing.gitlab_disk.min_quantity }} GB</span>
        </dd>
        <dt class="text-gray-500">GitLab 流量费</dt>
        <dd class="text-gray-900 font-medium tabular-nums text-right whitespace-nowrap">
          {{ currentPricing.gitlab_traffic?.price_yuan || '—' }} 元/GB
          <span class="ml-2 text-xs px-1.5 py-0.5 rounded bg-amber-100 text-amber-800">{{ tierLabel(currentPricing.gitlab_traffic?.required_tier) }}</span>
        </dd>
      </dl>
      <p v-if="currentPricing.gitlab_traffic?.requires_unit_type" class="text-xs text-amber-700 mt-2">
        ⚠ 需先购买 {{ getUnitTypeLabel(currentPricing.gitlab_traffic.requires_unit_type) }}
      </p>
      <p v-if="currentPricing.updated_at" class="text-xs text-gray-400 mt-3">
        最近更新：{{ currentPricing.updated_at }}
      </p>
    </section>

    <!-- 会员等级与购买门槛 -->
    <section
      v-if="currentPricing?.tiers"
      class="mb-8 bg-white border border-gray-200 rounded-lg p-4"
    >
      <h3 class="text-lg font-medium text-gray-800 mb-3">会员等级与购买门槛</h3>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div
          v-for="(tier, key) in currentPricing.tiers"
          :key="key"
          class="border rounded-lg p-3"
          :class="key === 'vip1' ? 'border-amber-300 bg-amber-50' : 'border-gray-200 bg-gray-50'"
        >
          <div class="flex items-center gap-2 mb-1">
            <span
              class="text-xs px-2 py-0.5 rounded font-medium"
              :class="key === 'vip1' ? 'bg-amber-200 text-amber-900' : 'bg-gray-200 text-gray-700'"
            >{{ tier.name }}</span>
          </div>
          <p class="text-sm text-gray-600">{{ tier.description }}</p>
          <p v-if="tier.upgrade_threshold_yuan" class="text-xs text-amber-700 mt-1">
            升级条件：累计消费满 {{ tier.upgrade_threshold_yuan }} 元自动升级
          </p>
        </div>
      </div>
    </section>

    <div v-if="loadError" class="text-red-600 text-sm mb-4" :data-traceId="loadTraceId">{{ loadError }}</div>

    <!-- 修改定价表单 -->
    <section class="mb-8">
      <h3 class="text-lg font-medium text-gray-800 mb-3">修改资源定价</h3>
      <form
        class="space-y-6"
        @submit.prevent="updatePricing"
      >
        <!-- 任务定价分组 -->
        <fieldset class="bg-white border border-gray-200 rounded-lg p-4">
          <legend class="text-sm font-semibold text-gray-700 bg-white px-2 -ml-2">任务定价</legend>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm text-gray-700 mb-1"
                >任务（元/帖/12个月）</label
              >
              <input
                v-model="form.task_post_price_yuan"
                type="number"
                min="0"
                step="0.01"
                class="w-full border border-gray-300 rounded-md px-3 py-2"
                placeholder="例如 0.55"
              />
              <p class="mt-1 text-xs text-gray-500">当前：{{ currentPricing?.task_post?.price_yuan || '—' }} 元/帖/12个月。续存只消耗 1 帖配额，不另扣费；存续期再延长 12 个月。</p>
            </div>
          </div>
        </fieldset>

        <!-- GitLab 资源定价分组 -->
        <fieldset class="bg-white border border-gray-200 rounded-lg p-4">
          <legend class="text-sm font-semibold text-gray-700 bg-white px-2 -ml-2">GitLab 资源定价</legend>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm text-gray-700 mb-1"
                >GitLab 磁盘（元/GB/月）</label
              >
              <input
                v-model="form.gitlab_disk_price_yuan"
                type="number"
                min="0"
                step="0.01"
                class="w-full border border-gray-300 rounded-md px-3 py-2"
                placeholder="例如 4.00"
              />
              <p class="mt-1 text-xs text-gray-500">当前：{{ currentPricing?.gitlab_disk?.price_yuan || '—' }} 元/GB/月</p>
            </div>
            <div>
              <label class="block text-sm text-gray-700 mb-1"
                >GitLab 流量费（元/GB）</label
              >
              <input
                v-model="form.gitlab_traffic_price_yuan"
                type="number"
                min="0"
                step="0.01"
                class="w-full border border-gray-300 rounded-md px-3 py-2"
                placeholder="例如 1.00"
              />
              <p class="mt-1 text-xs text-gray-500">当前：{{ currentPricing?.gitlab_traffic?.price_yuan || '—' }} 元/GB。同区域内网不计费。</p>
              <p v-if="currentPricing?.gitlab_traffic?.requires_unit_type" class="mt-1 text-xs text-amber-700">
                ⚠ 购买前置条件：用户须先购买 {{ getUnitTypeLabel(currentPricing.gitlab_traffic.requires_unit_type) }}
              </p>
            </div>
            <div>
              <!-- OPT-20260726-020: 管理员可通过下拉框编辑 requires_unit_type 依赖关系 -->
              <label class="block text-sm text-gray-700 mb-1">前置依赖资源（可选）</label>
              <select
                v-model="form.gitlab_traffic_requires_unit_type"
                class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value="">无依赖</option>
                <option value="gitlab_disk">GitLab 磁盘</option>
              </select>
              <p class="mt-1 text-xs text-gray-500">
                当前：{{ currentPricing?.gitlab_traffic?.requires_unit_type ? getUnitTypeLabel(currentPricing.gitlab_traffic.requires_unit_type) : '无依赖' }}
              </p>
            </div>
          </div>
        </fieldset>

        <!-- 操作按钮 -->
        <div class="bg-white border border-gray-200 rounded-lg p-4">
          <div v-if="saveError" class="text-red-600 text-sm mb-2" :data-traceId="saveTraceId">{{ saveError }}</div>
          <div v-if="saveOk" class="text-green-700 text-sm mb-2">{{ saveOk }}</div>
          <button
            type="submit"
            class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
            :disabled="saving"
          >
            {{ saving ? '保存中…' : '更新定价' }}
          </button>
        </div>
      </form>
    </section>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils'
import { getCookie } from '../utils/cookieUtils'
import { safeJson } from '../utils/safeResponseJson.js'

/* @alias:panel-system-admin-resource-pricing */

const policySummary = ref('加载中…')
const currentPricing = ref(null)
const loadError = ref('')
const loadTraceId = ref('')
const saveError = ref('')
const saveTraceId = ref('')
const saveOk = ref('')
const saving = ref(false)
const form = ref({
  task_post_price_yuan: '',
  gitlab_disk_price_yuan: '',
  gitlab_traffic_price_yuan: '',
  gitlab_traffic_requires_unit_type: '',  // OPT-20260726-020
})

const tierLabel = (tier) => {
  const labels = { normal: '普通会员', vip1: 'VIP1' }
  return labels[tier] || tier || '—'
}

const getUnitTypeLabel = (unitType) => {
  const labels = {
    server_start: '任务',
    gitlab_disk: 'GitLab 磁盘',
    gitlab_traffic: 'GitLab 流量费'}
  return labels[unitType] || unitType
}



const loadPricing = async () => {
  loadError.value = ''
  loadTraceId.value = ''
  let requestTraceId = ''
  try {
    const r = await apiFetch('/api/system-admin/resource-pricing/', { method: 'GET' })
    requestTraceId = r.traceId || ''
    if (!r.ok) {
      const d = await safeJson(r, {})
      loadTraceId.value = d._traceId || requestTraceId
      loadError.value = extractErrorMessage(d, r) || `请求失败 (HTTP ${r.status})，请确认以系统管理员登录。`
      return
    }
    const d = await safeJson(r, {})
    if (d.status === 'success') {
      currentPricing.value = d.pricing || {}
      policySummary.value = d.message || '资源定价直接管理，修改后即时生效。新订单按最新价格计算。'
    }
  } catch (e) {
    loadTraceId.value = e.traceId || requestTraceId
    loadError.value = '加载失败: ' + e.message
  }
}

const updatePricing = async () => {
  saveError.value = ''
  saveTraceId.value = ''
  saveOk.value = ''
  saving.value = true
  let requestTraceId = ''
  try {
    const body = {}
    if (form.value.task_post_price_yuan !== '') {
      body.task_post_price_yuan = Number(form.value.task_post_price_yuan)
    }
    if (form.value.gitlab_disk_price_yuan !== '') {
      body.gitlab_disk_price_yuan = Number(form.value.gitlab_disk_price_yuan)
    }
    if (form.value.gitlab_traffic_price_yuan !== '') {
      body.gitlab_traffic_price_yuan = Number(form.value.gitlab_traffic_price_yuan)
    }
    // OPT-20260726-020: 允许管理员通过 UI 修改前置依赖
    if (form.value.gitlab_traffic_requires_unit_type !== '' || currentPricing.value?.gitlab_traffic?.requires_unit_type) {
      body.gitlab_traffic_requires_unit_type = form.value.gitlab_traffic_requires_unit_type || ''
    }
    if (Object.keys(body).length === 0) {
      saveError.value = '请至少填写一项'
      return
    }

    const r = await apiFetch('/api/system-admin/resource-pricing/', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'X-CSRFToken': getCookie('csrftoken') },
      body: JSON.stringify(body)})
    // 先保存 apiFetch 注入的 traceId（来自响应头/请求头），防止后续 .json() 抛错时丢失
    requestTraceId = r.traceId || ''
    if (!r.ok) {
      // 安全解析 JSON：响应体可能为 HTML 错误页（网关 502/后台 crash 等）
      const d = await safeJson(r, {})
      saveTraceId.value = d._traceId || requestTraceId
      saveError.value = extractErrorMessage(d, r) || `更新失败 (HTTP ${r.status})`
      return
    }
    // 即使 r.ok 也可能 .json() 失败（网关/代理返回了 HTML 而非 JSON）
    const d = await safeJson(r, null)
    if (d === null) {
      saveTraceId.value = requestTraceId
      saveError.value = '服务器返回了无法解析的响应，请稍后重试'
      return
    }
    saveOk.value = d.message || '定价已更新'
    currentPricing.value = d.pricing || currentPricing.value
    form.value = { task_post_price_yuan: '', gitlab_disk_price_yuan: '', gitlab_traffic_price_yuan: '', gitlab_traffic_requires_unit_type: '' }
  } catch (e) {
    saveTraceId.value = e.traceId || requestTraceId
    saveError.value = '更新失败: ' + e.message
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadPricing()
})
</script>
