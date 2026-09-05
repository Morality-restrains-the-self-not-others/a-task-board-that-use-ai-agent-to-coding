<template>
  <div>
    <p
      v-if="errorMessage"
      class="text-sm text-danger"
      data-testid="gitlab-same-vpc-hint-error"
      :data-traceId="errorTraceId || undefined"
    >
      {{ errorMessage }}
    </p>
    <aside
      v-else-if="visible"
      class="rounded-lg border border-amber-200 bg-amber-50 p-4 space-y-3"
      data-testid="gitlab-same-vpc-hint"
      role="note"
      aria-labelledby="gitlab-same-vpc-hint-heading"
    >
      <h4 id="gitlab-same-vpc-hint-heading" class="text-sm font-semibold text-text">
        将 GitLab 与任务机器放在同一专有网络
      </h4>
      <p class="text-sm text-text-light">
        已检测到本租户的默认机器节点。请创建专有网络和交换机，并把自建 GitLab 部署在与任务机器相同的专有网络中，以便任务节点走内网访问、避免占用公网 Git 流量。
      </p>
      <dl v-if="summary" class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sm">
        <div>
          <dt class="text-text-light">默认机器地域</dt>
          <dd class="font-mono text-text" data-testid="gitlab-same-vpc-hint-region">
            {{ summary.region || '—' }}
          </dd>
        </div>
        <div>
          <dt class="text-text-light">专有网络 ID</dt>
          <dd class="font-mono text-text" data-testid="gitlab-same-vpc-hint-vpc">
            {{ displayVpcId || '尚未指定' }}
          </dd>
        </div>
        <div>
          <dt class="text-text-light">交换机 ID</dt>
          <dd class="font-mono text-text" data-testid="gitlab-same-vpc-hint-vswitch">
            {{ summary.vswitch_id || '尚未指定' }}
          </dd>
        </div>
      </dl>
      <p v-if="summary && !displayVpcId" class="text-sm text-amber-900">
        默认机器尚未指定专有网络。请先创建专有网络和交换机，再在「工作空间管理 → 机器节点」写入默认启动配置。
      </p>
      <p v-if="writebackMessage" class="text-sm text-success" data-testid="gitlab-same-vpc-writeback-message">
        {{ writebackMessage }}
      </p>
      <div class="flex flex-wrap gap-2 items-center pt-1">
        <!-- Anti-Replay-OK: opens existing create modal; write happens inside modal submit -->
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:opacity-50"
          data-testid="gitlab-same-vpc-create-vpc"
          :disabled="!canCreateVpc"
          @click="showCreateVpcModal = true"
        >
          创建专有网络
        </button>
        <!-- Anti-Replay-OK: opens existing create modal; requires VPC first -->
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg border border-border text-sm hover:bg-white disabled:opacity-50"
          data-testid="gitlab-same-vpc-create-vswitch"
          :disabled="!canCreateVswitch"
          @click="showCreateVswitchModal = true"
        >
          创建交换机
        </button>
        <!-- Anti-Replay-OK: real navigation link to workspace machine policy -->
        <a
          :href="taskPanelHref"
          class="text-sm text-primary underline"
          data-testid="gitlab-same-vpc-machine-settings-link"
        >
          去工作空间管理配置机器节点
        </a>
        <!-- Anti-Replay-OK: only writes vpc_id/vswitch_id onto the existing default
             machine config row when the operator explicitly opts in after creation -->
        <button
          v-if="canWriteback"
          type="button"
          class="px-3 py-1.5 rounded-lg border border-primary text-primary text-sm hover:bg-primary/10 disabled:opacity-50"
          data-testid="gitlab-same-vpc-writeback"
          :disabled="writebackBusy"
          @click="onWriteback"
        >
          {{ writebackBusy ? '写入中...' : '写入默认机器配置' }}
        </button>
      </div>
    </aside>

    <CreateVpcModal
      :visible="showCreateVpcModal"
      :initial-data="vpcInitialData"
      @close="showCreateVpcModal = false"
      @created="onVpcCreated"
    />
    <CreateVswitchModal
      :visible="showCreateVswitchModal"
      :initial-data="vswitchInitialData"
      @close="showCreateVswitchModal = false"
      @created="onVswitchCreated"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import {
  pickPrimaryDefaultNetwork,
  shouldShowSameVpcHint,
} from '../utils/defaultMachineNetworkHint.js'
import CreateVpcModal from '../components/cloud/CreateVpcModal.vue'
import CreateVswitchModal from '../components/cloud/CreateVswitchModal.vue'

const props = defineProps({
  tenantId: { type: [String, Number], default: '' },
})

const errorMessage = ref('')
const errorTraceId = ref('')
const configs = ref([])
const showCreateVpcModal = ref(false)
const showCreateVswitchModal = ref(false)
const createdVpc = ref({ vpc_id: '', name: '', cidr_block: '' })
const createdVswitch = ref({ vswitch_id: '' })
const writebackBusy = ref(false)
const writebackMessage = ref('')

const visible = computed(() => shouldShowSameVpcHint(configs.value))
const summary = computed(() => pickPrimaryDefaultNetwork(configs.value))
const displayVpcId = computed(
  () => String(createdVpc.value.vpc_id || summary.value?.vpc_id || '').trim()
)
const canCreateVpc = computed(() => {
  const s = summary.value
  return Boolean(s?.authorization_id && s?.platform_type && s?.region)
})
const canCreateVswitch = computed(() => canCreateVpc.value && Boolean(displayVpcId.value))
// OPT-20260826-006：仅在已有默认配置行时允许回写，避免误新建授权。
const canWriteback = computed(() => {
  const s = summary.value
  return Boolean(s?.authorization_id && displayVpcId.value)
})
const taskPanelHref = computed(
  () => `/tenant/${encodeURIComponent(String(props.tenantId || '').trim())}/settings/task-panel/`
)
const vpcInitialData = computed(() => ({
  authorization_id: summary.value?.authorization_id || '',
  platform_type: summary.value?.platform_type || '',
  region: summary.value?.region || '',
}))
const vswitchInitialData = computed(() => ({
  authorization_id: summary.value?.authorization_id || '',
  platform_type: summary.value?.platform_type || '',
  region: summary.value?.region || '',
  vpc_id: displayVpcId.value,
  vpc_name: createdVpc.value.name || displayVpcId.value,
  vpc_cidr_block: createdVpc.value.cidr_block || '',
  zone_id: summary.value?.zone_id || '',
}))

const onVpcCreated = (payload = {}) => {
  createdVpc.value = {
    vpc_id: String(payload.vpc_id || '').trim(),
    name: String(payload.name || '').trim(),
    cidr_block: String(payload.cidr_block || '').trim(),
  }
}

const onVswitchCreated = (payload = {}) => {
  createdVswitch.value = {
    vswitch_id: String(payload?.vswitch_id || '').trim(),
  }
}

// Anti-Replay-OK: 用户在创建后主动点「写入默认机器配置」才 POST；
// 仅把新建的 vpc_id/vswitch_id 写回已有默认配置行，不新建授权。
const onWriteback = async () => {
  const tid = String(props.tenantId || '').trim()
  const s = summary.value
  if (!tid || !s?.authorization_id || !displayVpcId.value) return
  writebackBusy.value = true
  writebackMessage.value = ''
  errorMessage.value = ''
  errorTraceId.value = ''
  try {
    const response = await apiFetch(`/api/cloud/server-config-default/tenant_id/${encodeURIComponent(tid)}/`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      credentials: 'include',
      body: JSON.stringify({
        authorization_id: String(s.authorization_id || ''),
        platform_type: String(s.platform_type || ''),
        region: String(s.region || ''),
        zone_id: String(s.zone_id || ''),
        vpc_id: displayVpcId.value,
        vswitch_id: createdVswitch.value?.vswitch_id || String(s.vswitch_id || ''),
      }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok || data.status === 'error') {
      const err = new Error(typeof data.message === 'string' ? data.message : '写入默认机器配置失败')
      err.traceId = response.traceId || extractTraceId(response) || extractTraceId(data) || ''
      throw err
    }
    writebackMessage.value = '已写入默认机器配置'
    await load()
  } catch (e) {
    errorMessage.value = e?.message || '写入默认机器配置失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    writebackBusy.value = false
  }
}

const load = async () => {
  const tid = String(props.tenantId || '').trim()
  if (!tid) return
  errorMessage.value = ''
  errorTraceId.value = ''
  try {
    const response = await apiFetch(`/api/cloud/server-config-default/tenant_id/${encodeURIComponent(tid)}/`, {
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.message === 'string' ? data.message : '无法加载默认机器节点')
      err.traceId = response.traceId || extractTraceId(response) || ''
      throw err
    }
    configs.value = Array.isArray(data.data) ? data.data : []
  } catch (e) {
    configs.value = []
    errorMessage.value = e?.message || '无法加载默认机器节点'
    errorTraceId.value = e?.traceId || ''
  }
}

onMounted(() => {
  load()
})
</script>
