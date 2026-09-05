<template>
  <div
    v-if="show"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50"
    data-testid="code-lang-options-modal"
  >
    <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl" @keydown.enter="onEnterKey">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">
          主要编程语言可选值
          <span v-if="workspaceName" class="text-sm font-normal text-gray-500">
            · {{ workspaceName }}
          </span>
        </h3>
        <button type="button" class="text-gray-400 hover:text-gray-600" @click="emit('close')">×</button>
      </div>
      <p class="text-sm text-gray-600 mb-2">每行一个可选值；留空保存将恢复默认 go / rust / js。</p>
      <textarea
        v-model="draftText"
        rows="6"
        class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary font-mono text-sm"
        data-testid="code-lang-options-textarea"
        placeholder="go&#10;rust&#10;js"
      />
      <p
        v-if="errorMessage"
        class="mt-2 text-sm text-red-600"
        data-testid="code-lang-options-error"
        :data-traceId="errorTraceId || undefined"
      >
        {{ errorMessage }}
      </p>
      <div class="mt-4 flex justify-end space-x-3">
        <button
          type="button"
          class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
          @click="emit('close')"
        >
          取消
        </button>
        <button
          type="button"
          class="btn-primary disabled:opacity-50"
          data-testid="code-lang-options-save"
          :disabled="saving"
          @click="save"
        >
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { DEFAULT_CODE_LANG_OPTIONS, normalizeCodeLangOptions, codeLangOptionsFromResponse } from '../utils/codeLangOptions.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const props = defineProps({
  show: { type: Boolean, default: false },
  tenantId: { type: [String, Number], default: '' },
  workspaceId: { type: [String, Number], default: '' },
  workspaceName: { type: String, default: '' },
})

const emit = defineEmits(['close', 'saved'])

const onEnterKey = (event) => {
  const tag = (event.target?.tagName || '').toLowerCase()
  if (tag === 'textarea') return
  if (!saving.value) save()
}

const draftText = ref(DEFAULT_CODE_LANG_OPTIONS.join('\n'))
const saving = ref(false)
const errorMessage = ref('')
const errorTraceId = ref('')

const saveGuard = createClickGuard()

const loadOptions = async () => {
  errorMessage.value = ''
  errorTraceId.value = ''
  draftText.value = DEFAULT_CODE_LANG_OPTIONS.join('\n')
  const tenantId = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  if (!tenantId || !wid) return
  try {
    const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}/${wid}/code-lang-options/`, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const body = await response.json().catch(() => ({}))
    if (!response.ok) {
      errorMessage.value = body?.error || body?.message || `加载失败 (${response.status})`
      errorTraceId.value = extractTraceId(response) || extractTraceId(body) || ''
      return
    }
    draftText.value = codeLangOptionsFromResponse(body).join('\n')
  } catch (error) {
    errorMessage.value = error?.message || '加载主要编程语言可选值失败'
  }
}

watch(
  () => [props.show, props.workspaceId, props.tenantId],
  ([show]) => {
    if (show) void loadOptions()
  },
  { immediate: true },
)

const save = async () => {
  const tenantId = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  if (!tenantId || !wid) return
  // OPT-20260819-038: 保存工作区设置是配置写操作，防连点/超时重试双发
  await saveGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    try {
      const options = normalizeCodeLangOptions(
        String(draftText.value || '')
          .split(/\r?\n/)
          .map((line) => line.trim())
          .filter(Boolean),
      )
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}/${wid}/code-lang-options/`, {
        method: 'PUT',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json', 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ options }),
      })
      const body = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorMessage.value = body?.error || body?.message || `保存失败 (${response.status})`
        errorTraceId.value = extractTraceId(response) || extractTraceId(body) || ''
        return
      }
      emit('saved', options)
      emit('close')
    } catch (error) {
      errorMessage.value = error?.message || '保存主要编程语言可选值失败'
      showRequestError(errorMessage.value, error)
    } finally {
      saving.value = false
    }
  })
}
</script>
