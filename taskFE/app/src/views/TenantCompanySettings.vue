<template>
  <div class="p-8">
    <div class="flex flex-col space-y-6">
      <div>
        <h2 class="text-2xl font-bold text-text">公司设置</h2>
        <p class="text-text-light mt-1">
          公司名称即租户对外展示名称；新建租户时默认与注册时一致，可随时由公司管理员修改。
        </p>
      </div>

      <div class="bg-white p-6 rounded-xl shadow max-w-xl">
        <div v-if="loading" class="text-text-light py-8">加载中…</div>
        <template v-else>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-text-light mb-1">公司名称</label>
              <input
                v-model.trim="formName"
                type="text"
                maxlength="255"
                class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                :disabled="!canEdit"
              >
            </div>
            <p v-if="!canEdit" class="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
              您不是该公司管理员，仅可查看名称；如需修改请联系管理员。
            </p>
            <div class="flex gap-3 pt-2">
              <button
                type="button"
                class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
                :disabled="!canEdit || saving || !formName"
                @click="save"
              >
                {{ saving ? '保存中…' : '保存' }}
              </button>
            </div>
          </div>
          <p v-if="successMessage" class="mt-4 text-sm text-green-700">{{ successMessage }}</p>
          <p v-if="errorMessage" class="mt-4 text-sm text-red-600" :data-traceId="errorTraceId || undefined">{{ errorMessage }}</p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import {
  clearStaleLastActiveTenantId,
  resolveMissingCompanyRedirectPath,
} from '../utils/staleTenantRecovery.js'

const route = useRoute()
const router = useRouter()
const tenantId = computed(() => route.params.tenant)

const loading = ref(true)
const saving = ref(false)
const formName = ref('')
const canEdit = ref(false)
const successMessage = ref('')
const errorMessage = ref('')
const errorTraceId = ref('')

// OPT-20260819-038: 保存公司名称是写操作，防连点/超时重试双发 PATCH
const saveGuard = createClickGuard()

/** companies/current 404：清陈旧 lastActiveTenantId，跳转 onboarding 或其它公司设置 */
const recoverFromMissingCompany = async (tid, errDetail, traceId) => {
  clearStaleLastActiveTenantId(tid)
  let companies = []
  try {
    const meRes = await apiFetch('/api/accounts/users/me/', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (meRes.ok) {
      const me = await meRes.json()
      companies = Array.isArray(me.companies) ? me.companies : []
    }
  } catch (_) {
    /* 回退 onboarding */
  }
  const redirectTo = resolveMissingCompanyRedirectPath({
    companies,
    targetPathSuffix: '/settings/company/',
  })
  errorTraceId.value = traceId || ''
  errorMessage.value = errDetail || '该公司不存在或已失效，正在为您跳转…'
  loading.value = false
  await router.replace(redirectTo)
}

const load = async () => {
  loading.value = true
  successMessage.value = ''
  errorMessage.value = ''
  errorTraceId.value = ''
  const tid = tenantId.value
  if (!tid) {
    loading.value = false
    return
  }
  try {
    const companyRes = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/accounts/companies/current/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })

    if (!companyRes.ok) {
      const { data: err, traceId } = await safeResponseJson(companyRes, { fallback: {} })
      if (companyRes.status === 404) {
        await recoverFromMissingCompany(tid, err.detail, traceId)
        return
      }
      errorTraceId.value = traceId
      errorMessage.value = err.detail || `加载公司信息失败（${companyRes.status}）`
      loading.value = false
      return
    }
    const company = await companyRes.json()
    formName.value = company.name || ''

    // Use member_is_admin and member_is_creator from the companies/current endpoint
    // (consistent with Sidebar's permission check — accounts for both admin and creator)
    canEdit.value = !!(company.member_is_admin || company.member_is_creator)
  } catch (e) {
    errorMessage.value = e.message || '加载失败'
    errorTraceId.value = e.traceId || ''
  } finally {
    loading.value = false
  }
}

const save = async () => {
  const tid = tenantId.value
  if (!tid || !formName.value) return
  // OPT-20260819-038: 保存公司名称是写操作，防连点/超时重试双发 PATCH
  await saveGuard.run(async ({ idempotencyKey }) => {
    successMessage.value = ''
    errorMessage.value = ''
    errorTraceId.value = ''
    saving.value = true
    try {
      const response = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/accounts/companies/current/`, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({ name: formName.value }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        errorTraceId.value = traceId
        errorMessage.value = data.detail || Object.values(data).flat().join(' ') || `保存失败（${response.status}）`
        return
      }
      formName.value = data.name || formName.value
      successMessage.value = '已保存'
    } catch (e) {
      errorMessage.value = e.message || '网络错误'
      errorTraceId.value = e.traceId || ''
    } finally {
      saving.value = false
    }
  })
}

watch(
  () => route.params.tenant,
  () => {
    load()
  }
)

onMounted(() => {
  load()
})
</script>
