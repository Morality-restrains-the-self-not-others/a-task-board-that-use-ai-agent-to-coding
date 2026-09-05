<template>
  <div v-if="visible" class="p-4 bg-gray-50 rounded-lg border border-gray-200">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-sm font-semibold text-gray-800">LLM 预算（CNY）</h3>
      <button
        v-if="canEdit"
        class="text-sm text-primary hover:text-primary/80"
        type="button"
        :disabled="saving"
        @click="saveOverrides"
      >
        {{ saving ? '保存中…' : '保存覆盖' }}
      </button>
    </div>
    <p
      v-if="loadError"
      class="text-sm text-red-600 mb-2"
      v-bind="loadErrorTraceId ? { 'data-traceId': loadErrorTraceId } : {}"
    >{{ loadError }}</p>
    <p v-else-if="loading" class="text-sm text-gray-500">加载预算中…</p>
    <p v-else-if="items.length === 0" class="text-sm text-gray-500">
      暂无 sub-token 模型预算配置。
    </p>
    <div v-else class="space-y-3">
      <div
        v-for="(item, idx) in items"
        :key="`${item.provider}-${item.base_url}-${item.model_name}`"
        class="rounded-md border border-gray-200 bg-white p-3"
      >
        <div class="flex flex-wrap items-center justify-between gap-2 mb-2">
          <div class="text-sm font-medium text-gray-900">
            {{ item.provider }} · {{ item.model_name }}
          </div>
          <span
            v-if="item.exhausted"
            class="text-xs px-2 py-0.5 rounded bg-red-100 text-red-700"
          >
            已超额
          </span>
        </div>
        <p class="text-xs text-gray-500 mb-2 truncate">{{ item.base_url }}</p>
        <div class="mb-2">
          <div class="h-2 bg-gray-200 rounded overflow-hidden">
            <div
              class="h-full bg-primary transition-all"
              :style="{ width: `${progressPct(item)}%` }"
            />
          </div>
          <p class="text-xs text-gray-600 mt-1">
            已用 ¥{{ formatAmount(item.spent_amount) }} /
            上限 ¥{{ formatAmount(item.effective_budget_limit) }}
            （来源：{{ sourceLabel(item.budget_limit_source) }}）
          </p>
        </div>
        <div v-if="canEdit" class="flex items-center gap-2">
          <label class="text-xs text-gray-600">覆盖上限（元）</label>
          <input
            v-model="draftLimits[idx]"
            type="number"
            min="0"
            step="0.01"
            class="w-32 px-2 py-1 text-sm border border-gray-300 rounded"
          />
          <button
            v-if="canRaise"
            type="button"
            class="text-xs text-amber-700 hover:underline"
            @click="raiseBudget(item, idx)"
          >
            临时上调
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import { taskModelBudgetsRaiseUrl, taskModelBudgetsUrl } from '../../utils/llmBudgetUrls.js'

const props = defineProps({
  tenantId: { type: String, required: true },
  workspaceId: { type: String, required: true },
  taskId: { type: String, required: true },
  canEdit: { type: Boolean, default: false },
  canRaise: { type: Boolean, default: false },
})

const visible = ref(false)
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const items = ref([])
const draftLimits = ref([])

const budgetsUrl = () =>
  taskModelBudgetsUrl(props.tenantId, props.workspaceId, props.taskId)

const raiseUrl = () => taskModelBudgetsRaiseUrl(props.tenantId, props.workspaceId, props.taskId)

const formatAmount = (v) => {
  const n = Number(v)
  return Number.isFinite(n) ? n.toFixed(2) : '0.00'
}

const progressPct = (item) => {
  const limit = Number(item.effective_budget_limit)
  const spent = Number(item.spent_amount)
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.round((spent / limit) * 100))
}

const sourceLabel = (source) => {
  if (source === 'override') return '任务覆盖'
  if (source === 'raised') return '临时上调'
  return '继承'
}

const loadFeatureEnabled = async () => {
  try {
    const resp = await apiFetch(`/api/cloud/feature-params/tenant_id/${props.tenantId}?view=summary`, {
      credentials: 'include',
    })
    if (!resp.ok) {
      visible.value = false
      return
    }
    const body = await resp.json()
    visible.value = Boolean(body?.data?.llm_budget_enabled)
  } catch {
    visible.value = false
  }
}

const loadBudgets = async () => {
  if (!visible.value) return
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const resp = await apiFetch(budgetsUrl(), { credentials: 'include' })
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}))
      loadErrorTraceId.value = resp.traceId || ''
      throw new Error(err.message || '加载预算失败')
    }
    const body = await resp.json()
    items.value = Array.isArray(body.items) ? body.items : []
    draftLimits.value = items.value.map((i) => i.effective_budget_limit ?? i.budget_limit ?? '0')
  } catch (e) {
    loadError.value = e.message || '加载预算失败'
    if (!loadErrorTraceId.value) loadErrorTraceId.value = extractTraceId(e)
  } finally {
    loading.value = false
  }
}

// OPT-20260819-038: LLM 预算覆盖/上调为资金路径，createClickGuard 防连点双发。
const saveOverridesGuard = createClickGuard()
const raiseBudgetGuard = createClickGuard()

const saveOverrides = async () => {
  await saveOverridesGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      const payload = {
        items: items.value.map((item, idx) => ({
          provider: item.provider,
          base_url: item.base_url,
          model_name: item.model_name,
          budget_limit: String(draftLimits.value[idx] ?? '0'),
        })),
      }
      const resp = await apiFetch(budgetsUrl(), {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(payload),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        loadErrorTraceId.value = resp.traceId || ''
        throw new Error(err.message || '保存失败')
      }
      await loadBudgets()
    } catch (e) {
      loadError.value = e.message || '保存失败'
      if (!loadErrorTraceId.value) loadErrorTraceId.value = extractTraceId(e)
    } finally {
      saving.value = false
    }
  })
}

const raiseBudget = async (item, idx) => {
  const newLimit = draftLimits.value[idx]
  if (!window.confirm(`确认将 ${item.model_name} 预算临时上调至 ¥${newLimit}？`)) {
    return
  }
  await raiseBudgetGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      const resp = await apiFetch(raiseUrl(), {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          provider: item.provider,
          base_url: item.base_url,
          model_name: item.model_name,
          new_budget_limit: String(newLimit),
          confirm: true,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        loadErrorTraceId.value = resp.traceId || ''
        throw new Error(err.message || '上调失败')
      }
      await loadBudgets()
    } catch (e) {
      loadError.value = e.message || '上调失败'
      if (!loadErrorTraceId.value) loadErrorTraceId.value = extractTraceId(e)
    } finally {
      saving.value = false
    }
  })
}

watch(
  () => [props.tenantId, props.workspaceId, props.taskId],
  async () => {
    await loadFeatureEnabled()
    await loadBudgets()
  }
)

onMounted(async () => {
  await loadFeatureEnabled()
  await loadBudgets()
})
</script>
