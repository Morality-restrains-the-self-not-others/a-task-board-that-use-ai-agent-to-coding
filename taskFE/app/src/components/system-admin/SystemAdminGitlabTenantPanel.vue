<template>
  <section class="bg-white p-6 rounded-xl shadow space-y-5" data-testid="gitlab-tenant-resources">
    <h3 class="text-lg font-semibold text-text">租户 GitLab 资源</h3>
    <div class="flex gap-2 items-center">
      <input v-model="tenantFilter" type="text" placeholder="输入租户 ID 查询"
             class="px-3 py-2 border border-border rounded-lg text-sm w-72" />
      <input v-model="regionFilter" type="text" placeholder="区域 slug（必填）"
             class="px-3 py-2 border border-border rounded-lg text-sm w-56"
             data-testid="gitlab-admin-region-filter" />
      <button class="px-4 py-2 bg-primary text-white rounded-lg text-sm hover:bg-primary/90"
              @click="lookupTenant">查询</button>
    </div>

    <p v-if="tenantLoading" class="text-sm text-text-light">查询中...</p>
    <template v-if="tenantData">
      <div class="border border-border rounded-lg p-4 space-y-2">
        <div class="flex justify-between">
          <span class="font-medium text-text">租户 {{ tenantData.tenant_id }}</span>
          <span :class="tenantData.provisioning_status === 'active' ? 'text-success' : 'text-warning'"
                class="text-sm font-medium">
            {{ tenantData.provisioning_status === 'active' ? '✅ 已开通' : '⏳ 待开通' }}
          </span>
        </div>
        <div class="grid grid-cols-3 gap-4 text-sm">
          <div><span class="text-text-light">区域：</span>{{ tenantData.region_name || tenantData.region }}</div>
          <div><span class="text-text-light">磁盘配额：</span>{{ tenantData.disk_gb }} GB</div>
          <div><span class="text-text-light">已用磁盘：</span>{{ tenantData.disk_used_gb }} GB</div>
          <div><span class="text-text-light">流量预购：</span>{{ tenantData.traffic_prepaid_gb }} GB</div>
          <div><span class="text-text-light">已用流量：</span>{{ tenantData.traffic_used_gb }} GB</div>
          <div><span class="text-text-light">到期时间：</span>{{ tenantData.disk_expires_at || '—' }}</div>
        </div>
        <div v-if="tenantData.provisioning_status === 'pending_admin'" class="pt-2 border-t border-gray-100">
          <button class="px-4 py-2 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700 disabled:opacity-60"
                  :disabled="provisioning"
                  @click="provisionTenant(tenantData.tenant_id, tenantData.region)">
            {{ provisioning ? '开通中...' : '开通实施' }}
          </button>
        </div>
      </div>
    </template>
    <p v-else-if="tenantLookedUp" class="text-sm text-text-light">未找到该租户的资源记录</p>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'

const emit = defineEmits(['error', 'success'])

const tenantFilter = ref('')
const regionFilter = ref('')
const tenantLoading = ref(false)
const tenantLookedUp = ref(false)
const tenantData = ref(null)
const provisioning = ref(false)

function reportError(e, fallback) {
  emit('error', (e && e.message) || fallback, (e && e.traceId) || '')
}

async function lookupTenant() {
  const tid = tenantFilter.value.trim()
  const regionSlug = regionFilter.value.trim()
  if (!tid || !regionSlug) {
    emit('error', '须同时填写租户 ID 与区域 slug', '')
    return
  }
  tenantLoading.value = true
  tenantLookedUp.value = true
  tenantData.value = null
  try {
    const resp = await apiFetch('/api/billing/gitlab-resources/tenant_id/' + encodeURIComponent(tid) + '/?region=' + encodeURIComponent(regionSlug))
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
    tenantData.value = await resp.json()
  } catch (e) {
    reportError(e, '查询失败')
  } finally {
    tenantLoading.value = false
  }
}

const provisionTenantGuard = createClickGuard()

async function provisionTenant(tid, region) {
  const regionSlug = String(region || '').trim()
  if (!regionSlug) {
    reportError(new Error('region required'), '开通失败')
    return
  }
  // OPT-20260819-038: 开通是资源写操作，防连点/超时重试双发
  await provisionTenantGuard.run(async ({ idempotencyKey }) => {
    provisioning.value = true
    try {
      const resp = await apiFetch('/api/billing/gitlab-resources/tenant_id/' + encodeURIComponent(tid) + '/provision/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ region: regionSlug }),
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
        const err = new Error(data.detail || data.error || '开通失败')
        err.traceId = resp.traceId
        throw err
      }
      emit('success', '租户 ' + tid + ' 已开通')
      await lookupTenant()
    } catch (e) {
      reportError(e, '开通失败')
    } finally {
      provisioning.value = false
    }
  })
}
</script>
