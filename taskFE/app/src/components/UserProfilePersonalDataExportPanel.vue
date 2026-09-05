<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6">
    <h2 class="text-lg font-semibold text-text mb-2">个人信息导出</h2>
    <p class="text-sm text-text-light mb-4">
      可下载你的账户数据副本（含账号资料、登录方式、账单记录等）。导出文件生成后保留
      {{ retentionDays }} 天，请及时下载；文件仅包含你的个人数据，不包含密码等敏感凭据。
    </p>

    <div v-if="isBusy" class="text-sm text-text-light">{{ busyLabel }}</div>

    <template v-else>
      <div v-if="status === 'ready' || status === 'partial'" class="space-y-3">
        <div v-if="status === 'partial'" class="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
          部分数据源暂不可用（{{ unavailableText }}），已导出的数据均为完整可用的，可稍后重新生成。
        </div>
        <p class="text-sm text-text-light">
          已生成于 <strong>{{ generatedAtLabel }}</strong>，有效期至 <strong>{{ expiresAtLabel }}</strong>
        </p>
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
            @click="downloadExport"
          >
            下载 JSON
          </button>
          <button
            type="button"
            class="px-4 py-2 rounded-lg border border-border text-text hover:bg-gray-50"
            @click="requestExport"
          >
            重新生成
          </button>
        </div>
      </div>

      <div v-else-if="status === 'expired'" class="space-y-3">
        <p class="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
          上次导出的文件已过期，请重新生成后下载。
        </p>
        <button
          type="button"
          class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
          @click="requestExport"
        >
          重新生成导出文件
        </button>
      </div>

      <div v-else class="space-y-3">
        <button
          type="button"
          class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
          @click="requestExport"
        >
          生成导出文件
        </button>
      </div>
    </template>

    <p v-if="inlineMessage" class="text-sm text-success mt-3">{{ inlineMessage }}</p>
    <p v-if="inlineError" class="text-sm text-danger mt-3" :data-traceId="traceId || undefined">{{ inlineError }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { safeResponseJson } from '@/utils/safeResponseJson.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const emit = defineEmits(['message', 'error'])

// OPT-20260819-038: 生成导出是写操作，防连点/超时重试双发 POST
const requestExportGuard = createClickGuard()

const isBusy = ref(false)
const busyLabel = ref('')
const status = ref('none')
const exportId = ref('')
const generatedAt = ref('')
const expiresAt = ref('')
const retentionDays = ref(7)
const unavailable = ref([])
const inlineMessage = ref('')
const inlineError = ref('')
const traceId = ref('')

const generatedAtLabel = computed(() => formatLabel(generatedAt.value))
const expiresAtLabel = computed(() => formatLabel(expiresAt.value))
const unavailableText = computed(() => {
  const map = {
    billing: '账单数据',
    tenant_memberships: '租户成员关系',
    cloud_servers: '云资源记录',
  }
  return unavailable.value.map((s) => map[s] || s).join('、')
})

const formatLabel = (value) => {
  if (!value) return ''
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
}

const loadStatus = async () => {
  const response = await apiFetch('/api/accounts/users/me/personal-data-export/status/', {
    headers: { Accept: 'application/json' },
  })
  const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
  if (!response.ok) {
    traceId.value = tid
    throw new Error(data?.detail || data?.error || '无法获取导出状态')
  }
  status.value = data.status || 'none'
  exportId.value = data.export_id || ''
  generatedAt.value = data.generated_at || ''
  expiresAt.value = data.expires_at || ''
  if (Number(data.retention_days) > 0) {
    retentionDays.value = Number(data.retention_days)
  }
}

const refresh = async () => {
  inlineMessage.value = ''
  inlineError.value = ''
  traceId.value = ''
  try {
    await loadStatus()
  } catch (error) {
    inlineError.value = humanizeRequestErrorMessage(error.message || '加载失败')
    emit('error', inlineError.value)
  }
}

const requestExport = async () => {
  // OPT-20260819-038: 生成导出是写操作，防连点/超时重试双发 POST
  await requestExportGuard.run(async ({ idempotencyKey }) => {
    isBusy.value = true
    busyLabel.value = '正在生成导出文件…'
    inlineMessage.value = ''
    inlineError.value = ''
    traceId.value = ''
    try {
      const response = await apiFetch('/api/accounts/users/me/personal-data-export/request/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
      })
      const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        traceId.value = tid
        throw new Error(data?.detail || data?.error || '生成失败')
      }
      status.value = data.status || 'ready'
      exportId.value = data.export_id || ''
      generatedAt.value = data.generated_at || ''
      expiresAt.value = data.expires_at || ''
      unavailable.value = Array.isArray(data.sections_unavailable) ? data.sections_unavailable : []
      inlineMessage.value = '导出文件已生成'
      emit('message', inlineMessage.value)
    } catch (error) {
      inlineError.value = humanizeRequestErrorMessage(error.message || '生成失败')
      emit('error', inlineError.value)
    } finally {
      isBusy.value = false
    }
  })
}

const downloadExport = async () => {
  isBusy.value = true
  busyLabel.value = '正在下载…'
  inlineMessage.value = ''
  inlineError.value = ''
  traceId.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/me/personal-data-export/download/', {
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) {
      const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
      traceId.value = tid
      if (response.status === 410) {
        status.value = 'expired'
        throw new Error(data?.detail || data?.error || '导出已过期，请重新生成')
      }
      throw new Error(data?.detail || data?.error || '下载失败')
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = exportId.value ? `personal-data-${exportId.value}.json` : 'personal-data-export.json'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    inlineMessage.value = '已开始下载'
    emit('message', inlineMessage.value)
  } catch (error) {
    inlineError.value = humanizeRequestErrorMessage(error.message || '下载失败')
    emit('error', inlineError.value)
  } finally {
    isBusy.value = false
  }
}

onMounted(refresh)
</script>
