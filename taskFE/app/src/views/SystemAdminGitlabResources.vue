<template>
  <div class="p-8" data-alias="view-system-admin-gitlab-resources">
    <!-- 云服务商选项（新建/编辑表单共用；编辑在父级、新建在子组件，datalist id 全局唯一） -->
    <datalist id="cloud-provider-options">
      <option value="tencent" />
      <option value="aws" />
      <option value="azure" />
      <option value="aliyun" />
      <option value="huawei" />
      <option value="gcp" />
    </datalist>
    <div class="flex flex-col space-y-8 max-w-6xl">
      <div>
        <h2 class="text-2xl font-bold text-text">GitLab 资源管理</h2>
        <p class="text-text-light mt-1">管理各区域 GitLab 实例的名称、域名、密钥、磁盘/流量/带宽容量（含是否带宽共享），查看租户配额及开通状态；区域卡片展示服务进程、配置文件与 GITLAB_HOME 路径便于运维调整。</p>
      </div>

      <!-- 全局错误/成功消息 -->
      <p v-if="errorMsg" class="text-sm text-danger" data-testid="gitlab-admin-error" :data-traceId="errorTraceId || undefined">{{ errorMsg }}</p>
      <p v-if="successMsg" class="text-sm text-success" data-testid="gitlab-admin-success">{{ successMsg }}</p>

      <!-- 区域管理 — 标题栏 + 新建按钮 -->
      <section class="bg-white p-6 rounded-xl shadow space-y-5" data-testid="gitlab-region-capacity">
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold text-text">区域管理与配置</h3>
          <button v-if="!showCreateForm"
                  class="px-4 py-2 bg-primary text-white rounded-lg text-sm hover:bg-primary/90"
                  @click="openCreateForm">
            + 新增区域
          </button>
        </div>

        <!-- 新建区域表单（子组件） -->
        <SystemAdminGitlabRegionForm v-if="showCreateForm"
                                     :creating="creatingRegion"
                                     @submit="createRegion"
                                     @cancel="cancelCreateForm" />

        <!-- 区域卡片列表 -->
        <p v-if="regionsLoading" class="text-sm text-text-light">加载中...</p>
        <template v-else>
          <div v-if="regions.length === 0" class="text-sm text-text-light">暂无区域，点击「新增区域」开始添加。</div>
          <div v-for="r in regions" :key="r.slug"
               :class="['border rounded-lg p-4 space-y-3 transition-colors', r.is_active ? 'border-border' : 'border-gray-200 bg-gray-50']">
            <!-- 头部：名称 + Slug + URL + 状态 + 操作按钮 -->
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-text">{{ r.name }}</span>
                <code class="text-xs bg-gray-100 px-2 py-0.5 rounded">{{ r.slug }}</code>
                <span
                  class="text-xs px-2 py-0.5 rounded"
                  data-testid="gitlab-region-access-mode"
                  :class="regionAccessMode(r) === 'development' ? 'bg-amber-100 text-amber-800' : 'bg-slate-100 text-slate-700'"
                >
                  {{ regionAccessMode(r) === 'development' ? '开发模式' : '发布模式' }}
                </span>
                <span
                  v-if="isGitlabRegionPendingNode(r)"
                  class="text-xs px-2 py-0.5 rounded bg-amber-100 text-amber-800"
                  data-testid="gitlab-region-pending-node"
                >待创建节点</span>
                <a v-if="r.gitlab_web_url" :href="r.gitlab_web_url" target="_blank"
                   class="text-xs text-primary underline">{{ r.gitlab_web_url }}</a>
              </div>
              <div class="flex items-center gap-2">
                <span :class="r.is_active ? 'text-success' : 'text-text-light'" class="text-xs">
                  {{ r.is_active ? '● 启用' : '○ 停用' }}
                </span>
                <button v-if="!editingRegion[r.slug]"
                        class="px-2 py-1 border border-border rounded text-xs hover:bg-gray-100"
                        @click="startEdit(r)">编辑</button>
                <button class="px-2 py-1 border border-red-200 rounded text-xs text-red-600 hover:bg-red-50"
                        @click="confirmDelete(r)">删除</button>
              </div>
            </div>

            <!-- API 信息 -->
            <div v-if="!editingRegion[r.slug]" class="text-xs text-text-light flex gap-4 flex-wrap">
              <span v-if="r.cloud_provider">☁️ {{ r.cloud_provider }}</span>
              <span v-if="r.gitlab_api_base">API: <code class="bg-gray-100 px-1 rounded">{{ r.gitlab_api_base }}</code></span>
              <span v-if="r.admin_private_token">🔑 Token 已配置</span>
            </div>

            <!-- 部署路径：服务进程 / 配置文件 / GITLAB_HOME -->
            <SystemAdminGitlabRegionDeployPaths
              :service-process="r.service_process"
              :service-start="r.service_start"
              :config-file="r.config_file"
              :data-dir="r.data_dir"
              :container-name="r.container_name"
              :config-exists="r.config_exists"
            />

            <!-- 编辑模式 -->
            <div v-if="editingRegion[r.slug]" class="space-y-3 border-t border-gray-200 pt-3">
              <h4 class="text-sm font-semibold text-text">编辑区域配置</h4>
              <div class="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <label class="block text-xs text-text-light mb-1">云服务商 <span class="text-danger">*</span></label>
                  <input v-model="editForm[r.slug].cloud_provider" type="text" placeholder="如：tencent / aws / azure"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm"
                         list="cloud-provider-options" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">区域名称</label>
                  <input v-model="editForm[r.slug].name" type="text"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">标识符 (创建后不可修改)</label>
                  <input :value="r.slug" disabled type="text"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm bg-gray-100 text-text-light" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">描述</label>
                  <input v-model="editForm[r.slug].description" type="text"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">排序</label>
                  <input v-model.number="editForm[r.slug].sort_order" type="number" min="0"
                         class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">GitLab API 地址</label>
                  <input v-model="editForm[r.slug].gitlab_api_base" type="text"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">GitLab Web 地址</label>
                  <input v-model="editForm[r.slug].gitlab_web_url" type="text"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm" />
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">Admin Private Token</label>
                  <input v-model="editForm[r.slug].admin_private_token" type="password"
                         placeholder="留空则不修改"
                         class="w-full px-2 py-1.5 border border-border rounded text-sm" />
                </div>
                <div class="flex items-center gap-2 pt-5">
                  <label class="text-xs text-text-light select-none cursor-pointer flex items-center gap-1">
                    <input v-model="editForm[r.slug].is_active" type="checkbox" class="rounded" />
                    启用此区域
                  </label>
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">区域模式</label>
                  <select
                    v-model="editForm[r.slug].access_mode"
                    aria-label="区域模式"
                    data-testid="gitlab-region-access-mode-select"
                    class="w-full px-2 py-1.5 border border-border rounded text-sm"
                  >
                    <option value="release">发布模式</option>
                    <option value="development">开发模式（仅测试角色可用）</option>
                  </select>
                  <p class="text-xs text-text-light mt-1">开发模式仅测试角色账号可在 SaaS 侧选择与购买该区域。</p>
                </div>
                <div>
                  <label class="block text-xs text-text-light mb-1">节点状态</label>
                  <select
                    v-model="editForm[r.slug].infra_status"
                    aria-label="节点状态"
                    data-testid="gitlab-region-infra-status-select"
                    class="w-full px-2 py-1.5 border border-border rounded text-sm"
                  >
                    <option value="ready">已就绪</option>
                    <option value="pending_node">待创建节点</option>
                  </select>
                </div>
              </div>
              <div class="flex gap-2">
                <button class="px-3 py-1.5 bg-primary text-white rounded text-sm hover:bg-primary/90 disabled:opacity-60"
                        :disabled="savingMetadata[r.slug]"
                        @click="saveRegionMetadata(r.slug)">
                  {{ savingMetadata[r.slug] ? '保存中...' : '保存配置' }}
                </button>
                <button class="px-3 py-1.5 border border-border rounded text-sm hover:bg-gray-50"
                        @click="cancelEdit(r)">取消</button>
              </div>
            </div>

            <SystemAdminGitlabRegionCapacity
              :region="r"
              :saving="savingRegion === r.slug"
              @save="saveRegionCapacity"
            />
          </div>
        </template>
      </section>

      <!-- 删除确认弹层：展示仓库地址供管理员确认 -->
      <SystemAdminGitlabRegionDeleteModal
        :target="deleteTarget"
        :submitting="deletingRegion"
        @cancel="deleteTarget = null"
        @confirm="deleteRegion"
      />

      <!-- 租户 GitLab 资源概览（子组件） -->
      <SystemAdminGitlabTenantPanel @error="onTenantError" @success="onTenantSuccess" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import SystemAdminGitlabRegionForm from '../components/system-admin/SystemAdminGitlabRegionForm.vue'
import SystemAdminGitlabRegionDeleteModal from '../components/system-admin/SystemAdminGitlabRegionDeleteModal.vue'
import SystemAdminGitlabRegionDeployPaths from '../components/system-admin/SystemAdminGitlabRegionDeployPaths.vue'
import SystemAdminGitlabRegionCapacity from '../components/system-admin/SystemAdminGitlabRegionCapacity.vue'
import SystemAdminGitlabTenantPanel from '../components/system-admin/SystemAdminGitlabTenantPanel.vue'
import { isGitlabRegionPendingNode } from '../utils/gitlabRegionSelect.js'

const regions = ref([])
const regionsLoading = ref(true)
const errorMsg = ref('')
const errorTraceId = ref('')
const successMsg = ref('')
const savingRegion = ref('')
const savingMetadata = reactive({})
const creatingRegion = ref(false)
const deletingRegion = ref(false)
const deleteTarget = ref(null)
const showCreateForm = ref(false)
const editingRegion = reactive({})
const editForm = reactive({})
const capacityGuard = createClickGuard()

function onTenantError(message, traceId) {
  errorMsg.value = message || '查询失败'
  errorTraceId.value = traceId || ''
}
function onTenantSuccess(message) { successMsg.value = message || '' }

async function loadRegions() {
  regionsLoading.value = true
  errorMsg.value = ''
  errorTraceId.value = ''
  try {
    const resp = await apiFetch('/api/system-admin/gitlab-regions/')
    if (!resp.ok) {
      let detail = ''
      try { const d = await resp.clone().json(); detail = d.error || d.detail || '' } catch {}
      const err = new Error(detail || 'HTTP ' + resp.status)
      err.traceId = resp.traceId
      throw err
    }
    const ct = resp.headers.get('content-type') || ''
    if (!ct.includes('application/json')) {
      let preview = ''
      try { preview = (await resp.clone().text()).replace(/<[^>]*>/g, '').trim().substring(0, 120) } catch {}
      throw new Error(preview || '服务返回了非 JSON 响应，请检查网关路由配置')
    }
    const data = await resp.json()
    regions.value = data.regions || []
  } catch (e) {
    errorMsg.value = e.message || '加载区域失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    regionsLoading.value = false
  }
}

async function saveRegionCapacity(payload) {
  const slug = payload?.slug
  if (!slug) return
  await capacityGuard.run(async ({ idempotencyKey }) => {
    savingRegion.value = slug
    errorMsg.value = ''
    errorTraceId.value = ''
    successMsg.value = ''
    try {
      const resp = await apiFetch('/api/system-admin/gitlab-regions/' + encodeURIComponent(slug) + '/capacity/', {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          region: slug,
          total_disk_gb: Number(payload.total_disk_gb) || 0,
          total_traffic_gb: Number(payload.total_traffic_gb) || 0,
          total_bandwidth_mbps: Number(payload.total_bandwidth_mbps) || 0,
          remaining_bandwidth_mbps: Number(payload.remaining_bandwidth_mbps) || 0,
          bandwidth_shared: !!payload.bandwidth_shared,
        }),
      })
      const ct = resp.headers.get('content-type') || ''
      let data = {}
      if (ct.includes('application/json')) {
        data = await resp.json().catch(() => ({}))
      } else {
        let preview = ''
        try { preview = (await resp.clone().text()).replace(/<[^>]*>/g, '').trim().substring(0, 120) } catch {}
        throw new Error(preview || '服务返回了非 JSON 响应')
      }
      if (!resp.ok) {
        const err = new Error(data.error || data.detail || '保存失败')
        err.traceId = resp.traceId || data.trace_id
        throw err
      }
      successMsg.value = '区域 ' + slug + ' 容量已更新'
      await loadRegions()
    } catch (e) {
      errorMsg.value = e.message || '保存失败'
      errorTraceId.value = e?.traceId || ''
    } finally {
      savingRegion.value = ''
    }
  })
}

// --- Region metadata CRUD ---

function regionAccessMode(r) {
  return r?.access_mode === 'development' ? 'development' : 'release'
}

function startEdit(r) {
  editingRegion[r.slug] = true
  if (!editForm[r.slug]) {
    editForm[r.slug] = {
      name: r.name,
      description: r.description || '',
      sort_order: r.sort_order || 0,
      gitlab_api_base: r.gitlab_api_base || '',
      gitlab_web_url: r.gitlab_web_url || '',
      admin_private_token: '',
      cloud_provider: r.cloud_provider || '',
      is_active: r.is_active,
      access_mode: regionAccessMode(r),
      infra_status: isGitlabRegionPendingNode(r) ? 'pending_node' : 'ready',
    }
  }
}

function cancelEdit(r) {
  editingRegion[r.slug] = false
  delete editForm[r.slug]
}

async function saveRegionMetadata(slug) {
  savingMetadata[slug] = true
  errorMsg.value = ''
  errorTraceId.value = ''
  successMsg.value = ''
  const form = editForm[slug]
  try {
    const payload = {
      name: form.name,
      description: form.description,
      gitlab_api_base: form.gitlab_api_base,
      gitlab_web_url: form.gitlab_web_url,
      cloud_provider: form.cloud_provider.trim(),
      is_active: form.is_active,
      sort_order: Number(form.sort_order) || 0,
      access_mode: form.access_mode === 'development' ? 'development' : 'release',
      infra_status: form.infra_status === 'pending_node' ? 'pending_node' : 'ready',
    }
    if (form.admin_private_token.trim()) {
      payload.admin_private_token = form.admin_private_token.trim()
    }
    const resp = await apiFetch('/api/system-admin/gitlab-regions/' + encodeURIComponent(slug) + '/', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    const ct = resp.headers.get('content-type') || ''
    let data = {}
    if (ct.includes('application/json')) {
      data = await resp.json().catch(() => ({}))
    } else {
      let preview = ''
      try { preview = (await resp.clone().text()).replace(/<[^>]*>/g, '').trim().substring(0, 120) } catch {}
      throw new Error(preview || '服务返回了非 JSON 响应')
    }
    if (!resp.ok) {
      const err = new Error(data.error || data.detail || '保存失败')
      err.traceId = resp.traceId || data.trace_id
      throw err
    }
    successMsg.value = '区域 ' + slug + ' 配置已更新'
    editingRegion[slug] = false
    delete editForm[slug]
    await loadRegions()
  } catch (e) {
    errorMsg.value = e.message || '保存失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    savingMetadata[slug] = false
  }
}

function openCreateForm() {
  showCreateForm.value = true
}

function cancelCreateForm() {
  showCreateForm.value = false
}

async function createRegion(payload) {
  if (!payload.name.trim() || !payload.slug.trim() || !payload.cloud_provider.trim()) {
    errorMsg.value = '区域名称、标识符和云服务商不能为空'
    return
  }
  creatingRegion.value = true
  errorMsg.value = ''
  errorTraceId.value = ''
  successMsg.value = ''
  try {
    const resp = await apiFetch('/api/system-admin/gitlab-regions/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: payload.name.trim(),
        slug: payload.slug.trim(),
        description: payload.description.trim(),
        sort_order: Number(payload.sort_order) || 0,
        gitlab_api_base: payload.gitlab_api_base.trim() || 'http://127.0.0.1:8012',
        gitlab_web_url: payload.gitlab_web_url.trim(),
        admin_private_token: payload.admin_private_token.trim(),
        cloud_provider: payload.cloud_provider.trim(),
        is_active: true,
        total_disk_gb: Number(payload.total_disk_gb) || 0,
        total_traffic_gb: Number(payload.total_traffic_gb) || 0,
        total_bandwidth_mbps: Number(payload.total_bandwidth_mbps) || 0,
        remaining_bandwidth_mbps: Number(payload.remaining_bandwidth_mbps) || 0,
        bandwidth_shared: !!payload.bandwidth_shared,
        access_mode: payload.access_mode === 'development' ? 'development' : 'release',
      }),
    })
    const ct = resp.headers.get('content-type') || ''
    let data = {}
    if (ct.includes('application/json')) {
      data = await resp.json().catch(() => ({}))
    } else {
      let preview = ''
      try { preview = (await resp.clone().text()).replace(/<[^>]*>/g, '').trim().substring(0, 120) } catch {}
      throw new Error(preview || '服务返回了非 JSON 响应')
    }
    if (!resp.ok) {
      const err = new Error(data.error || data.detail || '创建失败')
      err.traceId = resp.traceId || data.trace_id
      throw err
    }
    successMsg.value = '区域 ' + payload.name + ' 已创建'
    showCreateForm.value = false
    await loadRegions()
  } catch (e) {
    errorMsg.value = e.message || '创建失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    creatingRegion.value = false
  }
}

function confirmDelete(r) {
  deleteTarget.value = r
}

async function deleteRegion(slug) {
  deletingRegion.value = true
  errorMsg.value = ''
  errorTraceId.value = ''
  successMsg.value = ''
  try {
    const resp = await apiFetch('/api/system-admin/gitlab-regions/' + encodeURIComponent(slug) + '/', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
    })
    const ct = resp.headers.get('content-type') || ''
    let data = {}
    if (ct.includes('application/json')) {
      data = await resp.json().catch(() => ({}))
    } else {
      let preview = ''
      try { preview = (await resp.clone().text()).replace(/<[^>]*>/g, '').trim().substring(0, 120) } catch {}
      throw new Error(preview || '服务返回了非 JSON 响应')
    }
    if (!resp.ok) {
      const err = new Error(data.error || data.detail || '停用失败')
      err.traceId = resp.traceId || data.trace_id
      throw err
    }
    successMsg.value = '区域 ' + slug + ' 已停用'
    deleteTarget.value = null
    await loadRegions()
  } catch (e) {
    errorMsg.value = e.message || '停用失败'
    errorTraceId.value = e?.traceId || ''
  } finally {
    deletingRegion.value = false
  }
}

onMounted(() => { loadRegions() })
</script>
