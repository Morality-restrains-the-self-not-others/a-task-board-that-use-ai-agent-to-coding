<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6 space-y-4">
    <div>
      <h2 class="text-lg font-semibold text-text mb-1">推荐渠道</h2>
      <p class="text-sm text-text-light" data-testid="referral-channel-intro">为不同投放渠道创建独立推荐码与链接。无推荐资格时仍可发码；分账以被推荐人下单时是否具备资格为准，获资后可从已推荐用户的后续下单获得分成。</p>
    </div>

    <div v-if="loadError" class="text-sm text-red-500" :data-traceId="loadErrorTraceId || undefined">{{ loadError }}</div>
    <p v-else-if="loading" class="text-sm text-text-light">加载渠道...</p>

    <div
      v-for="ch in displayChannels"
      :key="ch.code"
      class="border border-border rounded-lg p-4 space-y-3"
      :data-testid="ch.is_default ? 'referral-default-channel' : undefined"
    >
      <div class="flex items-center gap-2">
        <p class="text-sm font-medium text-text">{{ ch.name }}</p>
        <span v-if="ch.is_default" class="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-text-light">默认</span>
        <span v-if="ch.status === 'disabled'" class="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-text-light">已禁用</span>
      </div>
      <div>
        <p class="text-sm text-text-light mb-2">推荐码</p>
        <p
          :data-testid="ch.is_default ? 'referral-code' : undefined"
          class="text-base font-medium text-text break-all"
        >{{ ch.code }}</p>
      </div>
      <div>
        <p class="text-sm text-text-light mb-2">推荐链接</p>
        <p
          :data-testid="ch.is_default ? 'referral-share-link' : undefined"
          class="text-sm text-text break-all bg-gray-50 border border-border rounded-lg p-3"
        >{{ shareLink(ch.code) }}</p>
      </div>
      <div class="flex items-center gap-3">
        <!-- Anti-Replay-OK: ui-only clipboard -->
        <button
          type="button"
          class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors"
          @click="copyLink(ch.code)"
        >
          复制推荐链接
        </button>
        <span v-if="copiedCode === ch.code" class="text-sm text-success">已复制</span>
        <button
          v-if="!ch.is_default && ch.status !== 'disabled'"
          type="button"
          class="text-sm text-text-light underline disabled:opacity-50"
          :disabled="disabling"
          :aria-busy="disabling ? 'true' : 'false'"
          @click="onDisable(ch.code)"
        >
          {{ disabling ? '禁用中...' : '禁用渠道' }}
        </button>
        <button
          v-if="!ch.is_default"
          type="button"
          class="text-sm text-text-light underline disabled:opacity-50"
          :data-testid="`referral-channel-delete-${ch.code}`"
          :disabled="deleting"
          :aria-busy="deleting ? 'true' : 'false'"
          @click="onDelete(ch.code, ch.name)"
        >
          {{ deleting ? '删除中...' : '删除渠道' }}
        </button>
      </div>
    </div>

    <form class="border-t border-border pt-4 space-y-3" @submit.prevent>
      <label class="block text-sm font-medium text-text" for="referral-channel-name">新渠道名称</label>
      <input
        id="referral-channel-name"
        v-model="newName"
        data-testid="referral-channel-name"
        class="w-full px-3 py-2 border border-border rounded-lg text-sm"
        maxlength="32"
        placeholder="例如：微信、抖音"
        :disabled="creating"
      />
      <button
        type="button"
        data-testid="referral-channel-create"
        class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        :disabled="!canCreate || creating"
        :aria-busy="creating ? 'true' : 'false'"
        @click="onCreate"
      >
        {{ creating ? '创建中...' : '创建渠道码' }}
      </button>
      <p v-if="createError" class="text-sm text-red-500" :data-traceId="createErrorTraceId || undefined">{{ createError }}</p>
    </form>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { messageFromFailedResponse } from '../utils/httpError.js'

const props = defineProps({
  fallbackCode: { type: String, default: '' },
})

const emit = defineEmits(['channels-loaded'])

const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const channels = ref([])
const newName = ref('')
const creating = ref(false)
const createError = ref('')
const createErrorTraceId = ref('')
const copiedCode = ref('')
const disabling = ref(false)
const deleting = ref(false)
const createGuard = createClickGuard()
const disableGuard = createClickGuard()
const deleteGuard = createClickGuard()

const displayChannels = computed(() => {
  if (channels.value.length) return channels.value
  return [{
    code: String(props.fallbackCode || '').trim(),
    name: '默认',
    is_default: true,
    status: 'active',
  }]
})

const canCreate = computed(() => {
  const n = Array.from(newName.value.trim()).length
  return n >= 1 && n <= 32
})

const shareLink = (code) => {
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  if (!code) return `${origin}/auth/register/`
  return `${origin}/auth/register/?accessCode=${encodeURIComponent(code)}`
}

const copyLink = async (code) => {
  copiedCode.value = ''
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(shareLink(code))
      copiedCode.value = code
    }
  } catch (error) {
    console.error('复制推荐链接失败:', error)
  }
}

const parseList = (data) => {
  if (Array.isArray(data)) return data
  if (data && Array.isArray(data.channels)) return data.channels
  return []
}

const fetchChannels = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/referral/channels/', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) {
      loadErrorTraceId.value = extractTraceId(response)
      loadError.value = '获取推荐渠道失败，请稍后重试'
      emit('channels-loaded', displayChannels.value)
      return
    }
    const data = await response.json()
    channels.value = parseList(data)
    emit('channels-loaded', displayChannels.value)
  } catch (error) {
    loadErrorTraceId.value = extractTraceId(error)
    loadError.value = '获取推荐渠道失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

const onCreate = async () => {
  if (!canCreate.value || creating.value) return
  await createGuard.run(async ({ idempotencyKey }) => {
    creating.value = true
    createError.value = ''
    createErrorTraceId.value = ''
    try {
      const response = await apiFetch('/api/referral/channels/', {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json', 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ name: newName.value.trim() }),
      })
      if (!response.ok) {
        createError.value = messageFromFailedResponse(response, '创建渠道失败，请稍后重试')
        createErrorTraceId.value = extractTraceId(response)
        return
      }
      const data = await response.json().catch(() => ({}))
      newName.value = ''
      await fetchChannels()
    } catch (error) {
      createError.value = '网络错误，请稍后重试'
      createErrorTraceId.value = extractTraceId(error)
    } finally {
      creating.value = false
    }
  })
}

const onDisable = async (code) => {
  if (!code || disabling.value) return
  await disableGuard.run(async ({ idempotencyKey }) => {
    disabling.value = true
    try {
      const response = await apiFetch(`/api/referral/channels/code/${encodeURIComponent(code)}/disable/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json', 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: '{}',
      })
      if (!response.ok) {
        loadErrorTraceId.value = extractTraceId(response)
        loadError.value = '禁用渠道失败，请稍后重试'
        return
      }
      await fetchChannels()
    } catch (error) {
      loadErrorTraceId.value = extractTraceId(error)
      loadError.value = '禁用渠道失败，请稍后重试'
    } finally {
      disabling.value = false
    }
  })
}

const onDelete = async (code, name) => {
  if (!code || deleting.value) return
  const confirmed = typeof window !== 'undefined' && window.confirm(
    `确定删除渠道「${name}」？删除后该渠道的推荐码与链接将永久失效（历史分账不受影响）。`,
  )
  if (!confirmed) return
  await deleteGuard.run(async ({ idempotencyKey }) => {
    deleting.value = true
    try {
      const response = await apiFetch(`/api/referral/channels/code/${encodeURIComponent(code)}/`, {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json' },
          idempotencyKey,
        ),
      })
      if (!response.ok) {
        loadErrorTraceId.value = extractTraceId(response)
        loadError.value = '删除渠道失败，请稍后重试'
        return
      }
      await fetchChannels()
    } catch (error) {
      loadErrorTraceId.value = extractTraceId(error)
      loadError.value = '删除渠道失败，请稍后重试'
    } finally {
      deleting.value = false
    }
  })
}

onMounted(fetchChannels)
</script>
