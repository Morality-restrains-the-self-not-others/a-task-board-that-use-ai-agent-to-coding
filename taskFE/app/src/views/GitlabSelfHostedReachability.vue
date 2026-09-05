<template>
  <div class="space-y-3">
    <label class="flex items-start gap-2 text-sm text-text">
      <input
        type="checkbox"
        class="mt-1"
        data-testid="gitlab-intranet-checkbox"
        :checked="intranet"
        @change="onIntranetChange"
      />
      <span>
        这是内网 GitLab（仅专有网络可达；平台无法访问属预期）
      </span>
    </label>

    <div
      v-if="configured"
      data-testid="gitlab-self-hosted-reachability"
      :data-status="displayStatus"
    >
      <p
        v-if="probeErrorMessage"
        class="text-sm text-danger"
        data-testid="gitlab-reachability-error"
        :data-traceId="probeErrorTraceId || undefined"
      >
        {{ probeErrorMessage }}
      </p>
      <p
        v-else-if="displayStatus === 'checking'"
        class="text-sm text-text-light"
        data-testid="gitlab-reachability-checking"
      >
        正在检测 GitLab 是否可达…
      </p>
      <p
        v-else-if="displayStatus === 'skipped_intranet'"
        class="text-sm text-amber-900"
        data-testid="gitlab-reachability-intranet"
      >
        已标记为内网服务。平台控制面无法访问属预期，不视为故障。
      </p>
      <p
        v-else-if="displayStatus === 'reachable'"
        class="text-sm text-success"
        data-testid="gitlab-reachability-ok"
      >
        已连接到 GitLab。
      </p>
      <p
        v-else-if="displayStatus === 'unreachable'"
        class="text-sm text-danger"
        data-testid="gitlab-reachability-down"
      >
        无法连接到该 GitLab。请确认服务器已启动，且平台能访问该地址。
      </p>
      <!-- Anti-Replay-OK: GET reachability probe; no write -->
      <button
        v-if="!intranet"
        type="button"
        class="mt-1 text-sm text-primary underline disabled:opacity-50"
        data-testid="gitlab-reachability-retry"
        :disabled="checking"
        @click="probe"
      >
        重新检测
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'

const props = defineProps({
  tenantId: { type: [String, Number], default: '' },
  configured: { type: Boolean, default: false },
  intranet: { type: Boolean, default: false },
})

const emit = defineEmits(['update:intranet'])

const checking = ref(false)
const probeStatus = ref('')
const probeErrorMessage = ref('')
const probeErrorTraceId = ref('')

const displayStatus = computed(() => {
  if (!props.configured) return ''
  if (props.intranet) return 'skipped_intranet'
  if (checking.value && !probeStatus.value) return 'checking'
  return probeStatus.value || (checking.value ? 'checking' : '')
})

const onIntranetChange = (event) => {
  emit('update:intranet', Boolean(event?.target?.checked))
}

const probe = async () => {
  const tid = String(props.tenantId || '').trim()
  if (!tid || !props.configured || props.intranet) {
    probeStatus.value = ''
    return
  }
  checking.value = true
  probeErrorMessage.value = ''
  probeErrorTraceId.value = ''
  try {
    const response = await apiFetch(
      `/api/git-oauth/tenant-connection/tenant_id/${encodeURIComponent(tid)}/reachability/`,
      { headers: { Accept: 'application/json' } },
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.detail === 'string' ? data.detail : '无法检测 GitLab 连通性')
      err.traceId = response.traceId || extractTraceId(response) || extractTraceId(data) || ''
      throw err
    }
    probeStatus.value = String(data.status || '')
  } catch (e) {
    probeStatus.value = ''
    probeErrorMessage.value = e?.message || '无法检测 GitLab 连通性'
    probeErrorTraceId.value = e?.traceId || ''
  } finally {
    checking.value = false
  }
}

watch(
  () => [props.configured, props.intranet, String(props.tenantId || '')],
  () => {
    if (!props.configured) {
      probeStatus.value = ''
      return
    }
    if (props.intranet) {
      probeStatus.value = 'skipped_intranet'
      return
    }
    probe()
  },
  { immediate: true },
)
</script>
