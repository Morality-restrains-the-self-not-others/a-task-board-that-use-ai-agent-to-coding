<template>
  <div
    v-if="show"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-40"
    data-alias="workspace-machine-policy-modal"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto" @keydown.enter="onEnterKey">
      <h4 class="text-lg font-bold mb-2">机器节点策略</h4>
      <p class="text-sm text-gray-500 mb-4">
        为工作空间「{{ workspaceName }}」配置启用的机器节点、闲置自动回收，以及默认服务器启动配置（供项目「快速应用默认模版」使用）。
      </p>

      <div v-if="loading" class="text-center py-8 text-gray-600">加载中...</div>
      <div v-else class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-2">启用的机器节点（云平台授权）</label>
          <p class="text-xs text-gray-500 mb-2">不勾选表示不限制，可使用租户全部授权。</p>
          <div v-if="authorizations.length === 0" class="text-sm text-gray-500">暂无云平台授权</div>
          <div v-else class="space-y-2 max-h-40 overflow-y-auto border border-gray-200 rounded-md p-2">
            <label
              v-for="auth in authorizations"
              :key="auth.id"
              class="flex items-center gap-2 text-sm"
            >
              <input
                v-model="selectedAuthIds"
                type="checkbox"
                :value="String(auth.id)"
                class="rounded border-gray-300"
              />
              <span>{{ authLabel(auth) }}</span>
            </label>
          </div>
        </div>

        <div data-alias="default-server-config-section">
          <label class="block text-sm font-medium text-gray-700 mb-2">默认服务器启动配置</label>
          <p class="text-xs text-gray-500 mb-2">
            按云平台授权配置启动预设；项目页「快速应用默认模版」下拉将读取这些配置。
          </p>
          <div v-if="authorizations.length === 0" class="text-sm text-amber-700">
            暂无云平台授权，请先在「设置 → 云平台绑定」中添加授权。
          </div>
          <ul v-else class="space-y-2 border border-gray-200 rounded-md p-2">
            <li
              v-for="auth in authorizations"
              :key="`default-cfg-${auth.id}`"
              class="flex items-center justify-between gap-2 text-sm py-1"
            >
              <span class="truncate">{{ authLabel(auth) }}</span>
              <button
                type="button"
                class="shrink-0 px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition-colors"
                data-alias="open-default-server-config"
                @click="openSetDefaultModal(auth)"
              >
                配置默认启动
              </button>
            </li>
          </ul>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">闲置自动回收（分钟）</label>
          <input
            v-model.number="idleRecycleMinutes"
            type="number"
            min="0"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <p class="text-xs text-gray-500 mt-1">0 表示关闭自动回收。</p>
        </div>

        <label class="flex items-center gap-2 text-sm" data-testid="workspace-container-image-at-mode-toggle">
          <input v-model="containerImageAtModeEnabled" type="checkbox" class="rounded border-gray-300" />
          <span>开启容器镜像 @ 模式（评论中 @ 已安装镜像可自动开跑）</span>
        </label>

        <p v-if="error" class="text-sm text-red-600" :data-traceId="errorTraceId || undefined">{{ error }}</p>
      </div>

      <div class="flex justify-end gap-2 mt-6">
        <button type="button" class="btn-secondary" :disabled="saving" @click="$emit('close')">取消</button>
        <button
          type="button"
          class="btn-primary"
          :disabled="loading || saving || saveGuard.isBusy()"
          :aria-busy="saving"
          @click="save"
        >
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>

  <SetDefaultConfigModal
    :visible="setDefaultModalVisible"
    :initial-data="setDefaultForm"
    @close="setDefaultModalVisible = false"
    @config-updated="onDefaultConfigUpdated"
  />
</template>

<script setup>
import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import SetDefaultConfigModal from './cloud/SetDefaultConfigModal.vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  tenantId: { type: [String, Number], default: '' },
  workspaceId: { type: [String, Number], default: '' },
  workspaceName: { type: String, default: '' },
})

const emit = defineEmits(['close', 'saved'])

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const errorTraceId = ref('')
const authorizations = ref([])
const selectedAuthIds = ref([])
const idleRecycleMinutes = ref(5)
const containerImageAtModeEnabled = ref(true)
const setDefaultModalVisible = ref(false)
const setDefaultForm = ref({
  authorization_id: '',
  platform_type: '',
})
const saveGuard = createClickGuard()

const openSetDefaultModal = (authorization) => {
  setDefaultForm.value = {
    authorization_id: authorization.id,
    platform_type: authorization.platform_type,
  }
  setDefaultModalVisible.value = true
}

const onDefaultConfigUpdated = () => {
  setDefaultModalVisible.value = false
}

const authLabel = (auth) => {
  const remark = String(auth.remark || '').trim()
  const pt = String(auth.platform_type || 'aliyun')
  const id = String(auth.id || '')
  return remark ? `${remark} (${pt} · ${id})` : `${pt} · ${id}`
}

const load = async () => {
  const tid = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  if (!tid || !wid) return
  loading.value = true
  error.value = ''
  errorTraceId.value = ''
  try {
    const [policyRes, authRes, wsRes] = await Promise.all([
      apiFetch(
        `/api/cloud/compute/workspace-machine-policy/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}`,
        { credentials: 'include', headers: { Accept: 'application/json' } },
      ),
      apiFetch(`/api/cloud/cloud-platform-authorizations/tenant_id/${encodeURIComponent(tid)}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
      apiFetch(`/api/projects/workspaces/tenant_id/${encodeURIComponent(tid)}/${encodeURIComponent(wid)}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
    ])
    if (!policyRes.ok) {
      throw new Error(`加载策略失败 HTTP ${policyRes.status}`)
    }
    const policy = await policyRes.json()
    idleRecycleMinutes.value = Number(policy.idle_recycle_minutes ?? 5)
    selectedAuthIds.value = Array.isArray(policy.enabled_authorization_ids)
      ? policy.enabled_authorization_ids.map(String)
      : []
    if (wsRes.ok) {
      const ws = await wsRes.json()
      // 默认开启：当后端未返回该字段或值为 null/undefined 时，默认 true
      containerImageAtModeEnabled.value = ws.container_image_at_mode_enabled != null ? !!ws.container_image_at_mode_enabled : true
    } else {
      containerImageAtModeEnabled.value = true
    }
    if (authRes.ok) {
      const authData = await authRes.json()
      const list = Array.isArray(authData)
        ? authData
        : Array.isArray(authData?.results)
          ? authData.results
          : Array.isArray(authData?.authorizations)
            ? authData.authorizations
            : []
      authorizations.value = list
    } else {
      authorizations.value = []
    }
  } catch (e) {
    errorTraceId.value = extractTraceId(e) || ''
    error.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

const save = async () => {
  await saveGuard.run(async ({ idempotencyKey }) => {
    const tid = String(props.tenantId || '').trim()
    const wid = String(props.workspaceId || '').trim()
    if (!tid || !wid) return
    const minutes = Number(idleRecycleMinutes.value)
    if (!Number.isFinite(minutes) || minutes < 0) {
      error.value = '闲置回收分钟数须为非负整数'
      return
    }
    saving.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      const res = await apiFetch(
        `/api/cloud/compute/workspace-machine-policy/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}`,
        {
          method: 'PUT',
          credentials: 'include',
          headers: mergeIdempotencyHeaders(
            { Accept: 'application/json', 'Content-Type': 'application/json' },
            idempotencyKey,
          ),
          body: JSON.stringify({
            idle_recycle_minutes: Math.floor(minutes),
            enabled_authorization_ids: selectedAuthIds.value.map(String),
          }),
        },
      )
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || body.detail || `保存失败 HTTP ${res.status}`)
      }
      const wsPatch = await apiFetch(`/api/projects/workspaces/tenant_id/${encodeURIComponent(tid)}/${encodeURIComponent(wid)}/`, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json', 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ container_image_at_mode_enabled: Boolean(containerImageAtModeEnabled.value) }),
      })
      if (!wsPatch.ok) {
        throw new Error(`保存 @ 模式开关失败 HTTP ${wsPatch.status}`)
      }
      emit('saved')
      emit('close')
    } catch (e) {
      errorTraceId.value = extractTraceId(e) || ''
      error.value = e?.message || '保存失败'
    } finally {
      saving.value = false
    }
  })
}

const onEnterKey = (event) => {
  const tag = (event.target?.tagName || '').toLowerCase()
  if (tag === 'textarea') return
  if (!loading.value && !saving.value && !saveGuard.isBusy()) save()
}

watch(
  () => [props.show, props.workspaceId, props.tenantId],
  ([show]) => {
    if (show) {
      load()
    } else {
      setDefaultModalVisible.value = false
    }
  },
)
</script>
