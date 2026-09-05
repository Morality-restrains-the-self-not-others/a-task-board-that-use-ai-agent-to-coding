<template>
  <div
    v-if="visible"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999"
    @click="handleOverlayClick"
  >
    <div
      class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 z-10000 relative mx-4"
      @click.stop
    >
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">创建新的 Git 身份</h3>
        <button
          class="text-gray-400 hover:text-gray-600"
          @click="close"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <p
        v-if="errorMessage"
        class="mb-3 text-xs text-red-600 leading-snug"
        v-bind="errorMessageTraceId ? { 'data-traceId': errorMessageTraceId } : {}"
      >{{ errorMessage }}</p>
      <p v-if="successMessage" class="mb-3 text-xs text-green-600 leading-snug">{{ successMessage }}</p>

      <div class="space-y-3">
        <!-- 公司 -->
        <div>
          <label class="text-xs text-gray-600 mb-0.5 block" for="git-id-modal-company">公司</label>
          <select
            id="git-id-modal-company"
            v-model="form.company_id"
            class="w-full px-2.5 py-1.5 text-xs border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary bg-white disabled:opacity-50"
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
          <p v-if="companiesError" class="mt-0.5 text-[11px] text-red-500" v-bind="companiesErrorTraceId ? { 'data-traceId': companiesErrorTraceId } : {}">{{ companiesError }}</p>
          <p
            v-else-if="!companiesLoading && membershipCompanies.length === 0"
            class="mt-0.5 text-[11px] text-gray-400"
          >
            你尚未加入任何公司，无法创建 Git 身份。
          </p>
        </div>

        <!-- 身份标签 -->
        <div>
          <label class="text-xs text-gray-600 mb-0.5 block" for="git-id-modal-label">身份标签（可选）</label>
          <input
            id="git-id-modal-label"
            v-model.trim="form.label"
            type="text"
            class="w-full px-2.5 py-1.5 text-xs border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
            placeholder="例如：公司主账号"
          >
        </div>

        <!-- Git 用户名 -->
        <div>
          <label class="text-xs text-gray-600 mb-0.5 block" for="git-id-modal-username">Git 用户名</label>
          <input
            id="git-id-modal-username"
            v-model.trim="form.git_user_name"
            type="text"
            class="w-full px-2.5 py-1.5 text-xs border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
            placeholder="例如：alice"
          >
        </div>

        <!-- Git 邮箱 -->
        <div>
          <label class="text-xs text-gray-600 mb-0.5 block" for="git-id-modal-email">Git 邮箱</label>
          <input
            id="git-id-modal-email"
            v-model.trim="form.git_user_email"
            type="email"
            class="w-full px-2.5 py-1.5 text-xs border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
            placeholder="例如：alice@company.com"
          >
        </div>

        <!-- 设为默认 -->
        <div class="flex items-center gap-2">
          <input id="git-id-modal-default" v-model="form.is_default" type="checkbox" class="w-3.5 h-3.5">
          <label for="git-id-modal-default" class="text-xs text-gray-600">设为该公司的默认身份</label>
        </div>
      </div>

      <div class="mt-5 flex justify-end gap-2">
        <button
          type="button"
          class="px-3 py-1.5 text-xs border border-gray-300 rounded bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          :disabled="creating"
          @click="close"
        >
          取消
        </button>
        <button
          type="button"
          class="px-3 py-1.5 text-xs rounded bg-primary text-white hover:bg-primary/90 disabled:opacity-60 disabled:cursor-not-allowed"
          :disabled="creating || !canSubmit || createIdentityGuard.isBusy()"
          @click="createIdentity"
        >
          {{ creating ? '创建中...' : '创建身份' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import { resolveAuthenticatedUserId } from '../../utils/sessionUserIdUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { humanizeRequestErrorMessage } from '../../utils/requestErrorDisplay.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  tenantId: { type: String, default: '' },
})

const emit = defineEmits(['close', 'created'])

const creating = ref(false)
const companiesLoading = ref(false)
const membershipCompanies = ref([])
const companiesError = ref('')
const companiesErrorTraceId = ref('')
const errorMessage = ref('')
const errorMessageTraceId = ref('')
const successMessage = ref('')

const form = ref({
  company_id: '',
  label: '',
  git_user_name: '',
  git_user_email: '',
  git_remote_username: '',
  is_default: false,
})

const canSubmit = computed(() => {
  return (
    String(form.value.company_id || '').trim() !== '' &&
    String(form.value.git_user_name || '').trim() !== '' &&
    String(form.value.git_user_email || '').trim() !== ''
  )
})

const close = () => {
  errorMessage.value = ''
  successMessage.value = ''
  emit('close')
}

const handleOverlayClick = () => {
  close()
}

const fetchMembershipCompanies = async () => {
  companiesLoading.value = true
  companiesError.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/profile/', {
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      throw new Error(data.detail || data.error || '获取公司列表失败')
    }
    membershipCompanies.value = Array.isArray(data.company_nicknames) ? data.company_nicknames : []
    syncFormCompanyFromContext()
  } catch (error) {
    membershipCompanies.value = []
    companiesError.value = humanizeRequestErrorMessage(error.message || '获取公司列表失败')
    companiesErrorTraceId.value = extractTraceId(error) || (error.response ? error.response.traceId : '')
    errorMessageTraceId.value = companiesErrorTraceId.value
  } finally {
    companiesLoading.value = false
  }
}

const syncFormCompanyFromContext = () => {
  const ids = new Set(membershipCompanies.value.map((x) => String(x.company_id)))
  const cur = String(form.value.company_id || '')
  if (cur && !ids.has(cur)) {
    form.value.company_id = ''
  }
  const tid = String(props.tenantId || '')
  if (!form.value.company_id && tid && ids.has(tid)) {
    form.value.company_id = tid
  }
}

watch(
  () => [props.tenantId, membershipCompanies.value],
  () => {
    syncFormCompanyFromContext()
  },
)

watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      errorMessage.value = ''
      successMessage.value = ''
      form.value.label = ''
      form.value.git_user_name = ''
      form.value.git_user_email = ''
      form.value.git_remote_username = ''
      form.value.is_default = false
      if (membershipCompanies.value.length === 0) {
        void fetchMembershipCompanies()
      } else {
        syncFormCompanyFromContext()
      }
    }
  },
)

const resolveApiPath = async () => {
  const userId = await resolveAuthenticatedUserId()
  if (!userId) throw new Error('缺少 userId，无法创建 Git 身份')
  return `/api/git-identities/user/${encodeURIComponent(userId)}/`
}

const createIdentityGuard = createClickGuard()

const createIdentity = async () => {
  // OPT-20260819-038: 创建 Git 身份是写操作，防连点双发 POST
  await createIdentityGuard.run(async ({ idempotencyKey }) => {
    if (!canSubmit.value) {
      errorMessage.value = '请选择公司，并填写 Git 用户名和 Git 邮箱'
      return
    }
    creating.value = true
    errorMessage.value = ''
    errorMessageTraceId.value = ''
    successMessage.value = ''
    try {
      const apiPath = await resolveApiPath()
      const response = await apiFetch(apiPath, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(form.value),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorMessageTraceId.value = response.traceId || ''
        throw new Error(data.detail || '创建身份失败')
      }
      successMessage.value = '已创建 Git 身份'
      form.value.label = ''
      form.value.git_user_name = ''
      form.value.git_user_email = ''
      form.value.git_remote_username = ''
      form.value.is_default = false
      emit('created')
      // 短暂延迟后关闭，让用户看到成功提示
      setTimeout(() => {
        close()
      }, 600)
    } catch (error) {
      errorMessage.value = humanizeRequestErrorMessage(error.message || '创建身份失败')
      if (!errorMessageTraceId.value) errorMessageTraceId.value = extractTraceId(error)
    } finally {
      creating.value = false
    }
  })
}
</script>
