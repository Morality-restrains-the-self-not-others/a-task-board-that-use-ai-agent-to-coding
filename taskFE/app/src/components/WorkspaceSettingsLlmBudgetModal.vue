<template>
  <div
    v-if="visible"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50"
    @click.self="emit('close')"
  >
    <div class="bg-white rounded-xl p-6 w-full max-w-5xl max-h-[90vh] overflow-y-auto" @keydown.enter="onEnterKey">
      <h4 class="text-lg font-bold mb-2">LLM 预算默认 · {{ workspaceLabel }}</h4>
      <p class="text-sm text-gray-500 mb-4">
        金额单位为人民币（元）。
      </p>
      <p v-if="loadError" class="text-sm text-red-600 mb-3" :data-traceId="loadErrorTraceId || undefined">{{ loadError }}</p>
      <p v-else-if="loading" class="text-sm text-gray-500 mb-3">加载中…</p>
      <div v-else-if="items.length === 0" class="text-sm text-gray-500 mb-4">
        当前没有可配置预算的模型端点。
      </div>
      <div v-else class="overflow-x-auto mb-4">
        <table class="min-w-full text-sm border border-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-2 py-2 text-left">供应商</th>
              <th class="px-2 py-2 text-left">端点</th>
              <th class="px-2 py-2 text-left">模型</th>
              <th class="px-2 py-2 text-left">Input（元/1M）</th>
              <th class="px-2 py-2 text-left">Output（元/1M）</th>
              <th class="px-2 py-2 text-left">默认上限（元）</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, idx) in items" :key="`${row.provider}-${row.model_name}-${idx}`">
              <td class="px-2 py-2 border-t">{{ row.provider }}</td>
              <td class="px-2 py-2 border-t truncate max-w-[12rem]" :title="row.base_url">{{ row.base_url }}</td>
              <td class="px-2 py-2 border-t">{{ row.model_name }}</td>
              <td class="px-2 py-2 border-t">
                <input v-model="row.input_price_per_1m" type="number" min="0" step="0.000001" class="w-28 px-2 py-1 border rounded" />
              </td>
              <td class="px-2 py-2 border-t">
                <input v-model="row.output_price_per_1m" type="number" min="0" step="0.000001" class="w-28 px-2 py-1 border rounded" />
              </td>
              <td class="px-2 py-2 border-t">
                <input v-model="row.budget_limit" type="number" min="0" step="0.01" class="w-28 px-2 py-1 border rounded" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="flex justify-end space-x-3">
        <button class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50" type="button" @click="emit('close')">
          取消
        </button>
        <button class="btn-primary" type="button" :disabled="saving || loading" @click="save">
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { llmBudgetDefaultsUrl } from '../utils/llmBudgetUrls.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  tenantId: { type: String, required: true },
  workspaceId: { type: String, default: '' },
  workspaceLabel: { type: String, default: '' },
})

const emit = defineEmits(['close', 'saved'])

const onEnterKey = (event) => {
  const tag = (event.target?.tagName || '').toLowerCase()
  if (tag === 'textarea') return
  if (!saving.value && !loading.value && !saveGuard.isBusy()) save()
}

const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const items = ref([])

// OPT-20260819-038: LLM 预算默认为资金路径，createClickGuard 防连点双发 PATCH。
const saveGuard = createClickGuard()

const loadItems = async () => {
  if (!props.visible || !props.workspaceId) return
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const resp = await apiFetch(
      llmBudgetDefaultsUrl(props.tenantId, props.workspaceId),
      { credentials: 'include' }
    )
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}))
      throw new Error(err.message || err.detail || '加载失败')
    }
    const body = await resp.json()
    items.value = (body.items || []).map((item) => ({ ...item }))
  } catch (e) {
    loadErrorTraceId.value = extractTraceId(e) || ''
    loadError.value = e.message || '加载失败'
    items.value = []
  } finally {
    loading.value = false
  }
}

const save = async () => {
  await saveGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      const resp = await apiFetch(
        llmBudgetDefaultsUrl(props.tenantId, props.workspaceId),
        {
          method: 'PATCH',
          credentials: 'include',
          headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
          body: JSON.stringify({ items: items.value }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.message || err.detail || '保存失败')
      }
      emit('saved')
      emit('close')
    } catch (e) {
      loadErrorTraceId.value = extractTraceId(e) || ''
      loadError.value = e.message || '保存失败'
    } finally {
      saving.value = false
    }
  })
}

watch(
  () => [props.visible, props.workspaceId],
  () => {
    if (props.visible) loadItems()
  }
)
</script>
