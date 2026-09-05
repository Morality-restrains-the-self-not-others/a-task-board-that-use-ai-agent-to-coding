<template>
  <div class="p-8" data-alias="view-system-admin-step-full-cos">
    <div class="max-w-3xl space-y-6">
      <div>
        <div class="flex items-center gap-3 flex-wrap">
          <h2 class="text-2xl font-bold text-text">执行日志 COS</h2>
          <p
            class="text-xs"
            :class="secretConfigured ? 'text-success' : 'text-text-light'"
            data-testid="step-full-cos-secret-status"
            role="status"
          >当前密钥状态：{{ secretStatusKnown ? (secretConfigured ? '已配置' : '未配置') : '未知' }}</p>
        </div>
        <p class="text-text-light mt-1">
          任务详情各层执行完成后，将 step_full.json 归档到腾讯云 COS。评论「启动日志」同时按独立路径规则写入 startup_logs.json。刷新时优先读对象存储，避免容器释放或分片裁剪后无法复查。密钥只保存在本机
          <code>step-full-cos.local.yaml</code>，接口不回显。
        </p>
      </div>
      <p v-if="errorMsg" class="text-sm text-danger" data-testid="step-full-cos-error" :data-traceId="errorTraceId || undefined">{{ errorMsg }}</p>
      <p v-if="successMsg" class="text-sm text-success" data-testid="step-full-cos-success">{{ successMsg }}</p>
      <section class="bg-white p-6 rounded-xl shadow space-y-4">
        <label class="block text-sm text-text">
          后端
          <select v-model="form.backend" class="mt-1 w-full border rounded-md px-3 py-2" data-testid="step-full-cos-backend">
            <option value="local">local（仅库内 JSON，测试/无密钥）</option>
            <option value="cos">cos（腾讯云 COS）</option>
          </select>
        </label>
        <label class="block text-sm text-text">
          Bucket
          <input v-model.trim="form.bucket" type="text" class="mt-1 w-full border rounded-md px-3 py-2" data-testid="step-full-cos-bucket">
        </label>
        <label class="block text-sm text-text">
          Region
          <input v-model.trim="form.region" type="text" class="mt-1 w-full border rounded-md px-3 py-2" placeholder="ap-guangzhou" data-testid="step-full-cos-region">
        </label>
        <label class="block text-sm text-text">
          对象路径规则（step_full）
          <input v-model.trim="form.pathRule" type="text" class="mt-1 w-full border rounded-md px-3 py-2 font-mono text-xs" data-testid="step-full-cos-path-rule">
        </label>
        <label class="block text-sm text-text">
          启动日志路径规则
          <input v-model.trim="form.startupLogsPathRule" type="text" class="mt-1 w-full border rounded-md px-3 py-2 font-mono text-xs" data-testid="step-full-cos-startup-logs-path-rule">
        </label>
        <label class="block text-sm text-text">
          Key 前缀（可选）
          <input v-model.trim="form.keyPrefix" type="text" class="mt-1 w-full border rounded-md px-3 py-2" data-testid="step-full-cos-key-prefix">
        </label>
        <label class="block text-sm text-text">
          SecretId（留空则不改）
          <input v-model="form.secretId" type="password" autocomplete="off" class="mt-1 w-full border rounded-md px-3 py-2" data-testid="step-full-cos-secret-id">
        </label>
        <label class="block text-sm text-text">
          SecretKey（留空则不改）
          <input v-model="form.secretKey" type="password" autocomplete="off" class="mt-1 w-full border rounded-md px-3 py-2" data-testid="step-full-cos-secret-key">
        </label>
        <button
          type="button"
          class="px-4 py-2 bg-primary text-white rounded-lg text-sm hover:bg-primary/90 disabled:opacity-50"
          data-testid="step-full-cos-save"
          :disabled="saving"
          :aria-busy="saving ? 'true' : 'false'"
          @click="save"
        >
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </section>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const form = reactive({
  backend: 'local',
  bucket: '',
  region: 'ap-guangzhou',
  pathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json',
  startupLogsPathRule: 'workspace_{workspaceId}/task_{taskId}/comment_{commentId}/startup_logs.json',
  keyPrefix: '',
  secretId: '',
  secretKey: '',
})
const secretConfigured = ref(false)
/**
 * 密钥状态是否「已知」：仅 GET/PATCH 成功 applyPayload 后置 true。
 * 读取失败（如 403 无 system-admin 权限）时保持 false，状态行展示「未知」而非「未配置」，
 * 避免把「读不到配置」误报成「未配置」。
 */
const secretStatusKnown = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const errorTraceId = ref('')
const successMsg = ref('')
const saveGuard = createClickGuard()

function applyPayload(data) {
  if (!data || typeof data !== 'object') return
  form.backend = data.backend || 'local'
  form.bucket = data.bucket || ''
  form.region = data.region || ''
  form.pathRule = data.pathRule || form.pathRule
  form.startupLogsPathRule = data.startupLogsPathRule || form.startupLogsPathRule
  form.keyPrefix = data.keyPrefix || ''
  secretConfigured.value = Boolean(data.secret_configured)
  secretStatusKnown.value = true
}

async function load() {
  errorMsg.value = ''
  errorTraceId.value = ''
  try {
    const resp = await apiFetch('/api/system-admin/step-full-cos/')
    if (!resp.ok) {
      let detail = ''
      try { const d = await resp.clone().json(); detail = d.error || d.detail || '' } catch {}
      errorMsg.value = detail || ('HTTP ' + resp.status)
      errorTraceId.value = extractTraceId(resp) || ''
      return
    }
    applyPayload(await resp.json())
  } catch (e) {
    errorMsg.value = e?.message || '加载失败'
  }
}

async function save() {
  await saveGuard.run(async (idempotencyKey) => {
    saving.value = true
    errorMsg.value = ''
    errorTraceId.value = ''
    successMsg.value = ''
    try {
      const body = {
        backend: form.backend,
        bucket: form.bucket,
        region: form.region,
        pathRule: form.pathRule,
        startupLogsPathRule: form.startupLogsPathRule,
        keyPrefix: form.keyPrefix,
      }
      if (form.secretId) body.secretId = form.secretId
      if (form.secretKey) body.secretKey = form.secretKey
      const resp = await apiFetch('/api/system-admin/step-full-cos/', {
        method: 'PATCH',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(body),
      })
      if (!resp.ok) {
        let detail = ''
        try { const d = await resp.clone().json(); detail = d.error || d.detail || '' } catch {}
        errorMsg.value = detail || ('HTTP ' + resp.status)
        errorTraceId.value = extractTraceId(resp) || ''
        return
      }
      applyPayload(await resp.json())
      form.secretId = ''
      form.secretKey = ''
      successMsg.value = '已保存'
    } finally {
      saving.value = false
    }
  })
}

onMounted(load)
</script>
