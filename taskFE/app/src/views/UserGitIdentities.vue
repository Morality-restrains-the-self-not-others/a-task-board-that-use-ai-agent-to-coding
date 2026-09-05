<template>
  <div class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="git-identities" class="shrink-0" />
      <main class="flex-1 max-w-3xl space-y-6">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6">
          <h1 class="text-2xl font-semibold text-text mb-2">Git 提交身份</h1>
          <p class="text-sm text-text-light">
            管理各公司下的 git commit 署名（name / email）。同一公司可配置多个身份。
            这与「Git 网站 OAuth」（GitHub/GitLab 登录授权）是两套独立设置。
          </p>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-border p-6">
          <h2 class="text-lg font-semibold text-text mb-4">新建 Git 身份</h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label class="text-sm text-text-light" for="git-identity-company">公司</label>
              <select
                id="git-identity-company"
                v-model="form.company_id"
                class="mt-1 w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none bg-white"
                :disabled="creating || companiesLoading || membershipCompanies.length === 0"
              >
                <option value="" disabled>{{ companiesLoading ? '加载公司列表…' : '请选择公司' }}</option>
                <option
                  v-for="c in membershipCompanies"
                  :key="String(c.company_id)"
                  :value="String(c.company_id)"
                >
                  {{ c.company_name || '未设置' }}
                </option>
              </select>
              <p v-if="membershipCompaniesError" class="mt-1 text-xs text-danger" :data-traceId="membershipCompaniesErrorTraceId || undefined">{{ membershipCompaniesError }}</p>
              <p
                v-else-if="!companiesLoading && membershipCompanies.length === 0"
                class="mt-1 text-xs text-text-light"
              >
                你尚未加入任何公司，无法在此新建 Git 身份。
              </p>
            </div>
            <div>
              <label class="text-sm text-text-light">身份标签（可选）</label>
              <input
                v-model.trim="form.label"
                type="text"
                class="mt-1 w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none"
                placeholder="例如：公司主账号"
              >
            </div>
            <div>
              <label class="text-sm text-text-light">Git 用户名</label>
              <input
                v-model.trim="form.git_user_name"
                type="text"
                class="mt-1 w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none"
                placeholder="例如：alice"
              >
            </div>
            <div>
              <label class="text-sm text-text-light">Git 邮箱</label>
              <input
                v-model.trim="form.git_user_email"
                type="email"
                class="mt-1 w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none"
                placeholder="例如：alice@company.com"
              >
            </div>
            <div class="md:col-span-2">
              <label class="text-sm text-text-light">Git 网站 / 远端用户名（推送鉴权展示）</label>
              <input
                v-model.trim="form.git_remote_username"
                type="text"
                disabled
                class="mt-1 w-full px-3 py-2 border border-border rounded-lg outline-none bg-gray-50 text-text-light cursor-not-allowed"
                placeholder="例如：GitHub 登录名或部署用户"
              >
            </div>
            <div class="md:col-span-2 flex items-center gap-2">
              <input id="identity-default" v-model="form.is_default" type="checkbox" class="w-4 h-4">
              <label for="identity-default" class="text-sm text-text-light">设为该公司的默认身份</label>
            </div>
          </div>
          <div class="mt-4 flex justify-end">
            <button
              type="button"
              :disabled="creating"
              class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-60 disabled:cursor-not-allowed"
              @click="createIdentity"
            >
              {{ creating ? '创建中...' : '新建身份' }}
            </button>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-border p-6">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-semibold text-text">已配置身份</h2>
            <button
              type="button"
              :disabled="loading"
              class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-gray-50 disabled:opacity-60"
              @click="fetchIdentities"
            >
              刷新
            </button>
          </div>
          <p v-if="errorMessage" class="text-sm text-danger mb-3" :data-traceId="errorMessageTraceId || undefined">{{ errorMessage }}</p>
          <p v-if="successMessage" class="text-sm text-success mb-3">{{ successMessage }}</p>
          <p v-if="loading" class="text-sm text-text-light">加载中...</p>
          <template v-else>
            <div v-if="groupedItems.length === 0" class="text-sm text-text-light">
              暂无 Git 身份，请先新建。
            </div>
            <div v-else class="space-y-4">
              <div
                v-for="group in groupedItems"
                :key="group.company_id"
                class="border border-border rounded-lg p-4"
              >
                <p class="text-sm font-medium text-text mb-2">
                  公司：{{ group.company_name || '未设置' }}
                </p>
                <ul class="space-y-2">
                  <li
                    v-for="identity in group.identities"
                    :key="identity.id"
                    class="text-sm text-text bg-gray-50 border border-border rounded px-3 py-2"
                  >
                    <span class="font-medium">{{ identity.label || '未命名身份' }}</span>
                    <span
                      v-if="identity.is_default"
                      class="ml-2 inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary"
                    >
                      默认
                    </span>
                    <span class="mx-2 text-text-light">|</span>
                    <span>{{ identity.git_user_name }}</span>
                    <span class="mx-1 text-text-light">&lt;</span>
                    <span>{{ identity.git_user_email }}</span>
                    <span class="text-text-light">&gt;</span>
                    <span v-if="identity.git_remote_username" class="ml-2 text-xs text-text-light">
                      远端用户：{{ identity.git_remote_username }}
                    </span>
                    <button
                      v-if="!identity.is_default"
                      type="button"
                      :disabled="defaultingIdentityId === identity.id"
                      class="ml-3 text-xs px-2 py-1 rounded border border-primary/40 text-primary hover:bg-primary/5 disabled:opacity-50"
                      @click="setIdentityAsDefault(identity)"
                    >
                      {{ defaultingIdentityId === identity.id ? '设置中...' : '设为默认' }}
                    </button>
                  </li>
                </ul>
              </div>
            </div>
          </template>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import { apiFetch } from '../utils/apiUtils'
import { getCookie } from '../utils/cookieUtils'
import { extractTraceId } from '../utils/traceId.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { resolveAuthenticatedUserId } from '../utils/sessionUserIdUtils.js'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))

const loading = ref(false)
const companiesLoading = ref(false)
const membershipCompanies = ref([])
const membershipCompaniesError = ref('')
const membershipCompaniesErrorTraceId = ref('')
const creating = ref(false)
const defaultingIdentityId = ref('')
const errorMessage = ref('')
const errorMessageTraceId = ref('')
const successMessage = ref('')
const identities = ref([])
const form = ref({
  company_id: tenantId.value || '',
  label: '',
  git_user_name: '',
  git_user_email: '',
  git_remote_username: '',
  is_default: false,
})

const groupedItems = computed(() => {
  const map = new Map()
  for (const item of identities.value) {
    const cid = String(item.company_id || '')
    if (!map.has(cid)) {
      map.set(cid, {
        company_id: cid,
        company_name: item.company_name || '',
        identities: []
      })
    }
    map.get(cid).identities.push(item)
  }
  return Array.from(map.values())
})

const profileUserId = computed(() => {
  const fromRoute = String(route.params.id || route.params.userId || '').trim()
  if (fromRoute) return fromRoute
  return String(getCookie('userId') || '').trim()
})

const resolveProfileGitIdentitiesApiPath = async () => {
  const fromRoute = String(route.params.id || route.params.userId || '').trim()
  if (fromRoute) {
    return `/api/git-identities/user/${encodeURIComponent(fromRoute)}/`
  }
  const userId = await resolveAuthenticatedUserId()
  if (!userId) return ''
  return `/api/git-identities/user/${encodeURIComponent(userId)}/`
}

const syncFormCompanyFromContext = () => {
  const ids = new Set(membershipCompanies.value.map((x) => String(x.company_id)))
  const cur = String(form.value.company_id || '')
  if (cur && !ids.has(cur)) {
    form.value.company_id = ''
  }
  const tid = String(tenantId.value || '')
  if (!form.value.company_id && tid && ids.has(tid)) {
    form.value.company_id = tid
  }
}

const fetchMembershipCompanies = async () => {
  companiesLoading.value = true
  membershipCompaniesError.value = ''
  membershipCompaniesErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/profile/', {
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      membershipCompaniesErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      throw new Error(data.detail || data.error || '获取公司列表失败')
    }
    membershipCompanies.value = Array.isArray(data.company_nicknames) ? data.company_nicknames : []
    syncFormCompanyFromContext()
  } catch (error) {
    membershipCompanies.value = []
    membershipCompaniesError.value = error.message ? humanizeRequestErrorMessage(error.message) : '获取公司列表失败'
  } finally {
    companiesLoading.value = false
  }
}

watch([tenantId, membershipCompanies], () => {
  syncFormCompanyFromContext()
})

const fetchIdentities = async () => {
  loading.value = true
  errorMessage.value = ''
  errorMessageTraceId.value = ''
  try {
    const apiPath = await resolveProfileGitIdentitiesApiPath()
    if (!apiPath) {
      throw new Error('缺少 userId，无法获取 Git 身份')
    }
    const response = await apiFetch(apiPath, {
      headers: { Accept: 'application/json' }
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      errorMessageTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      throw new Error(data.detail || '获取身份列表失败')
    }
    identities.value = Array.isArray(data.identities) ? data.identities : []
  } catch (error) {
    errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '获取身份列表失败'
  } finally {
    loading.value = false
  }
}

const createIdentityGuard = createClickGuard()

const createIdentity = async () => {
  if (!form.value.company_id || !form.value.git_user_name || !form.value.git_user_email) {
    errorMessage.value = '请选择公司，并填写 Git 用户名和 Git 邮箱'
    errorMessageTraceId.value = ''
    return
  }
  // OPT-20260819-038: 创建 Git 身份是资源写操作，防连点双发 POST
  await createIdentityGuard.run(async ({ idempotencyKey }) => {
    creating.value = true
    errorMessage.value = ''
    errorMessageTraceId.value = ''
    successMessage.value = ''
    try {
      const apiPath = await resolveProfileGitIdentitiesApiPath()
      if (!apiPath) {
        throw new Error('缺少 userId，无法创建 Git 身份')
      }
      const response = await apiFetch(apiPath, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(form.value)
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorMessageTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        throw new Error(data.detail || '创建身份失败')
      }
      successMessage.value = '已创建 Git 身份'
      form.value.label = ''
      form.value.git_user_name = ''
      form.value.git_user_email = ''
      form.value.git_remote_username = ''
      form.value.is_default = false
      await fetchIdentities()
    } catch (error) {
      errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '创建身份失败'
    } finally {
      creating.value = false
    }
  })
}

const setIdentityAsDefaultGuard = createClickGuard()

const setIdentityAsDefault = async (identity) => {
  const identityId = String(identity?.id || '').trim()
  if (!identityId) return
  // OPT-20260819-038: 设默认身份是资源写操作，防连点双发 PATCH
  await setIdentityAsDefaultGuard.run(async ({ idempotencyKey }) => {
    defaultingIdentityId.value = identityId
    errorMessage.value = ''
    errorMessageTraceId.value = ''
    successMessage.value = ''
    try {
      const apiPath = await resolveProfileGitIdentitiesApiPath()
      if (!apiPath) {
        throw new Error('缺少 userId，无法设置默认身份')
      }
      const response = await apiFetch(apiPath, {
        method: 'PATCH',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          identity_id: identityId,
          is_default: true
        })
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorMessageTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        throw new Error(data.detail || '设置默认身份失败')
      }
      successMessage.value = '已设置默认身份'
      await fetchIdentities()
    } catch (error) {
      errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '设置默认身份失败'
    } finally {
      defaultingIdentityId.value = ''
    }
  })
}

onMounted(() => {
  void fetchIdentities()
  void fetchMembershipCompanies()
})
</script>
