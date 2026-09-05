<template>
  <section
    class="bg-white p-6 rounded-xl shadow space-y-4"
    data-testid="gitlab-oidc-sso"
    aria-labelledby="gitlab-oidc-sso-heading"
  >
    <div>
      <h3 id="gitlab-oidc-sso-heading" class="text-lg font-semibold text-text">用平台账号登录自建 GitLab</h3>
      <p class="text-sm text-text-light mt-1">
        让同事用本网站账号打开你们自己的 GitLab 网页。下面的编号和密钥不要填在本页，要写进 GitLab 服务器。
      </p>
    </div>

    <aside
      class="rounded-lg border border-border bg-gray-50 p-4 space-y-2"
      data-testid="gitlab-oidc-sso-where"
      aria-labelledby="gitlab-oidc-sso-where-heading"
    >
      <h4 id="gitlab-oidc-sso-where-heading" class="text-sm font-medium text-text">这些值放到哪里？</h4>
      <p class="text-sm text-text">
        不要填在本网站。请写进你们 GitLab 机器上的
        <code class="font-mono text-xs">/etc/gitlab/gitlab.rb</code>
        （GitLab 自己的配置文件）。不会操作的话，把本段发给管服务器的人即可。
      </p>
      <ol class="list-decimal pl-5 space-y-1 text-sm text-text">
        <li>在本页点「签发 SSO」，记下 client_id、issuer，以及只出现一次的密钥。</li>
        <li>登录安装 GitLab 的那台机器（SSH）。若 GitLab 跑在 Docker 里，先进入该容器再改文件。</li>
        <li>用编辑器打开 <code class="font-mono text-xs">/etc/gitlab/gitlab.rb</code>。</li>
        <li>
          把下方灰色代码整段粘贴进去，并按对应关系填写：
          client_id 对应 <code class="font-mono text-xs">identifier</code>；
          issuer 对应 <code class="font-mono text-xs">issuer</code>；
          密钥对应 <code class="font-mono text-xs">secret</code>（须紧挨 identifier 加一行，默认片段不含密钥）。
        </li>
        <li>保存后执行 <code class="font-mono text-xs">sudo gitlab-ctl reconfigure</code>（大约一两分钟）。</li>
        <li>打开你们的 GitLab 登录页，应出现「Daydaymoney SSO」按钮，用本平台账号点它即可。</li>
      </ol>
      <p class="text-sm text-text-light">
        GitLab 服务器必须能访问 issuer 地址。纯内网、出不去外网则无法登录。
        Helm 安装则把同一段放到 values 的 omniauth.providers，而不是 gitlab.rb。
        <!-- Anti-Replay-OK: 只读外链文档导航 -->
        <a
          class="text-primary underline"
          href="https://docs.gitlab.com/integration/openid_connect_provider/"
          target="_blank"
          rel="noopener noreferrer"
        >GitLab 官方说明</a>
      </p>
    </aside>

    <p
      v-if="ssoHttpWarning"
      class="text-sm text-amber-800"
      data-testid="gitlab-oidc-sso-http-warning"
    >
      {{ ssoHttpWarning }}
    </p>
    <p
      v-if="ssoBlockReason"
      class="text-sm text-danger"
      data-testid="gitlab-oidc-sso-url-blocked"
    >
      {{ ssoBlockReason }}
    </p>
    <p
      v-else-if="errorMessage"
      class="text-sm text-danger"
      data-testid="gitlab-oidc-sso-error"
      :data-traceId="errorTraceId || undefined"
    >
      {{ errorMessage }}
    </p>
    <p v-if="successMessage" class="text-sm text-success" data-testid="gitlab-oidc-sso-success">{{ successMessage }}</p>
    <p v-if="copyFeedback" class="text-sm text-success" data-testid="gitlab-oidc-sso-copy-feedback">{{ copyFeedback }}</p>
    <p v-if="loading" class="text-sm text-text-light">加载 SSO 配置中...</p>

    <template v-else>
      <div class="space-y-1">
        <p class="text-sm">
          client_id：<code class="font-mono text-xs" data-testid="gitlab-oidc-sso-client-id">{{ clientId }}</code>
        </p>
        <p class="text-xs text-text-light" data-testid="gitlab-oidc-sso-client-id-hint">
          写到 gitlab.rb 里 <code class="font-mono">identifier</code> 那一行（客户端编号，用来认出本平台）。
        </p>
      </div>
      <div class="space-y-1">
        <p class="text-sm">
          issuer：<code class="font-mono text-xs" data-testid="gitlab-oidc-sso-issuer">{{ issuer }}</code>
        </p>
        <p class="text-xs text-text-light" data-testid="gitlab-oidc-sso-issuer-hint">
          写到 gitlab.rb 里 <code class="font-mono">issuer</code> 那一行（GitLab 会向这个地址核对登录）。
        </p>
      </div>
      <div v-if="oneTimeSecret" class="space-y-1" data-testid="gitlab-oidc-sso-secret-once">
        <div class="flex items-center gap-2">
          <p class="text-sm font-medium text-text flex-1">client_secret（仅显示一次，请立即保存）</p>
          <button
            type="button"
            class="px-3 py-1 rounded-lg border border-border text-sm hover:bg-gray-50 shrink-0"
            data-testid="gitlab-oidc-sso-secret-copy"
            @click="copySecret"
          >
            复制
          </button>
        </div>
        <code class="block font-mono text-xs break-all bg-gray-50 p-2 rounded">{{ oneTimeSecret }}</code>
        <p class="text-xs text-text-light" data-testid="gitlab-oidc-sso-secret-hint">
          写到 gitlab.rb 的 <code class="font-mono">secret</code> 一行。仅显示一次，刷新或离开本页后无法再看，请立刻粘贴进配置。
        </p>
      </div>
      <div v-if="snippet" class="flex items-center gap-2">
        <p class="text-xs text-text-light flex-1">整段复制到 <code class="font-mono">/etc/gitlab/gitlab.rb</code>：</p>
        <button
          type="button"
          class="px-3 py-1 rounded-lg border border-border text-sm hover:bg-gray-50 shrink-0"
          data-testid="gitlab-oidc-sso-snippet-copy"
          @click="copySnippet"
        >
          复制
        </button>
      </div>
      <pre
        v-if="snippetForDisplay"
        class="text-xs bg-gray-50 p-3 rounded overflow-auto max-h-64"
        data-testid="gitlab-oidc-sso-snippet"
      >{{ snippetForDisplay }}</pre>
      <div class="flex flex-wrap gap-3 pt-1">
        <button
          type="button"
          class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
          data-testid="gitlab-oidc-sso-enable"
          :disabled="!canOperate || enableGuard.isBusy() || enabling || Boolean(ssoBlockReason)"
          :aria-busy="enabling ? 'true' : 'false'"
          @click="onEnable"
        >
          {{ enabling ? '签发中...' : configured ? '已签发' : '签发 SSO' }}
        </button>
        <button
          v-if="configured"
          type="button"
          class="px-4 py-2 rounded-lg border border-border disabled:opacity-60"
          data-testid="gitlab-oidc-sso-rotate"
          :disabled="!canOperate || rotateGuard.isBusy() || rotating"
          :aria-busy="rotating ? 'true' : 'false'"
          @click="onRotate"
        >
          {{ rotating ? '轮换中...' : '轮换密钥' }}
        </button>
        <button
          v-if="configured"
          type="button"
          class="px-4 py-2 rounded-lg border border-danger text-danger disabled:opacity-60"
          data-testid="gitlab-oidc-sso-disable"
          :disabled="!canOperate || disableGuard.isBusy() || disabling"
          :aria-busy="disabling ? 'true' : 'false'"
          @click="onDisable"
        >
          {{ disabling ? '关闭中...' : '关闭 SSO' }}
        </button>
      </div>
    </template>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { usePermissions } from '../composables/usePermissions.js'
import { gitlabOidcSsoBlockReason, gitlabOidcSsoHttpWarning } from '../utils/gitlabOidcSsoHttps.js'
import { insertOmniAuthSecret } from '../utils/gitlabOidcSsoSnippet.js'

const props = defineProps({
  tenantId: { type: String, required: true },
  pathABaseUrl: { type: String, default: '' },
})

const perms = usePermissions()
const loading = ref(true)
const enabling = ref(false)
const rotating = ref(false)
const disabling = ref(false)
const configured = ref(false)
const clientId = ref('')
const issuer = ref('')
const snippet = ref('')
const oneTimeSecret = ref('')
const errorMessage = ref('')
const errorTraceId = ref('')
const successMessage = ref('')
const copyFeedback = ref('')
let copyFeedbackTimer = null

const enableGuard = createClickGuard()
const rotateGuard = createClickGuard()
const disableGuard = createClickGuard()

const apiPath = computed(
  () => `/api/tenant/${encodeURIComponent(props.tenantId)}/gitlab-oidc-sso/`
)
const canOperate = computed(() => perms.hasRegionOperate(props.tenantId, 'settings.gitlab.main'))
const ssoBlockReason = computed(() => gitlabOidcSsoBlockReason(props.pathABaseUrl))
const ssoHttpWarning = computed(() => gitlabOidcSsoHttpWarning(props.pathABaseUrl))
const snippetForDisplay = computed(() => insertOmniAuthSecret(snippet.value, oneTimeSecret.value))

const applyPayload = (data = {}, { keepSecret = false } = {}) => {
  configured.value = Boolean(data.configured)
  clientId.value = String(data.client_id || '')
  issuer.value = String(data.issuer || '')
  snippet.value = String(data.omniauth_snippet || '')
  if (!keepSecret) {
    oneTimeSecret.value = String(data.client_secret || '')
  }
}

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  errorTraceId.value = ''
  try {
    await perms.load(apiFetch)
    if (!props.tenantId) {
      throw new Error('缺少租户 ID')
    }
    const response = await apiFetch(apiPath.value, { headers: { Accept: 'application/json' } })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.detail === 'string' ? data.detail : '无法加载 SSO 配置')
      err.traceId = response.traceId
      throw err
    }
    applyPayload(data)
    oneTimeSecret.value = ''
  } catch (e) {
    errorMessage.value = e?.message || '无法加载 SSO 配置'
    errorTraceId.value = e?.traceId || ''
  } finally {
    loading.value = false
  }
}

const onEnable = () => enableGuard.run(async ({ headers, idempotencyKey }) => {
  enabling.value = true
  errorMessage.value = ''
  successMessage.value = ''
  if (ssoBlockReason.value) {
    enabling.value = false
    return
  }
  try {
    const response = await apiFetch(apiPath.value, {
      method: 'PUT',
      headers: mergeIdempotencyHeaders({
        Accept: 'application/json',
        'Content-Type': 'application/json',
        ...headers,
      }, idempotencyKey),
      body: JSON.stringify({ base_url: String(props.pathABaseUrl || '').trim() }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.detail === 'string' ? data.detail : '签发失败')
      err.traceId = response.traceId
      throw err
    }
    applyPayload(data)
    successMessage.value = data.client_secret ? '已签发，请保存密钥' : 'SSO 已配置'
  } catch (e) {
    errorMessage.value = e?.message || '签发失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    enabling.value = false
  }
})

const onRotate = () => rotateGuard.run(async ({ headers, idempotencyKey }) => {
  rotating.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const response = await apiFetch(`${apiPath.value}rotate/`, {
      method: 'POST',
      headers: mergeIdempotencyHeaders({ Accept: 'application/json', ...headers }, idempotencyKey),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.detail === 'string' ? data.detail : '轮换失败')
      err.traceId = response.traceId
      throw err
    }
    applyPayload(data)
    successMessage.value = '密钥已轮换，请保存新密钥'
  } catch (e) {
    errorMessage.value = e?.message || '轮换失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    rotating.value = false
  }
})

// Anti-Replay-OK: 只读剪贴板（navigator.clipboard.writeText），无写 API。
const copySecret = async () => {
  const text = oneTimeSecret.value
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copyFeedback.value = 'client_secret 已复制'
  } catch {
    copyFeedback.value = '复制失败，请手动选择文本'
  }
  scheduleClearCopyFeedback()
}

// Anti-Replay-OK: 只读剪贴板（navigator.clipboard.writeText），无写 API。
const copySnippet = async () => {
  const text = snippetForDisplay.value
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copyFeedback.value = 'gitlab.rb 片段已复制'
  } catch {
    copyFeedback.value = '复制失败，请手动选择文本'
  }
  scheduleClearCopyFeedback()
}

const scheduleClearCopyFeedback = () => {
  if (copyFeedbackTimer) clearTimeout(copyFeedbackTimer)
  copyFeedbackTimer = setTimeout(() => {
    copyFeedback.value = ''
  }, 2500)
}

const onDisable = () => disableGuard.run(async ({ headers, idempotencyKey }) => {
  disabling.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const response = await apiFetch(apiPath.value, {
      method: 'DELETE',
      headers: mergeIdempotencyHeaders({ Accept: 'application/json', ...headers }, idempotencyKey),
    })
    if (!response.ok && response.status !== 204) {
      const data = await response.json().catch(() => ({}))
      const err = new Error(typeof data.detail === 'string' ? data.detail : '关闭失败')
      err.traceId = response.traceId
      throw err
    }
    configured.value = false
    oneTimeSecret.value = ''
    successMessage.value = '已关闭 SSO'
  } catch (e) {
    errorMessage.value = e?.message || '关闭失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    disabling.value = false
  }
})

watch(() => props.tenantId, () => { load() })
onMounted(load)
</script>
