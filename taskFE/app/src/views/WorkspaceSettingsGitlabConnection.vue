<template>
  <div class="p-8" data-alias="view-tenant-gitlab-connection">
    <div class="flex flex-col space-y-8 max-w-3xl">
      <div>
        <h2 class="text-2xl font-bold text-text">GitLab</h2>
        <p class="text-text-light mt-1">
          <template v-if="hasGitlabResource">
            管理本租户系统内建 GitLab 资源配额，并配置自建 GitLab 的 OAuth 连接。
          </template>
          <template v-else>
            配置本租户自建 GitLab 的 OAuth 连接。如需使用系统内建 GitLab，请先选择区域并
            <a
              :href="`/tenant/${tenantId}/billing/orders/create/`"
              class="text-primary underline"
              data-testid="gitlab-purchase-link"
            >购买 GitLab 资源</a>。
          </template>
        </p>
      </div>

      <section class="bg-white p-6 rounded-xl shadow space-y-3" data-testid="gitlab-region-picker">
        <h3 class="text-lg font-semibold text-text">选择 GitLab 区域</h3>
        <p class="text-sm text-text-light">平台无默认区域，须显式选购至少一个区域后才能使用系统内建 GitLab。</p>
        <select
          v-model="region"
          class="w-full max-w-md px-2 py-1.5 border border-border rounded text-sm"
          data-testid="gitlab-region-select"
        >
          <option value="">请选择区域</option>
          <!-- OPT-20260902-013：按云厂商 optgroup 分组，与购买页 OrderCreate 文案一致 -->
          <optgroup v-for="g in regionGroups" :key="g.provider" :label="g.label">
            <option v-for="r in g.regions" :key="r.slug" :value="r.slug">{{ r.name || r.slug }}</option>
          </optgroup>
        </select>
        <p
          v-if="selectedRegionPendingNode"
          class="text-sm text-amber-700 mt-1"
          data-testid="gitlab-region-pending-node-hint"
        >
          该区域节点尚未部署。请先购买 GitLab 资源等待平台开通，开通后自动获得区域访问地址。
        </p>

        <!-- 区块一：系统内建 GitLab 资源配额 — 已选区域或已购买/开通时可见 -->
        <section
          v-if="showGitlabDetails"
          class="mt-4 space-y-5"
          data-testid="gitlab-builtin-resources"
          aria-labelledby="gitlab-builtin-heading"
        >
        <div>
          <h3 id="gitlab-builtin-heading" class="text-lg font-semibold text-text">系统内建 GitLab</h3>
          <p class="text-sm text-text-light mt-1">
            系统内建 GitLab 的磁盘与流量配额使用情况。
          </p>
          <p v-if="displayRegionName" class="text-sm text-text-light mt-1">
            区域：<span class="font-medium text-text" data-testid="gitlab-region-name">{{ displayRegionName }}</span>
            <span v-if="displayGitlabWebUrl" class="ml-2">
              (<a :href="displayGitlabWebUrl" target="_blank" class="text-primary underline" data-testid="gitlab-web-url">{{ displayGitlabWebUrl }}</a>)
            </span>
          </p>
          <p v-if="provisioningStatus === 'pending_admin'" class="text-sm text-warning mt-1" data-testid="gitlab-provisioning-pending">
            ⏳ 等待管理员开通实施
          </p>
        </div>

        <p
          v-if="resErrorMessage"
          class="text-sm text-danger"
          data-testid="gitlab-resources-error"
          :data-traceId="resErrorTraceId || undefined"
        >
          {{ resErrorMessage }}
        </p>
        <p v-if="resSuccessMessage" class="text-sm text-success" data-testid="gitlab-resources-success">
          {{ resSuccessMessage }}
        </p>
        <p v-if="resLoading" class="text-sm text-text-light">加载资源配额中...</p>

        <template v-else>
          <dl class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
            <div>
              <dt class="text-text-light">当前磁盘配额</dt>
              <dd class="font-medium text-text" data-testid="gitlab-current-disk-gb">
                {{ currentDiskGb > 0 ? `${currentDiskGb} GB` : '未购买' }}
              </dd>
            </div>
            <div>
              <dt class="text-text-light">已用磁盘</dt>
              <dd class="font-medium text-text" data-testid="gitlab-disk-used-gb">
                {{ formatUsedGb(diskUsedGb) }} GB
                <span class="text-text-light font-normal">/
                  {{ currentDiskGb > 0 ? `${currentDiskGb} GB` : '未设配额' }}
                </span>
              </dd>
            </div>
            <div>
              <dt class="text-text-light">当前流量预购</dt>
              <dd class="font-medium text-text" data-testid="gitlab-current-traffic-gb">{{ currentTrafficGb }} GB</dd>
            </div>
            <div>
              <dt class="text-text-light">已用流量</dt>
              <dd
                class="font-medium text-text"
                data-testid="gitlab-traffic-used-gb"
                :class="trafficQuotaBlocked ? 'text-danger' : ''"
              >
                {{ formatUsedGb(trafficUsedGb) }} GB
                <span class="text-text-light font-normal">/
                  {{ currentTrafficGb > 0 ? `${currentTrafficGb} GB` : '未预购' }}
                </span>
              </dd>
            </div>
            <div>
              <dt class="text-text-light">已购磁盘时长</dt>
              <dd class="font-medium text-text" data-testid="gitlab-current-disk-months">
                {{ currentDiskMonths > 0 ? `${currentDiskMonths} 个月` : '—' }}
              </dd>
            </div>
            <div>
              <dt class="text-text-light">磁盘到期时间</dt>
              <dd class="font-medium text-text" data-testid="gitlab-disk-expires-at">
                {{ currentDiskExpiresAt || '—' }}
              </dd>
            </div>
            <div>
              <dt class="text-text-light">磁盘单价</dt>
              <dd class="font-medium text-text">{{ formatYuan(diskUnitPrice) }} 元 / GB / 月</dd>
            </div>
            <div>
              <dt class="text-text-light">流量单价</dt>
              <dd class="font-medium text-text">{{ formatYuan(trafficUnitPrice) }} 元 / GB</dd>
            </div>
          </dl>
          <p
            v-if="trafficQuotaBlocked"
            class="text-sm text-danger mt-3"
            data-testid="gitlab-traffic-quota-blocked"
          >
            公网 Git 克隆/拉取已阻断：流量未预购或已用尽（含任务贴启动的服务器节点走公网）。同区域内网与 CI 仍可拉取。
            <!-- Anti-Replay-OK: real navigation link to order create; no write POST -->
            <a
              :href="`/tenant/${tenantId}/billing/orders/create/`"
              class="text-primary underline"
              data-testid="gitlab-traffic-quota-purchase-link"
            >购买流量</a>
          </p>

        </template>
      </section>
      </section>

      <!-- 区块二：自建 GitLab 连接 -->
      <section
        class="bg-white p-6 rounded-xl shadow space-y-5"
        data-testid="gitlab-self-hosted-connection"
        aria-labelledby="gitlab-self-hosted-heading"
      >
        <div>
          <h3 id="gitlab-self-hosted-heading" class="text-lg font-semibold text-text">自建 GitLab 连接</h3>
          <p class="text-sm text-text-light mt-1">
            登记本租户自建 GitLab 的 OAuth Application，供成员在「Git 网站授权」中绑定账号。
          </p>
        </div>

        <GitlabSelfHostedSameVpcHint :tenant-id="tenantId" />

        <p
          v-if="connErrorMessage"
          class="text-sm text-danger"
          data-testid="gitlab-connection-error"
          :data-traceId="connErrorTraceId || undefined"
        >
          {{ connErrorMessage }}
        </p>
        <p v-if="connSuccessMessage" class="text-sm text-success">{{ connSuccessMessage }}</p>
        <p v-if="connLoading" class="text-sm text-text-light">加载连接中...</p>

        <template v-else>
          <div>
            <label class="block text-sm font-medium text-text mb-1">GitLab Base URL</label>
            <input
              v-model="form.base_url"
              type="url"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm"
              placeholder="https://gitlab.daydaymoney.com"
            />
          </div>
          <GitlabSelfHostedReachability
            :tenant-id="tenantId"
            :configured="configured"
            v-model:intranet="form.intranet"
          />
          <div>
            <label class="block text-sm font-medium text-text mb-1">Application ID (client_id)</label>
            <input
              v-model="form.client_id"
              type="text"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm"
              autocomplete="off"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-text mb-1">Secret (client_secret)</label>
            <input
              v-model="form.client_secret"
              type="password"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm"
              :placeholder="configured ? '留空则须重新填写以更新' : ''"
              autocomplete="new-password"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-text mb-1">备注（可选）</label>
            <input
              v-model="form.remark"
              type="text"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm"
              placeholder="展示名称"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-text mb-1">Redirect URI（只读，请粘贴到 GitLab Application）</label>
            <div class="flex gap-2 items-stretch">
              <input
                :value="redirectUri"
                type="text"
                readonly
                data-testid="gitlab-redirect-uri"
                class="flex-1 px-3 py-2 border border-border rounded-lg text-sm bg-gray-50 font-mono"
              />
              <button
                type="button"
                class="px-3 py-2 rounded-lg border border-border text-sm hover:bg-gray-50 shrink-0"
                @click="copyRedirectUri"
              >
                复制
              </button>
            </div>
            <p v-if="providerKey" class="text-xs text-text-light mt-2">
              provider_key：<span class="font-mono">{{ providerKey }}</span>
            </p>
          </div>

          <WorkspaceSettingsGitlabOauthAppHelp :redirect-uri="redirectUri" />

          <div class="flex flex-wrap gap-3 pt-2">
            <button
              type="button"
              class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
              :disabled="saving"
              @click="save"
            >
              {{ saving ? '保存中...' : '保存' }}
            </button>
            <button
              v-if="configured"
              type="button"
              class="px-4 py-2 rounded-lg border border-danger text-danger hover:bg-danger/5 disabled:opacity-60"
              :disabled="deleting"
              @click="remove"
            >
              {{ deleting ? '删除中...' : '删除连接' }}
            </button>
          </div>
        </template>
      </section>

      <WorkspaceSettingsGitlabOidcSso
        :tenant-id="tenantId"
        :path-a-base-url="form.base_url"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { useGitlabResourcePurchase } from '../composables/useGitlabResourcePurchase.js'
import { formatUsedGb } from '../utils/formatUsedGb.js'
import { groupGitlabRegionsByProvider, isGitlabRegionPendingNode } from '../utils/gitlabRegionSelect.js'
import WorkspaceSettingsGitlabOauthAppHelp from './WorkspaceSettingsGitlabOauthAppHelp.vue'
import WorkspaceSettingsGitlabOidcSso from './WorkspaceSettingsGitlabOidcSso.vue'
import GitlabSelfHostedSameVpcHint from './GitlabSelfHostedSameVpcHint.vue'
import GitlabSelfHostedReachability from './GitlabSelfHostedReachability.vue'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || '').trim())

const {
  loading: resLoading,
  errorMessage: resErrorMessage,
  errorTraceId: resErrorTraceId,
  successMessage: resSuccessMessage,
  diskUnitPrice,
  trafficUnitPrice,
  currentDiskGb,
  currentTrafficGb,
  currentDiskMonths,
  currentDiskExpiresAt,
  diskUsedGb,
  trafficUsedGb,
  region,
  regionName,
  availableRegions,
  gitlabWebUrl,
  provisioningStatus,
  load: loadResources,
} = useGitlabResourcePurchase(tenantId)

/** 已购买/开通，或已显式选择区域时展示配额详情（嵌在区域选择器内）。 */
const hasGitlabResource = computed(
  () => provisioningStatus.value === 'pending_admin' || provisioningStatus.value === 'active'
)
const showGitlabDetails = computed(
  () => hasGitlabResource.value || Boolean(String(region.value || '').trim())
)
const trafficQuotaBlocked = computed(() => {
  const prepaid = Number(currentTrafficGb.value) || 0
  const used = Number(trafficUsedGb.value) || 0
  return prepaid <= 0 || used >= prepaid
})

/** 当前下拉选项的元数据；切换选项时立刻更新详情标题/入口，不依赖配额接口返回。 */
const selectedRegionMeta = computed(() => {
  const slug = String(region.value || '').trim()
  if (!slug) return null
  return (availableRegions.value || []).find((r) => String(r.slug) === slug) || null
})
/** OPT-20260902-013：按云厂商分组下拉，与购买页 OrderCreate 共用同一 util。 */
const regionGroups = computed(() => groupGitlabRegionsByProvider(availableRegions.value))
const selectedRegionPendingNode = computed(() => isGitlabRegionPendingNode(selectedRegionMeta.value))
const displayRegionName = computed(() => {
  const meta = selectedRegionMeta.value
  if (meta) return String(meta.name || meta.slug || '')
  return String(regionName.value || '')
})
const displayGitlabWebUrl = computed(() => {
  const meta = selectedRegionMeta.value
  const fromOption = meta ? String(meta.gitlab_web_url || meta.gitlabWebUrl || '').trim() : ''
  if (fromOption) return fromOption
  return String(gitlabWebUrl.value || '')
})

/**
 * 后端单价单位为「分」（cents），前端展示为「元」。
 * @param {number} cents
 * @returns {string}
 */
const formatYuan = (cents) => {
  const v = Number(cents)
  if (!Number.isFinite(v) || v <= 0) return '0.00'
  return (v / 100).toFixed(2)
}

const connLoading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const configured = ref(false)
const redirectUri = ref('')
const providerKey = ref('')
const connErrorMessage = ref('')
const connErrorTraceId = ref('')
const connSuccessMessage = ref('')

const form = reactive({
  base_url: '',
  client_id: '',
  client_secret: '',
  remark: '',
  intranet: false,
})

const apiPath = computed(
  () => `/api/git-oauth/tenant-connection/tenant_id/${encodeURIComponent(tenantId.value)}/`
)

const applyPayload = (data = {}) => {
  configured.value = Boolean(data.configured)
  form.base_url = String(data.base_url || '')
  form.client_id = String(data.client_id || '')
  form.client_secret = ''
  form.remark = String(data.remark || '')
  form.intranet = Boolean(data.intranet)
  redirectUri.value = String(data.redirect_uri || '')
  providerKey.value = String(data.provider_key || '')
}

const loadConnection = async () => {
  connLoading.value = true
  connErrorMessage.value = ''
  connErrorTraceId.value = ''
  try {
    if (!tenantId.value) {
      throw new Error('缺少租户 ID')
    }
    const response = await apiFetch(apiPath.value, { headers: { Accept: 'application/json' } })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.detail === 'string' ? data.detail : '无法加载 GitLab 连接')
      err.traceId = response.traceId
      throw err
    }
    applyPayload(data)
  } catch (e) {
    connErrorMessage.value = e?.message || '无法加载 GitLab 连接'
    connErrorTraceId.value = e?.traceId || ''
  } finally {
    connLoading.value = false
  }
}

const save = async () => {
  saving.value = true
  connErrorMessage.value = ''
  connErrorTraceId.value = ''
  connSuccessMessage.value = ''
  try {
    if (!form.base_url.trim() || !form.client_id.trim() || !form.client_secret.trim()) {
      throw new Error('请填写 base_url、client_id 与 client_secret')
    }
    const response = await apiFetch(apiPath.value, {
      method: 'PUT',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify({
        base_url: form.base_url.trim(),
        client_id: form.client_id.trim(),
        client_secret: form.client_secret,
        remark: form.remark.trim() || undefined,
        intranet: Boolean(form.intranet),
      }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.detail === 'string' ? data.detail : '保存失败')
      err.traceId = response.traceId
      throw err
    }
    applyPayload(data)
    connSuccessMessage.value = '已保存'
  } catch (e) {
    connErrorMessage.value = e?.message || '保存失败'
    connErrorTraceId.value = e?.traceId || ''
  } finally {
    saving.value = false
  }
}

const remove = async () => {
  if (!window.confirm('确定删除本租户的 GitLab OAuth 连接？已绑定的用户凭据将一并清除。')) {
    return
  }
  deleting.value = true
  connErrorMessage.value = ''
  connErrorTraceId.value = ''
  connSuccessMessage.value = ''
  try {
    const response = await apiFetch(apiPath.value, {
      method: 'DELETE',
      headers: { Accept: 'application/json' },
    })
    if (!response.ok && response.status !== 204) {
      const data = await response.json().catch(() => ({}))
      const err = new Error(typeof data.detail === 'string' ? data.detail : '删除失败')
      err.traceId = response.traceId
      throw err
    }
    applyPayload({
      configured: false,
      redirect_uri: redirectUri.value,
      provider_key: providerKey.value,
    })
    connSuccessMessage.value = '已删除'
  } catch (e) {
    connErrorMessage.value = e?.message || '删除失败'
    connErrorTraceId.value = e?.traceId || ''
  } finally {
    deleting.value = false
  }
}

const copyRedirectUri = async () => {
  const text = redirectUri.value
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    connSuccessMessage.value = 'Redirect URI 已复制'
  } catch {
    connErrorMessage.value = '复制失败，请手动选择文本'
  }
}

watch(region, (slug, prev) => {
  if (resLoading.value) return
  if (prev === undefined) return
  if (String(slug || '').trim()) loadResources()
})

onMounted(() => {
  loadResources()
  loadConnection()
})
</script>
