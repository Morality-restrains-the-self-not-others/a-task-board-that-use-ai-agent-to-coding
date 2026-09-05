<template>
  <div class="mb-4">
    <p
      v-if="fieldError"
      class="text-sm text-red-600 mb-2"
      role="alert"
      data-testid="project-inline-edit-error"
      v-bind="fieldErrorTraceId ? { 'data-traceId': fieldErrorTraceId } : {}"
    >
      {{ fieldError }}
    </p>
    <h2 class="text-xl font-semibold mb-2">已安装镜像</h2>
    <div v-if="!editing">
      <button
        type="button"
        class="w-full text-left rounded-md px-1 -mx-1 py-0.5 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
        data-testid="project-container-image-display"
        title="点击编辑"
        :disabled="saving"
        @click="startEdit"
      >
        <p class="text-gray-900">{{ containerImageDisplay }}</p>
        <span class="sr-only">点击编辑已安装镜像</span>
      </button>
    </div>
    <div v-else data-testid="project-container-image-edit" class="space-y-2">
      <select
        id="project-detail-inline-container-image"
        v-model="draftImageId"
        class="w-full border border-gray-300 rounded-md px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        data-testid="project-container-image-select"
        :disabled="saving"
        @click="emit('request-load-images')"
      >
        <option value="">未设置</option>
        <option v-for="image in installedImages" :key="String(image.id)" :value="String(image.id)">
          {{ image.name }}
          <template v-if="image.version"> ({{ image.version }})</template>
          <template v-if="formatContainerImageArchitectures(image)">
            · 架构 {{ formatContainerImageArchitectures(image) }}
          </template>
          <template v-if="image.hardware_summary"> - {{ image.hardware_summary }}</template>
        </option>
      </select>
      <p v-if="saveBlockedHint" class="text-xs text-amber-700" data-testid="project-image-arch-save-hint">
        {{ saveBlockedHint }}
      </p>
      <div class="flex gap-2">
        <button
          type="button"
          class="text-sm px-3 py-1.5 rounded-md bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-50"
          data-testid="project-container-image-save"
          :disabled="saving"
          :aria-busy="saving ? 'true' : 'false'"
          @click="saveImage"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
        <button
          type="button"
          class="text-sm px-3 py-1.5 rounded-md border border-gray-300 text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          data-testid="project-container-image-cancel"
          :disabled="saving"
          @click="cancelEdit"
        >
          取消
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import {
  formatContainerImageArchitectures,
  formatImageRequiredArch,
  formatInstanceSupportedArch,
  instanceCompatibleWithImageArchitectures,
  resolveImageArchitectures,
} from '../utils/containerImageArchitecture.js'
import { formatInstalledImageDisplayLabel, resolveInstalledImageById } from '../utils/installedImageDisplay.js'
import { coerceSnowflakeId, readJsonPreservingSnowflakeIds } from '../utils/snowflakeId.js'
import { buildClearImagePatchBody, buildImagePatchBody } from '../utils/projectDetailInlineEditUtils.js'
import {
  resolveRunTemplateInstanceType,
  runTemplateIsHardwareComplete,
} from '../utils/projectRunTemplateUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const props = defineProps({
  project: { type: Object, required: true },
  installedImages: { type: Array, default: () => [] },
  referencedContainerImage: { type: Object, default: null },
  tenantId: { type: String, required: true },
  projectId: { type: String, required: true },
  liveRunTemplateGetter: { type: Function, default: null },
  installedImagesLoaded: { type: Boolean, default: false },
})

const emit = defineEmits(['saved', 'request-load-images', 'draft-image-id'])

const editing = ref(false)
const draftImageId = ref('')
const saving = ref(false)
const fieldError = ref('')
const fieldErrorTraceId = ref('')
const saveGuard = createClickGuard()

const containerImageDisplay = computed(() => {
  const imageId = coerceSnowflakeId(props.project?.container_image_id)
  const matched =
    resolveInstalledImageById(props.installedImages, imageId) ||
    (props.referencedContainerImage &&
    coerceSnowflakeId(props.referencedContainerImage?.id) === imageId
      ? props.referencedContainerImage
      : null)
  const rawContainerImage = props.project?.container_image
  const storedName =
    rawContainerImage && typeof rawContainerImage === 'object'
      ? String(rawContainerImage.name || '').trim()
      : String(rawContainerImage || '').trim()
  return formatInstalledImageDisplayLabel({
    image: matched,
    imageId,
    storedName,
    treatMissingBindingAsUnavailable: Boolean(props.installedImagesLoaded && imageId && !matched),
  })
})

const readLiveTemplate = () => {
  if (typeof props.liveRunTemplateGetter === 'function') {
    const live = props.liveRunTemplateGetter()
    if (live && typeof live === 'object') return live
  }
  return props.project?.server_run_template && typeof props.project.server_run_template === 'object'
    ? props.project.server_run_template
    : {}
}

const effectiveTemplateForImage = () => {
  const live = readLiveTemplate()
  const saved =
    props.project?.server_run_template && typeof props.project.server_run_template === 'object'
      ? props.project.server_run_template
      : {}
  if (runTemplateIsHardwareComplete(live)) return live
  if (runTemplateIsHardwareComplete(saved)) return saved
  return live
}

const saveBlockedHint = computed(() => {
  const nextId = String(draftImageId.value || '').trim()
  if (!nextId) return ''
  const currentId = String(props.project?.container_image_id || '').trim()
  if (nextId === currentId) return ''
  const tmpl = effectiveTemplateForImage()
  if (!runTemplateIsHardwareComplete(tmpl)) {
    return '更换镜像须同时配置完整运行模版（云平台、地域与实例）'
  }
  const image = resolveInstalledImageById(props.installedImages, nextId)
  if (!image) return ''
  const inst = resolveRunTemplateInstanceType(tmpl)
  if (!inst) return ''
  if (instanceCompatibleWithImageArchitectures(inst, image)) return ''
  const imageLabel = formatImageRequiredArch(image)
  const instanceLabel = formatInstanceSupportedArch(inst)
  if (!resolveImageArchitectures(image).length) {
    return `无法解析已安装镜像的 CPU 架构（镜像要求: ${imageLabel}；实例系统支持: ${instanceLabel}），拒绝保存以免规格不匹配`
  }
  return `镜像要求的 CPU 架构（${imageLabel}）与实例系统支持的 CPU 架构（${instanceLabel}）不匹配，请在运行模版中改选实例后再保存`
})

watch(draftImageId, (id) => {
  emit('draft-image-id', String(id || '').trim())
})

const startEdit = () => {
  fieldError.value = ''
  fieldErrorTraceId.value = ''
  editing.value = true
  draftImageId.value = coerceSnowflakeId(props.project?.container_image_id) || ''
  emit('request-load-images')
}

const cancelEdit = () => {
  if (saving.value) return
  editing.value = false
  emit('draft-image-id', '')
}

const parseApiError = async (response) => {
  const data = await response.json().catch(() => ({}))
  fieldErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
  if (typeof data?.detail === 'string' && data.detail.trim()) return data.detail
  if (typeof data?.error === 'string' && data.error.trim()) return data.error
  if (typeof data?.message === 'string' && data.message.trim()) return data.message
  return `保存失败（HTTP ${response.status}）`
}

const patchProject = async (body, extraHeaders = {}) => {
  const tid = String(props.tenantId || '').trim()
  const pid = String(props.projectId || '').trim()
  if (!tid || !pid) throw new Error('缺少租户或项目 ID，无法保存')
  const response = await apiFetch(`/api/projects/${pid}/tenant_id/${tid}/`, {
    method: 'PATCH',
    credentials: 'include',
    headers: mergeIdempotencyHeaders(
      { 'Content-Type': 'application/json', Accept: 'application/json' },
      extraHeaders['Idempotency-Key'],
    ),
    body: JSON.stringify(body),
  })
  if (!response.ok) throw new Error(await parseApiError(response))
  if (typeof response.jsonPreservingSnowflakeIds === 'function') {
    return response.jsonPreservingSnowflakeIds()
  }
  return readJsonPreservingSnowflakeIds(response)
}

const saveImage = async () => {
  const outcome = await saveGuard.run(async ({ idempotencyKey }) => {
    fieldError.value = ''
    fieldErrorTraceId.value = ''
    saving.value = true
    try {
      const nextId = String(draftImageId.value || '').trim()
      const live = readLiveTemplate()
      const saved =
        props.project?.server_run_template && typeof props.project.server_run_template === 'object'
          ? props.project.server_run_template
          : {}
      const body = nextId
        ? buildImagePatchBody(nextId, props.installedImages, live, saved)
        : buildClearImagePatchBody(props.project, props.installedImages)
      const data = await patchProject(body, { 'Idempotency-Key': idempotencyKey })
      editing.value = false
      emit('draft-image-id', '')
      emit('saved', data)
    } catch (e) {
      fieldError.value = e?.message || '保存已安装镜像失败'
    } finally {
      saving.value = false
    }
  })
  if (outcome?.skipped) return
}

defineExpose({ fieldError, fieldErrorTraceId })
</script>
