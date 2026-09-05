<template>
  <div data-testid="project-detail-inline-editable-fields">
    <p
      v-if="fieldError"
      class="text-sm text-red-600 mb-2"
      role="alert"
      data-testid="project-inline-edit-error"
      v-bind="fieldErrorTraceId ? { 'data-traceId': fieldErrorTraceId } : {}"
    >
      {{ fieldError }}
    </p>

    <!-- 项目标签 -->
    <div v-if="field === 'tags'" class="col-span-2">
      <label class="block text-sm font-medium text-gray-700 mb-1">项目标签</label>
      <div v-if="!editing">
        <button
          type="button"
          class="w-full text-left rounded-md px-1 -mx-1 py-0.5 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="project-tags-display"
          title="点击编辑"
          :disabled="saving"
          @click="startEdit"
        >
          <div v-if="projectTagList.length" class="flex flex-wrap gap-2">
            <span
              v-for="tag in projectTagList"
              :key="tag"
              class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-100"
            >
              {{ tag }}
            </span>
          </div>
          <p v-else class="text-gray-500">未设置</p>
          <span class="sr-only">点击编辑项目标签</span>
        </button>
      </div>
      <div v-else data-testid="project-tags-edit" class="space-y-2">
        <ProjectTagsInput
          v-model="draftTags"
          label=""
          hint=""
          input-id="project-detail-inline-tags"
        />
        <button
          type="button"
          class="text-sm px-3 py-1.5 rounded-md border border-gray-300 text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          data-testid="project-tags-sync-daydaymoney"
          :disabled="saving"
          @click="syncTagsFromAidevYaml"
        >
          从 daydaymoney.yaml 同步
        </button>
        <div class="flex gap-2">
          <button
            type="button"
            class="text-sm px-3 py-1.5 rounded-md bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-50"
            data-testid="project-tags-save"
            :disabled="saving"
            @click="saveTags"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
          <button
            type="button"
            class="text-sm px-3 py-1.5 rounded-md border border-gray-300 text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            data-testid="project-tags-cancel"
            :disabled="saving"
            @click="cancelEdit"
          >
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 是否允许自动运行 -->
    <div v-else-if="field === 'auto_run'">
      <label class="block text-sm font-medium text-gray-700 mb-1">是否允许自动运行</label>
      <div v-if="!editing">
        <button
          type="button"
          class="w-full text-left rounded-md px-1 -mx-1 py-0.5 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="project-default-auto-run-display"
          title="点击编辑"
          :disabled="saving"
          @click="startEdit"
        >
          <p
            :class="gitAuthBlocked ? 'text-red-600' : 'text-gray-900'"
            data-testid="project-default-auto-run-label"
          >
            {{ autoRunDisplayLabel }}
          </p>
          <span
            v-if="gitAuthBlocked && gitAuthGateMessage"
            class="block text-xs text-red-600 mt-0.5"
            data-testid="project-auto-run-git-gate-hint"
            :data-traceId="gitAuthGateTraceId || undefined"
          >
            {{ gitAuthGateMessage }}
          </span>
          <span class="sr-only">点击编辑是否允许自动运行</span>
        </button>
      </div>
      <div v-else data-testid="project-default-auto-run-edit" class="space-y-2">
        <label
          class="flex items-start gap-2"
          :class="{ 'opacity-60 cursor-not-allowed': !canEnableAutoRun, 'cursor-pointer': canEnableAutoRun }"
        >
          <input
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            v-model="draftAutoRun"
            :disabled="!canEnableAutoRun || saving"
            data-testid="project-default-auto-run"
          />
          <span class="text-sm">
            <span class="font-medium text-gray-700">是否允许自动运行</span>
            <span v-if="canEnableAutoRun" class="block text-xs text-gray-500 mt-0.5">
              开启后，创建任务允许勾选自动运行并按运行模版启动机器节点；关闭后创建任务将不允许自动启动机器节点
            </span>
            <span v-else class="block text-xs text-gray-500 mt-0.5" data-testid="project-auto-run-disabled-hint">
              {{ autoRunDisabledHint }}
            </span>
          </span>
        </label>
        <div class="flex gap-2">
          <button
            type="button"
            class="text-sm px-3 py-1.5 rounded-md bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-50"
            data-testid="project-default-auto-run-save"
            :disabled="saving || (!canEnableAutoRun && draftAutoRun)"
            @click="saveAutoRun"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
          <button
            type="button"
            class="text-sm px-3 py-1.5 rounded-md border border-gray-300 text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            data-testid="project-default-auto-run-cancel"
            :disabled="saving"
            @click="cancelEdit"
          >
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 已安装镜像 -->
    <ProjectDetailImageField
      v-else-if="field === 'image'"
      :project="project"
      :installed-images="installedImages"
      :referenced-container-image="referencedContainerImage"
      :tenant-id="tenantId"
      :project-id="projectId"
      :live-run-template-getter="liveRunTemplateGetter"
      :installed-images-loaded="installedImagesLoaded"
      @saved="emit('saved', $event)"
      @request-load-images="emit('request-load-images')"
      @draft-image-id="emit('draft-image-id', $event)"
    />
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { normalizeProjectTags } from '../utils/projectTagsUtils.js'
import { mergeDaydaymoneyTags, parseDaydaymoneyYaml } from '../utils/daydaymoneyMeta.js'
import { readJsonPreservingSnowflakeIds } from '../utils/snowflakeId.js'
import {
  buildAutoRunPatchBody,
  buildTagsPatchBody,
  canEnableDefaultAutoRun,
} from '../utils/projectDetailInlineEditUtils.js'
import { resolveDefaultAutoRunDisplayLabel } from '../utils/projectDetailAutoRunGitGate.js'
import { extractTraceId } from '../utils/traceId.js'
import ProjectTagsInput from './ProjectTagsInput.vue'
import ProjectDetailImageField from './ProjectDetailImageField.vue'

const props = defineProps({
  field: {
    type: String,
    required: true,
    validator: (v) => ['tags', 'auto_run', 'image'].includes(v),
  },
  project: { type: Object, required: true },
  installedImages: { type: Array, default: () => [] },
  referencedContainerImage: { type: Object, default: null },
  tenantId: { type: String, required: true },
  projectId: { type: String, required: true },
  gitAuthGate: {
    type: Object,
    default: () => ({ blocked: false, code: '', message: '', displayLabel: '', traceId: '' }),
  },
  liveRunTemplateGetter: { type: Function, default: null },
  installedImagesLoaded: { type: Boolean, default: false },
})

const emit = defineEmits(['saved', 'request-load-images', 'draft-image-id'])

const editing = ref(false)
const draftTags = ref([])
const draftAutoRun = ref(false)
const saving = ref(false)
// OPT-20260819-038: 保存标签/自动运行是写操作，防连点/超时重试双发 PATCH
const saveTagsGuard = createClickGuard()
const saveAutoRunGuard = createClickGuard()
const fieldError = ref('')
const fieldErrorTraceId = ref('')
const autoDisablingForGitGate = ref(false)
/** 已因 Git 门禁发起过关闭，避免父组件尚未回写时重复 PATCH */
const gitGateDisableRequested = ref(false)

const projectTagList = computed(() =>
  normalizeProjectTags(Array.isArray(props.project?.tags) ? props.project.tags : []),
)

const defaultAutoRunEnabled = computed(() =>
  Boolean(props.project?.server_run_template?.default_auto_run),
)

const gitAuthBlocked = computed(() => Boolean(props.gitAuthGate?.blocked))
const gitAuthGateMessage = computed(() => String(props.gitAuthGate?.message || '').trim())
const gitAuthGateTraceId = computed(() => String(props.gitAuthGate?.traceId || '').trim())

const autoRunDisplayLabel = computed(() =>
  resolveDefaultAutoRunDisplayLabel({
    enabled: defaultAutoRunEnabled.value,
    gitAuthBlock: props.gitAuthGate,
  }),
)

const canEnableAutoRun = computed(
  () => canEnableDefaultAutoRun(props.project) && !gitAuthBlocked.value,
)

const autoRunDisabledHint = computed(() => {
  if (gitAuthBlocked.value && gitAuthGateMessage.value) {
    return gitAuthGateMessage.value
  }
  if (!String(props.project?.container_image_id || '').trim()) {
    return '请先选择已安装镜像'
  }
  return '请先配置项目运行模版（云平台、地域与实例）后再启用'
})

watch(canEnableAutoRun, (enabled) => {
  if (!enabled) {
    draftAutoRun.value = false
  }
})

const startEdit = () => {
  fieldError.value = ''
  fieldErrorTraceId.value = ''
  editing.value = true
  if (props.field === 'tags') {
    draftTags.value = [...projectTagList.value]
  } else if (props.field === 'auto_run') {
    draftAutoRun.value = canEnableAutoRun.value ? defaultAutoRunEnabled.value : false
  }
}

const cancelEdit = () => {
  if (saving.value) return
  fieldError.value = ''
  fieldErrorTraceId.value = ''
  editing.value = false
}

const parseApiError = async (response) => {
  const data = await response.json().catch(() => ({}))
  fieldErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
  if (typeof data?.detail === 'string' && data.detail.trim()) return data.detail
  if (typeof data?.error === 'string' && data.error.trim()) return data.error
  return `保存失败（HTTP ${response.status}）`
}

const patchProject = async (body, idempotencyKey) => {
  const tid = String(props.tenantId || '').trim()
  const pid = String(props.projectId || '').trim()
  if (!tid || !pid) {
    throw new Error('缺少租户或项目 ID，无法保存')
  }
  // taskProjectService handleProjectsRoute：projectId 位置段在前、kv 键值对在后
  const response = await apiFetch(`/api/projects/${pid}/tenant_id/${tid}/`, {
    method: 'PATCH',
    credentials: 'include',
    headers: mergeIdempotencyHeaders(
      { 'Content-Type': 'application/json', Accept: 'application/json' },
      idempotencyKey,
    ),
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    throw new Error(await parseApiError(response))
  }
  if (typeof response.jsonPreservingSnowflakeIds === 'function') {
    return response.jsonPreservingSnowflakeIds()
  }
  return readJsonPreservingSnowflakeIds(response)
}

watch(gitAuthBlocked, (blocked) => {
  if (!blocked) {
    gitGateDisableRequested.value = false
  }
})

watch(
  () => ({
    field: props.field,
    blocked: gitAuthBlocked.value,
    enabled: defaultAutoRunEnabled.value,
  }),
  async ({ field, blocked, enabled }) => {
    if (
      field !== 'auto_run'
      || !blocked
      || !enabled
      || gitGateDisableRequested.value
      || autoDisablingForGitGate.value
      || saving.value
    ) {
      return
    }
    gitGateDisableRequested.value = true
    autoDisablingForGitGate.value = true
    fieldError.value = ''
    fieldErrorTraceId.value = ''
    saving.value = true
    try {
      const data = await patchProject(buildAutoRunPatchBody(props.project, false))
      emit('saved', data)
    } catch (e) {
      gitGateDisableRequested.value = false
      fieldError.value = e?.message || '关闭是否允许自动运行失败'
      console.error('[ProjectDetailInlineEditableFields] auto-disable auto_run for git gate failed', e)
    } finally {
      saving.value = false
      autoDisablingForGitGate.value = false
    }
  },
  { immediate: true },
)

const syncTagsFromAidevYaml = () => {
  fieldError.value = ''
  fieldErrorTraceId.value = ''
  const raw = window.prompt('请粘贴 daydaymoney.yaml 内容：')
  if (raw == null) return
  try {
    const meta = parseDaydaymoneyYaml(raw)
    draftTags.value = mergeDaydaymoneyTags(draftTags.value, meta.tags, meta.service_id)
  } catch (e) {
    fieldError.value = e?.message || 'daydaymoney.yaml 解析失败'
    fieldErrorTraceId.value = ''
  }
}

const saveTags = async () => {
  fieldError.value = ''
  fieldErrorTraceId.value = ''
  // OPT-20260819-038: 保存标签是写操作，防连点/超时重试双发 PATCH
  await saveTagsGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    try {
      const data = await patchProject(buildTagsPatchBody(draftTags.value), idempotencyKey)
      editing.value = false
      emit('saved', data)
    } catch (e) {
      fieldError.value = e?.message || '保存标签失败'
      console.error('[ProjectDetailInlineEditableFields] saveTags failed', e)
    } finally {
      saving.value = false
    }
  })
}

const saveAutoRun = async () => {
  fieldError.value = ''
  fieldErrorTraceId.value = ''
  const enabled = Boolean(draftAutoRun.value)
  if (enabled && !canEnableAutoRun.value) {
    fieldError.value = autoRunDisabledHint.value
    fieldErrorTraceId.value = ''
    return
  }
  // OPT-20260819-038: 保存是否允许自动运行是写操作，防连点/超时重试双发 PATCH
  await saveAutoRunGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    try {
      const data = await patchProject(buildAutoRunPatchBody(props.project, enabled), idempotencyKey)
      editing.value = false
      emit('saved', data)
    } catch (e) {
      fieldError.value = e?.message || '保存是否允许自动运行失败'
      console.error('[ProjectDetailInlineEditableFields] saveAutoRun failed', e)
    } finally {
      saving.value = false
    }
  })
}</script>
