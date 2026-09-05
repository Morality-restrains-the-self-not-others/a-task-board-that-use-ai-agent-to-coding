<template>
  <div class="p-6" data-alias="view-order-create">
    <h1 class="text-2xl font-bold mb-6">购买资源</h1>

    <div
      v-if="membership"
      class="rounded-lg p-4 mb-6 text-sm flex items-center justify-between"
      :class="membership.tier === 'vip1' ? 'bg-amber-50 border border-amber-200' : 'bg-gray-50 border border-gray-200'"
    >
      <div>
        <span class="font-medium">当前会员等级：</span>
        <span
          class="ml-2 px-2 py-0.5 rounded text-xs font-medium"
          :class="membership.tier === 'vip1' ? 'bg-amber-200 text-amber-900' : 'bg-gray-200 text-gray-700'"
        >{{ tierLabel(membership.tier) }}</span>
        <span class="ml-3 text-gray-500">
          累计消费 {{ membership.cumulative_consumption_yuan }} 元
        </span>
      </div>
      <div v-if="membership.tier === 'normal'" class="text-gray-500 text-right">
        距 VIP1 还差 <strong class="text-amber-700">{{ membership.remaining_to_upgrade_yuan }}</strong> 元
      </div>
      <div v-else class="text-amber-700 text-right">
        ✅ 可购买全部资源
      </div>
    </div>

    <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6 text-sm">
      <p class="font-medium mb-2">当前资源单价</p>
      <div v-if="pricingLoading" class="text-gray-500">加载中…</div>
      <ul v-else class="space-y-1">
        <li>任务：<strong>{{ pricing.task_post?.price_yuan }} 元/帖/12个月</strong>（创建帖消耗 1 次配额，含 12 个月存续期；续存再消耗 1 次配额，不另扣费） <span class="text-xs text-gray-500">— {{ tierLabel(pricing.task_post?.required_tier) }}可购买</span></li>
        <li v-if="isVip1">GitLab 磁盘：<strong>{{ pricing.gitlab_disk?.price_yuan }} 元/GB/月</strong></li>
        <li v-if="isVip1">GitLab 流量：<strong>{{ pricing.gitlab_traffic?.price_yuan }} 元/GB</strong>（同区域内网不计费）</li>
      </ul>
    </div>

    <div class="bg-white rounded-xl shadow-md p-6 mb-6">
      <h2 class="text-lg font-semibold mb-4">选择资源</h2>

      <div class="border border-gray-200 rounded-lg p-4 mb-4">
        <div class="flex justify-between items-center mb-2">
          <label class="font-medium text-gray-800">任务帖（创建帖次数）</label>
          <span class="text-sm text-gray-500">单价 {{ pricing.task_post?.price_yuan || '—' }} 元/帖/12个月</span>
        </div>
        <div class="flex items-center gap-3">
          <input
            v-model.number="form.taskPostQty"
            type="number"
            min="0"
            step="1"
            class="w-32 border border-gray-300 rounded-md px-3 py-2"
            @input="recalculate"
          />
          <span class="text-gray-600">帖</span>
          <span v-if="form.taskPostQty > 0" class="text-primary font-medium ml-auto">
            小计：{{ taskPostSubtotal }} 元
          </span>
        </div>
      </div>

      <div
        v-if="isVip1"
        class="border rounded-lg p-4 mb-4 border-gray-200"
        data-testid="order-gitlab-resources"
      >
        <div class="flex justify-between items-center mb-2">
          <label class="font-medium text-gray-800">GitLab 磁盘</label>
          <span class="text-sm text-gray-500">单价 {{ pricing.gitlab_disk?.price_yuan || '—' }} 元/GB/月</span>
        </div>
        <label class="block text-xs text-gray-600 mb-1">区域</label>
        <select
          v-model="form.gitlabRegion"
          class="w-full max-w-md border border-gray-300 rounded-md px-3 py-2 mb-3"
          data-testid="order-gitlab-region"
        >
          <option value="">请选择区域</option>
          <optgroup v-for="g in regionGroups" :key="g.provider" :label="g.label">
            <option v-for="r in g.regions" :key="'gitlab-'+r.slug" :value="r.slug">{{ r.name || r.slug }}</option>
          </optgroup>
        </select>
        <p
          v-if="gitlabPendingNode"
          class="text-xs text-amber-700 mb-3"
          data-testid="order-gitlab-pending-node-hint"
        >
          该区域节点尚未部署。下单后由平台在对应云上创建服务节点并挂载，再为您开通 GitLab。
        </p>
        <div class="flex items-center gap-3 flex-wrap">
          <input
            v-model.number="form.diskGB"
            type="number"
            min="0"
            step="1"
            class="w-32 border border-gray-300 rounded-md px-3 py-2"
            data-testid="order-gitlab-disk-gb"
            @input="recalculate"
          />
          <span class="text-gray-600">GB ×</span>
          <select v-model.number="form.diskMonths" class="border border-gray-300 rounded-md px-3 py-2" @change="recalculate">
            <option v-for="m in diskMonthOptions" :key="m" :value="m">{{ m }} 个月</option>
          </select>
          <span v-if="form.diskGB > 0" class="text-primary font-medium ml-auto">
            小计：{{ diskSubtotal }} 元
          </span>
        </div>
        <p class="text-xs text-gray-500 mt-2" data-testid="order-gitlab-disk-min-hint">
          起购 {{ gitlabDiskMinGB }} GB（0 表示本单不购买磁盘）
        </p>

        <div class="border-t border-gray-200 mt-4 pt-4">
          <div class="flex justify-between items-center mb-2">
            <label class="font-medium text-gray-800">GitLab 流量</label>
            <span class="text-sm text-gray-500">单价 {{ pricing.gitlab_traffic?.price_yuan || '—' }} 元/GB</span>
          </div>
          <div class="flex items-center gap-3">
            <input
              v-model.number="form.trafficGB"
              type="number"
              min="0"
              step="1"
              class="w-32 border border-gray-300 rounded-md px-3 py-2"
              data-testid="order-gitlab-traffic-gb"
              @input="recalculate"
            />
            <span class="text-gray-600">GB</span>
            <span v-if="form.trafficGB > 0" class="text-primary font-medium ml-auto">
              小计：{{ trafficSubtotal }} 元
            </span>
          </div>
        </div>
      </div>

      <div class="border-t border-gray-200 pt-4 mt-4">
        <div class="flex justify-between items-center">
          <span class="text-lg font-semibold">合计</span>
          <span class="text-2xl font-bold text-primary">{{ totalYuan }} 元</span>
        </div>
        <p v-if="totalCents === 0" class="text-sm text-red-500 mt-2">请至少选择一项资源</p>

        <div v-if="needsConstructionNote" class="mt-4">
          <label class="block text-sm font-medium text-gray-800 mb-1" for="order-buyer-note">施工留言（选填）</label>
          <p class="text-xs text-gray-500 mb-2">GitLab 磁盘需人工开通，可留下路径、容量或联系说明，最多 2000 字。</p>
          <textarea
            id="order-buyer-note"
            v-model="form.buyerNote"
            rows="3"
            maxlength="2000"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            data-testid="order-buyer-note"
            placeholder="例如：请将组开到 team-foo"
          />
        </div>

        <div v-if="createError" class="text-red-600 text-sm mt-2" v-bind="createErrorTraceId ? { 'data-traceId': createErrorTraceId } : {}">{{ createError }}</div>

        <div class="mt-4 flex gap-3">
          <button
            type="button"
            class="px-6 py-2.5 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
            :disabled="totalCents === 0 || creating"
            :aria-busy="creating"
            @click="createOrder"
          >
            {{ creating ? '创建中…' : '创建订单' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { DISK_MONTH_OPTIONS } from '../composables/useGitlabResourcePurchase.js'
import { groupGitlabRegionsByProvider, isGitlabRegionPendingNode } from '../utils/gitlabRegionSelect.js'

const diskMonthOptions = DISK_MONTH_OPTIONS

/* @alias:view-order-create */

const route = useRoute()
const router = useRouter()
const tenantId = computed(() => route.params.tenant)

const pricingLoading = ref(true)
const pricing = ref({})
const membership = ref(null)

const isVip1 = computed(() => membership.value?.tier === 'vip1')
const gitlabDiskMinGB = computed(() => {
  const n = Number(pricing.value.gitlab_disk?.min_quantity)
  return Number.isFinite(n) && n > 0 ? n : 10
})

const tierLabel = (tier) => {
  const labels = { normal: '普通会员', vip1: 'VIP1' }
  return labels[tier] || tier || '—'
}

const form = ref({ taskPostQty: 0, diskGB: 0, diskMonths: 1, trafficGB: 0, gitlabRegion: '', buyerNote: '' })
const availableRegions = ref([])
const regionGroups = computed(() => groupGitlabRegionsByProvider(availableRegions.value))
const selectedGitlabRegion = computed(() =>
  availableRegions.value.find((r) => r.slug === form.value.gitlabRegion) || null)
const gitlabPendingNode = computed(() => isGitlabRegionPendingNode(selectedGitlabRegion.value))
const creating = ref(false)
const createError = ref('')
const createErrorTraceId = ref('')
const createOrderGuard = createClickGuard()

const taskPostSubtotal = computed(() => {
  if (!form.value.taskPostQty || !pricing.value.task_post?.price_yuan) return '0.00'
  return (Number(pricing.value.task_post.price_yuan) * form.value.taskPostQty).toFixed(2)
})

const diskSubtotal = computed(() => {
  if (!form.value.diskGB || !pricing.value.gitlab_disk?.price_yuan) return '0.00'
  return (Number(pricing.value.gitlab_disk.price_yuan) * form.value.diskGB * form.value.diskMonths).toFixed(2)
})

const trafficSubtotal = computed(() => {
  if (!form.value.trafficGB || !pricing.value.gitlab_traffic?.price_yuan) return '0.00'
  return (Number(pricing.value.gitlab_traffic.price_yuan) * form.value.trafficGB).toFixed(2)
})

const totalYuan = computed(() => {
  const t = Number(taskPostSubtotal.value) + Number(diskSubtotal.value) + Number(trafficSubtotal.value)
  return t.toFixed(2)
})

const totalCents = computed(() => Math.round(Number(totalYuan.value) * 100))

const needsConstructionNote = computed(() => Number(form.value.diskGB) > 0 || gitlabPendingNode.value)

const loadPricing = async () => {
  pricingLoading.value = true
  try {
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/order-pricing/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    if (r.ok) {
      const d = await r.json().catch(() => ({}))
      pricing.value = d.pricing || {}
    }
  } catch (e) {
    console.error('加载定价失败', e)
  } finally {
    pricingLoading.value = false
  }
}

const loadMembership = async () => {
  try {
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/membership/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    if (r.ok) {
      const d = await r.json().catch(() => ({}))
      membership.value = d.membership || null
    }
  } catch (e) {
    console.error('加载会员等级失败', e)
  }
}

const recalculate = () => { /* computed reactive */ }

const createOrder = async () => {
  await createOrderGuard.run(async ({ idempotencyKey }) => {
    createError.value = ''
    createErrorTraceId.value = ''
    creating.value = true
    try {
      const gitlabRegion = String(form.value.gitlabRegion || '').trim()
      if ((form.value.diskGB > 0 || form.value.trafficGB > 0) && !gitlabRegion) {
        createError.value = '请先选择 GitLab 区域'
        return
      }
      if (form.value.diskGB > 0 && form.value.diskGB < gitlabDiskMinGB.value) {
        createError.value = `GitLab 磁盘起购 ${gitlabDiskMinGB.value} GB`
        return
      }
      const items = []
      if (form.value.taskPostQty > 0) items.push({ resource_type: 'task_post', quantity: form.value.taskPostQty })
      if (form.value.diskGB > 0) items.push({ resource_type: 'gitlab_disk', quantity: form.value.diskGB, disk_months: form.value.diskMonths, region: gitlabRegion })
      if (form.value.trafficGB > 0) items.push({ resource_type: 'gitlab_traffic', quantity: form.value.trafficGB, region: gitlabRegion })

      const payload = { items }
      if (needsConstructionNote.value) {
        const note = String(form.value.buyerNote || '').trim()
        if (note) payload.buyer_note = note
      }

      const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/orders/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify(payload),
      })
      const d = await r.json().catch(() => ({}))
      if (!r.ok) {
        createError.value = extractErrorMessage(d, r) || d.error || '创建订单失败'
        createErrorTraceId.value = d.trace_id || d._traceId || r.traceId || ''
        return
      }
      const orderId = d.id != null ? String(d.id) : ''
      if (!orderId) {
        createError.value = '创建成功但缺少订单 ID'
        return
      }
      // Vue Router 4 named route: https://router.vuejs.org/guide/essentials/navigation.html
      await router.push({
        name: 'billing_order_detail',
        params: { tenant: String(tenantId.value), orderId },
      })
    } catch (e) {
      createError.value = '创建订单失败: ' + e.message
      createErrorTraceId.value = e.traceId || ''
    } finally {
      creating.value = false
    }
  })
}

const loadRegions = async () => {
  try {
    const r = await apiFetch(`/api/billing/gitlab-regions/tenant_id/${encodeURIComponent(tenantId.value)}/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (r.ok) {
      const d = await r.json().catch(() => ({}))
      availableRegions.value = Array.isArray(d.regions)
        ? d.regions.filter((r) => r && r.slug && r.is_active !== false && r.is_active !== 0)
        : []
    }
  } catch (e) {
    console.error('加载 GitLab 区域失败', e)
  }
}

onMounted(() => {
  loadPricing()
  loadMembership()
  loadRegions()
})
</script>
