<template>
  <div
    v-if="show"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50"
    data-testid="create-task-field-settings-modal"
  >
    <div class="bg-white rounded-xl p-6 w-full max-w-lg shadow-xl max-h-[90vh] flex flex-col" @keydown.enter="onEnterKey">
      <div class="flex justify-between items-center mb-4 shrink-0">
        <h3 class="text-lg font-semibold text-gray-900">
          创建任务可选字段
          <span v-if="workspaceName" class="text-sm font-normal text-gray-500">
            · {{ workspaceName }}
          </span>
        </h3>
        <button type="button" class="text-gray-400 hover:text-gray-600" @click="emit('close')">×</button>
      </div>
      <p class="text-sm text-gray-600 mb-3 shrink-0">
        勾选后，成员在工作面板创建任务时才会看到对应输入项。标题、进度与交付物类别始终显示。
        任务类型 / 编程语言开启时可编辑其下拉可选值。
      </p>
      <div class="space-y-3 overflow-y-auto min-h-0 flex-1 pr-1">
        <div
          v-for="key in fieldKeys"
          :key="key"
          class="rounded-md border border-gray-100 px-3 py-2"
        >
          <label
            class="flex items-center gap-2 text-sm text-gray-800 cursor-pointer"
            :data-testid="`create-task-field-setting-${key}`"
          >
            <input
              v-model="draft[key]"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
            >
            <span>{{ labels[key] || key }}</span>
          </label>
          <div v-if="key === 'task_kind' && draft.task_kind" class="mt-2 ml-6">
            <p class="text-xs text-gray-500 mb-1">每行一个可选值；留空保存将恢复默认 bug-fix / feature。</p>
            <textarea
              v-model="taskKindDraftText"
              rows="4"
              class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary font-mono text-sm"
              data-testid="task-kind-options-textarea"
              placeholder="bug-fix&#10;feature"
            />
          </div>
          <div v-if="key === 'code_lang' && draft.code_lang" class="mt-2 ml-6">
            <p class="text-xs text-gray-500 mb-1">每行一个可选值；留空保存将恢复默认 go / rust / js。</p>
            <textarea
              v-model="codeLangDraftText"
              rows="4"
              class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary font-mono text-sm"
              data-testid="code-lang-options-textarea"
              placeholder="go&#10;rust&#10;js"
            />
          </div>
        </div>
      </div>
      <p
        v-if="errorMessage"
        class="mt-2 text-sm text-red-600 shrink-0"
        data-testid="create-task-field-settings-error"
        :data-traceId="errorTraceId || undefined"
      >
        {{ errorMessage }}
      </p>
      <div class="mt-4 flex justify-end space-x-3 shrink-0">
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
          data-testid="create-task-field-settings-save"
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
import { reactive, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import {
  CREATE_TASK_FIELD_SETTING_KEYS,
  CREATE_TASK_FIELD_SETTING_LABELS,
  createTaskFieldSettingsFromResponse,
  defaultCreateTaskFieldSettings,
  normalizeCreateTaskFieldSettings,
} from '../utils/createTaskFieldSettings.js'
import {
  DEFAULT_CODE_LANG_OPTIONS,
  normalizeCodeLangOptions,
  codeLangOptionsFromResponse,
} from '../utils/codeLangOptions.js'
import {
  DEFAULT_TASK_KIND_OPTIONS,
  normalizeTaskKindOptions,
  taskKindOptionsFromResponse,
} from '../utils/taskKindOptions.js'
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

const fieldKeys = CREATE_TASK_FIELD_SETTING_KEYS
const labels = CREATE_TASK_FIELD_SETTING_LABELS
const draft = reactive(defaultCreateTaskFieldSettings())
const taskKindDraftText = ref(DEFAULT_TASK_KIND_OPTIONS.join('\n'))
const codeLangDraftText = ref(DEFAULT_CODE_LANG_OPTIONS.join('\n'))
const saving = ref(false)
const saveGuard = createClickGuard()
const errorMessage = ref('')
const errorTraceId = ref('')

const applyDraft = (fields) => {
  const normalized = normalizeCreateTaskFieldSettings(fields)
  for (const key of CREATE_TASK_FIELD_SETTING_KEYS) {
    draft[key] = normalized[key]
  }
}

const linesToOptions = (text) =>
  String(text || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)

const loadAll = async () => {
  errorMessage.value = ''
  errorTraceId.value = ''
  applyDraft(defaultCreateTaskFieldSettings())
  taskKindDraftText.value = DEFAULT_TASK_KIND_OPTIONS.join('\n')
  codeLangDraftText.value = DEFAULT_CODE_LANG_OPTIONS.join('\n')
  const tenantId = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  if (!tenantId || !wid) return

  const getJson = async (path) => {
    const response = await apiFetch(path, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const body = await response.json().catch(() => ({}))
    return { response, body }
  }

  try {
    const base = `/api/projects/workspaces/tenant_id/${tenantId}/${wid}`
    const [fieldsRes, kindRes, langRes] = await Promise.all([
      getJson(`${base}/create-task-field-settings/`),
      getJson(`${base}/task-kind-options/`),
      getJson(`${base}/code-lang-options/`),
    ])

    if (!fieldsRes.response.ok) {
      errorMessage.value = fieldsRes.body?.error || fieldsRes.body?.message || `加载字段设置失败 (${fieldsRes.response.status})`
      errorTraceId.value = extractTraceId(fieldsRes.response) || extractTraceId(fieldsRes.body) || ''
      return
    }
    applyDraft(createTaskFieldSettingsFromResponse(fieldsRes.body))

    if (kindRes.response.ok) {
      taskKindDraftText.value = taskKindOptionsFromResponse(kindRes.body).join('\n')
    }
    if (langRes.response.ok) {
      codeLangDraftText.value = codeLangOptionsFromResponse(langRes.body).join('\n')
    }
  } catch (error) {
    errorMessage.value = error?.message || '加载创建任务字段设置失败'
  }
}

watch(
  () => [props.show, props.workspaceId, props.tenantId],
  ([show]) => {
    if (show) void loadAll()
  },
  { immediate: true },
)

const putJson = async (path, body, idempotencyKey) => {
  const response = await apiFetch(path, {
    method: 'PUT',
    credentials: 'include',
    headers: mergeIdempotencyHeaders(
      { Accept: 'application/json', 'Content-Type': 'application/json' },
      idempotencyKey,
    ),
    body: JSON.stringify(body),
  })
  const parsed = await response.json().catch(() => ({}))
  return { response, body: parsed }
}

const save = async () => {
  const tenantId = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  if (!tenantId || !wid) return
  // OPT-20260819-038: 保存字段设置是配置写操作（同批 3 个 PUT 共用同一幂等键），
  // 防连点/超时重试双发
  await saveGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    try {
      const fields = normalizeCreateTaskFieldSettings(draft)
      const taskKindOptions = normalizeTaskKindOptions(linesToOptions(taskKindDraftText.value))
      const codeLangOptions = normalizeCodeLangOptions(linesToOptions(codeLangDraftText.value))
      const base = `/api/projects/workspaces/tenant_id/${tenantId}/${wid}`

      const results = await Promise.all([
        putJson(`${base}/create-task-field-settings/`, { fields }, idempotencyKey),
        putJson(`${base}/task-kind-options/`, { options: taskKindOptions }, idempotencyKey),
        putJson(`${base}/code-lang-options/`, { options: codeLangOptions }, idempotencyKey),
      ])

      for (const { response, body } of results) {
        if (!response.ok) {
          errorMessage.value = body?.error || body?.message || `保存失败 (${response.status})`
          errorTraceId.value = extractTraceId(response) || extractTraceId(body) || ''
          return
        }
      }

      const saved = createTaskFieldSettingsFromResponse(results[0].body)
      emit('saved', {
        fields: saved,
        taskKindOptions,
        codeLangOptions,
      })
      emit('close')
    } catch (error) {
      errorMessage.value = error?.message || '保存创建任务字段设置失败'
      showRequestError(errorMessage.value, error)
    } finally {
      saving.value = false
    }
  })
}
</script>
