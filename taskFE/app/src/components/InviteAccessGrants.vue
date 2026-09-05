<template>
  <div class="space-y-3" data-testid="invite-access-grants">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <label class="block text-sm font-medium text-text">页面与区域权限（可选）</label>
        <p class="text-xs text-text-light mt-1">
          受邀人加入后自动获得勾选权限；可快捷授予「访问管理」。
        </p>
      </div>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg border border-border text-sm text-primary hover:bg-primary/5 disabled:opacity-50"
        data-testid="invite-grant-access-mgmt"
        :disabled="disabled || catalogLoading"
        @click="grantAccessManagement"
      >
        授予访问管理
      </button>
    </div>

    <p v-if="disabled" class="text-xs text-amber-700" data-testid="invite-grants-admin-hint">
      管理员入职后拥有全部权限，无需预授。
    </p>

    <div v-if="catalogLoading" class="text-sm text-text-light py-4">加载权限目录…</div>
    <div
      v-else-if="loadError"
      class="text-sm text-red-600"
      :data-traceId="loadErrorTraceId || undefined"
    >
      {{ loadError }}
    </div>
    <div v-else class="max-h-64 overflow-y-auto border border-border rounded-lg p-3" :class="{'opacity-50 pointer-events-none': disabled}">
      <ResourceGrantMatrix
        :pages="catalogPages"
        :model-value="selectedGrants"
        :disabled="disabled"
        @update:model-value="onMatrixUpdate"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import ResourceGrantMatrix from './ResourceGrantMatrix.vue'
import {
  grantsMapToList,
  togglePageGrants,
} from '../domain/auth/resourceGrantEffects.js'

const props = defineProps({
  companyId: { type: String, required: true },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['update:grants'])

const catalogPages = ref([])
const catalogLoading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
/** @type {import('vue').Ref<Record<string, 'view'|'operate'>>} */
const selectedGrants = ref({})

const grantsList = computed(() => grantsMapToList(selectedGrants.value))

watch(
  grantsList,
  (list) => {
    emit('update:grants', props.disabled ? [] : list)
  },
  { deep: true, immediate: true },
)

watch(
  () => props.disabled,
  (d) => {
    if (d) emit('update:grants', [])
    else emit('update:grants', grantsMapToList(selectedGrants.value))
  },
)

/** 共享矩阵更新（view/operate 效果映射） */
function onMatrixUpdate(next) {
  selectedGrants.value = next
}

/** 快捷：授予访问管理整页（page + 全部 region operate） */
function grantAccessManagement() {
  const page = catalogPages.value.find((p) => p.group_key === 'people.access')
  if (!page) return
  selectedGrants.value = togglePageGrants(selectedGrants.value, page, true)
}

async function loadCatalog() {
  catalogLoading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const resp = await apiFetch(`/api/auth/resource-groups/?company_id=${encodeURIComponent(props.companyId)}`, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const { data, traceId } = await safeResponseJson(resp, { fallback: {} })
    if (!resp.ok) {
      loadError.value = data?.detail || data?.error || `目录加载失败 (${resp.status})`
      loadErrorTraceId.value = traceId || ''
      return
    }
    catalogPages.value = Array.isArray(data?.pages) ? data.pages : Array.isArray(data) ? data : []
  } catch (e) {
    loadError.value = e?.message || '目录加载失败'
  } finally {
    catalogLoading.value = false
  }
}

onMounted(loadCatalog)

defineExpose({ grantAccessManagement, selectedGrants, loadCatalog })
</script>
