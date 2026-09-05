<template>
  <div class="app-modal-overlay bg-black/50 flex items-center justify-center z-50">
    <div class="bg-white rounded-xl p-6 max-w-lg w-full max-h-[85vh] overflow-y-auto">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold">
          Git 提交身份 · {{ memberName || '成员' }}
        </h3>
        <button type="button" class="text-text-light hover:text-text" @click="$emit('close')">关闭</button>
      </div>
      <p class="mb-3 text-xs text-text-light leading-relaxed">
        此处仅管理 git commit 的 name/email，与 GitHub/GitLab 网站 OAuth 授权无关。
        OAuth 请在账号中心「Git 网站 OAuth」中绑定。
      </p>

      <p
        v-if="errorMessage"
        class="mb-3 text-sm text-danger"
        data-testid="member-git-identity-error"
        :data-traceId="errorTraceId || undefined"
      >
        {{ errorMessage }}
      </p>

      <div v-if="loading" class="py-6 text-center text-text-light">加载中...</div>

      <ul v-else class="space-y-2 mb-4">
        <li
          v-for="row in identities"
          :key="row.id"
          class="border border-border rounded-lg px-3 py-2 flex justify-between gap-3 items-start"
        >
          <div class="min-w-0">
            <div class="font-medium truncate">
              {{ row.git_user_name }}
              <span v-if="row.is_default" class="ml-2 text-xs text-primary">默认</span>
              <span v-if="row.label === 'system-auto'" class="ml-2 text-xs text-text-light">系统</span>
            </div>
            <div class="text-sm text-text-light truncate">{{ row.git_user_email }}</div>
            <div v-if="row.label" class="text-xs text-text-light">{{ row.label }}</div>
          </div>
          <div class="flex flex-col gap-1 shrink-0">
            <button
              v-if="!row.is_default"
              type="button"
              class="text-xs text-primary hover:underline"
              @click="setDefault(row)"
            >
              设默认
            </button>
            <button type="button" class="text-xs text-danger hover:underline" @click="removeIdentity(row)">
              删除
            </button>
          </div>
        </li>
        <li v-if="!identities.length" class="text-sm text-text-light py-2">暂无 Git 身份</li>
      </ul>

      <form class="border-t border-border pt-4 space-y-3" @submit.prevent="createIdentity">
        <h4 class="text-sm font-medium">新增身份</h4>
        <input
          v-model.trim="form.label"
          type="text"
          placeholder="标签（可选）"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm"
        >
        <input
          v-model.trim="form.git_user_name"
          type="text"
          required
          placeholder="Git 用户名"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm"
        >
        <input
          v-model.trim="form.git_user_email"
          type="email"
          required
          placeholder="Git 邮箱"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm"
        >
        <label class="inline-flex items-center gap-2 text-sm">
          <input v-model="form.is_default" type="checkbox">
          设为默认
        </label>
        <div class="flex justify-end gap-2">
          <button type="button" class="px-3 py-2 border border-border rounded-lg text-sm" @click="$emit('close')">
            取消
          </button>
          <button
            type="submit"
            class="px-3 py-2 bg-primary text-white rounded-lg text-sm disabled:opacity-50"
            :disabled="saving"
          >
            {{ saving ? '保存中…' : '创建' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { humanizeRequestErrorMessage, showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const props = defineProps({
  tenantId: { type: String, required: true },
  memberId: { type: String, required: true },
  memberName: { type: String, default: '' },
})

defineEmits(['close'])

const identities = ref([])
const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const errorTraceId = ref('')
const form = reactive({
  label: '',
  git_user_name: '',
  git_user_email: '',
  is_default: false,
})

// OPT-20260819-038: Git 提交身份增删/设默认为资源写路径，createClickGuard 防连点双发。
const createGuard = createClickGuard()
const setDefaultGuard = createClickGuard()
const removeGuard = createClickGuard()

const apiBase = () =>
  `/api/git-identities/tenant/${encodeURIComponent(props.tenantId)}/member/${encodeURIComponent(props.memberId)}/`

const identityApi = (id) =>
  `/api/git-identities/tenant/${encodeURIComponent(props.tenantId)}/identity/${encodeURIComponent(id)}/`

const setError = (err) => {
  errorMessage.value = humanizeRequestErrorMessage(err?.message || '操作失败')
  errorTraceId.value = err?.traceId || ''
  showRequestError(errorMessage.value, err)
}

const loadIdentities = async () => {
  loading.value = true
  errorMessage.value = ''
  errorTraceId.value = ''
  try {
    const resp = await apiFetch(apiBase(), {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) {
      const err = new Error('加载 Git 身份失败')
      err.traceId = resp.traceId || ''
      throw err
    }
    const data = await resp.json()
    identities.value = Array.isArray(data.identities) ? data.identities : []
  } catch (err) {
    setError(err)
    identities.value = []
  } finally {
    loading.value = false
  }
}

const createIdentity = async () => {
  await createGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    errorMessage.value = ''
    try {
      const resp = await apiFetch(apiBase(), {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey
        ),
        body: JSON.stringify({
          label: form.label,
          git_user_name: form.git_user_name,
          git_user_email: form.git_user_email,
          is_default: form.is_default,
        }),
      })
      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}))
        const err = new Error(body.detail || body.message || '创建失败')
        err.traceId = resp.traceId || body._traceId || ''
        throw err
      }
      form.label = ''
      form.git_user_name = ''
      form.git_user_email = ''
      form.is_default = false
      await loadIdentities()
    } catch (err) {
      setError(err)
    } finally {
      saving.value = false
    }
  })
}

const setDefault = async (row) => {
  await setDefaultGuard.run(async ({ idempotencyKey }) => {
    try {
      const resp = await apiFetch(identityApi(row.id), {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey
        ),
        body: JSON.stringify({ is_default: true }),
      })
      if (!resp.ok) {
        const err = new Error('设置默认失败')
        err.traceId = resp.traceId || ''
        throw err
      }
      await loadIdentities()
    } catch (err) {
      setError(err)
    }
  })
}

const removeIdentity = async (row) => {
  if (!window.confirm(`确认删除身份 ${row.git_user_email}？`)) return
  await removeGuard.run(async ({ idempotencyKey }) => {
    try {
      const resp = await apiFetch(identityApi(row.id), {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
      })
      if (!resp.ok) {
        const err = new Error('删除失败')
        err.traceId = resp.traceId || ''
        throw err
      }
      await loadIdentities()
    } catch (err) {
      setError(err)
    }
  })
}

onMounted(loadIdentities)
watch(() => [props.tenantId, props.memberId], loadIdentities)
</script>
