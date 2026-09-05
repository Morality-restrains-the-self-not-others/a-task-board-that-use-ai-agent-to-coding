<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6 overflow-x-auto" data-testid="access-token-panel">
    <h2 class="text-lg font-semibold text-text mb-4">访问令牌</h2>
    <p class="text-sm text-text-light mb-4">
      访问令牌可用于替代密码登录或 API 认证。创建后<strong class="font-medium text-danger">仅完整展示一次</strong>，请妥善保管。
    </p>

    <div class="mb-6 flex flex-wrap gap-3 items-end">
      <div class="flex-1 min-w-[200px] max-w-md">
        <label class="block text-sm text-text-light mb-1">令牌名称</label>
        <input
          v-model="newTokenName"
          type="text"
          maxlength="255"
          placeholder="例如：CLI 工具、API 自动部署"
          class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none transition-colors"
          @keyup.enter="createAccessToken"
        >
      </div>
      <div class="w-40">
        <label class="block text-sm text-text-light mb-1">有效期</label>
        <select
          v-model="newTokenExpiry"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-primary outline-none bg-white transition-colors"
        >
          <option value="0">永不过期</option>
          <option value="30">30 天</option>
          <option value="90">90 天</option>
          <option value="180">180 天</option>
          <option value="365">365 天</option>
        </select>
      </div>
      <button
        type="button"
        :disabled="isCreatingToken"
        class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
        @click="createAccessToken"
      >
        {{ isCreatingToken ? '创建中...' : '创建令牌' }}
      </button>
    </div>

    <div v-if="newlyCreatedToken" class="mb-6 rounded-lg border-2 border-warning/50 bg-warning/5 p-4">
      <p class="text-sm font-semibold text-text mb-2 flex items-center gap-1">
        <span>⚠️</span> 令牌已创建，以下内容仅显示一次，请立即复制保存
      </p>
      <div class="flex gap-2 items-center">
        <code class="flex-1 px-3 py-2 bg-gray-100 rounded text-sm font-mono break-all select-all">{{ newlyCreatedToken }}</code>
        <button
          type="button"
          class="shrink-0 px-3 py-2 text-sm rounded-lg border border-border text-text hover:bg-gray-50 transition-colors"
          @click="copyNewToken"
        >
          {{ tokenCopied ? '已复制' : '复制' }}
        </button>
      </div>
      <button
        type="button"
        class="mt-3 text-sm text-primary hover:underline"
        @click="newlyCreatedToken = ''; tokenCopied = false"
      >
        我已保存，关闭提示
      </button>
    </div>

    <div
      class="mb-4 flex flex-wrap gap-2"
      role="tablist"
      aria-label="按状态筛选访问令牌"
      data-testid="access-token-status-tabs"
    >
      <button
        v-for="tab in statusTabs"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="statusFilter === tab.id"
        :data-testid="`access-token-tab-${tab.id}`"
        class="px-3 py-1.5 text-sm rounded-lg border transition-colors"
        :class="statusFilter === tab.id
          ? 'bg-primary text-white border-primary'
          : 'border-border text-text hover:bg-gray-50'"
        @click="setStatusFilter(tab.id)"
      >
        {{ tab.label }}
        <span class="ml-1 tabular-nums opacity-80">({{ tab.count }})</span>
      </button>
    </div>

    <div v-if="isLoadingTokens" class="text-sm text-text-light">加载令牌列表…</div>

    <div v-else-if="listView.total > 0" class="space-y-3" data-testid="access-token-list">
      <div
        v-for="token in listView.items"
        :key="token.id"
        class="flex flex-wrap items-center justify-between gap-3 border border-border rounded-lg p-4"
        :class="token.is_revoked ? 'opacity-60 border-dashed bg-gray-50/60' : 'bg-white'"
        :data-testid="token.is_revoked ? 'access-token-row-revoked' : 'access-token-row-active'"
      >
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="text-sm font-medium text-text">{{ token.name }}</span>
            <span
              v-if="token.is_revoked"
              class="text-xs px-2 py-0.5 rounded-full bg-red-100 text-danger"
            >已吊销</span>
            <span
              v-else
              class="text-xs px-2 py-0.5 rounded-full bg-green-100 text-success"
            >有效</span>
          </div>
          <div class="text-xs text-text-light mt-1 flex flex-wrap gap-x-4 gap-y-1">
            <span>尾号: <code class="text-text">{{ token.last_4 }}</code></span>
            <span>创建于 {{ formatTokenDate(token.created_at) }}</span>
            <span v-if="token.expires_at" class="text-warning">过期 {{ formatTokenDate(token.expires_at) }}</span>
            <span v-if="token.last_used_at">最后使用 {{ formatTokenDate(token.last_used_at) }}</span>
          </div>
        </div>
        <button
          v-if="!token.is_revoked"
          type="button"
          :disabled="isRevokingToken === token.id"
          class="shrink-0 px-3 py-1.5 text-sm rounded-lg border border-red-300 text-danger hover:bg-red-50 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
          @click="revokeAccessToken(token.id)"
        >
          {{ isRevokingToken === token.id ? '吊销中...' : '吊销' }}
        </button>
      </div>

      <div
        v-if="listView.totalPages > 1"
        class="flex flex-wrap items-center justify-between gap-3 pt-2"
        data-testid="access-token-pagination"
      >
        <p class="text-xs text-text-light">
          共 {{ listView.total }} 条，第 {{ listView.currentPage }} / {{ listView.totalPages }} 页
        </p>
        <nav class="flex items-center gap-1" aria-label="访问令牌分页">
          <button
            type="button"
            class="px-3 py-1 border rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed"
            :disabled="listView.currentPage === 1"
            @click="setPage(listView.currentPage - 1)"
          >
            上一页
          </button>
          <button
            v-for="(item, idx) in pageItems"
            :key="item === 'ellipsis' ? `e-${idx}` : `p-${item}`"
            type="button"
            class="px-3 py-1 border rounded-md text-sm transition-colors"
            :class="item === 'ellipsis'
              ? 'text-gray-400 cursor-default'
              : (listView.currentPage === item
                ? 'bg-primary text-white border-primary'
                : 'hover:bg-gray-50')"
            :disabled="item === 'ellipsis'"
            :aria-current="listView.currentPage === item ? 'page' : undefined"
            @click="item !== 'ellipsis' && setPage(item)"
          >
            {{ item === 'ellipsis' ? '…' : item }}
          </button>
          <button
            type="button"
            class="px-3 py-1 border rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed"
            :disabled="listView.currentPage === listView.totalPages"
            @click="setPage(listView.currentPage + 1)"
          >
            下一页
          </button>
        </nav>
      </div>
    </div>

    <p v-else-if="accessTokens.length === 0" class="text-sm text-text-light">
      暂无访问令牌。请输入名称并点击「创建令牌」来生成第一个。
    </p>
    <p v-else class="text-sm text-text-light" data-testid="access-token-empty-filter">
      当前筛选下没有令牌。可切换到「全部」或其它状态查看。
    </p>

    <p
      v-if="tokenErrorMessage"
      class="mt-3 text-sm text-danger"
      :data-traceId="tokenErrorTraceId || undefined"
    >{{ tokenErrorMessage }}</p>
    <p v-if="tokenMessage" class="mt-3 text-sm text-success">{{ tokenMessage }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { buildAccessTokenListView } from '../utils/accessTokenListView.js'
import { buildPaginationItems } from '../utils/paginationPages.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const accessTokens = ref([])
const newTokenName = ref('')
const newTokenExpiry = ref('0')
const newlyCreatedToken = ref('')
const isCreatingToken = ref(false)
const isRevokingToken = ref(null)
const isLoadingTokens = ref(false)
const tokenErrorMessage = ref('')
const tokenErrorTraceId = ref('')
const tokenMessage = ref('')
const tokenCopied = ref(false)
const statusFilter = ref('active')
const currentPage = ref(1)

const listView = computed(() =>
  buildAccessTokenListView(accessTokens.value, {
    statusFilter: statusFilter.value,
    page: currentPage.value,
  }),
)

// OPT-20260819-038: 访问令牌创建/吊销为安全写路径，createClickGuard 防连点双发。
const createTokenGuard = createClickGuard()
const revokeGuard = createClickGuard()

const statusTabs = computed(() => {
  const c = listView.value.counts
  return [
    { id: 'active', label: '有效', count: c.active },
    { id: 'revoked', label: '已吊销', count: c.revoked },
    { id: 'all', label: '全部', count: c.all },
  ]
})

const pageItems = computed(() =>
  buildPaginationItems(listView.value.currentPage, listView.value.totalPages),
)

const formatTokenDate = (dateStr) => {
  if (!dateStr) return '—'
  try {
    const d = new Date(dateStr)
    if (Number.isNaN(d.getTime())) return dateStr
    return d.toLocaleString()
  } catch {
    return dateStr
  }
}

function extractTraceId(response) {
  if (!response) return ''
  const h = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
  return String(h || '').trim()
}

function setStatusFilter(id) {
  statusFilter.value = id
  currentPage.value = 1
}

function setPage(page) {
  currentPage.value = page
}

const fetchAccessTokens = async () => {
  isLoadingTokens.value = true
  tokenErrorMessage.value = ''
  tokenErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/access-tokens/', {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json()
    if (!response.ok) {
      tokenErrorTraceId.value = extractTraceId(response) || String(data.trace_id || data.traceId || '')
      throw new Error(data.error || data.detail || '获取令牌列表失败')
    }
    accessTokens.value = Array.isArray(data.tokens) ? data.tokens : []
    // 同步页码：筛选/吊销后可能需要回退到有效页
    currentPage.value = listView.value.currentPage
  } catch (error) {
    console.error('获取访问令牌列表失败:', error)
    tokenErrorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '获取令牌列表失败'
  } finally {
    isLoadingTokens.value = false
  }
}

const createAccessToken = async () => {
  await createTokenGuard.run(async ({ idempotencyKey }) => {
    tokenErrorMessage.value = ''
    tokenErrorTraceId.value = ''
    tokenMessage.value = ''
    newlyCreatedToken.value = ''
    tokenCopied.value = false
    isCreatingToken.value = true
    try {
      const response = await apiFetch('/api/accounts/users/access-tokens/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey
        ),
        body: JSON.stringify({
          name: newTokenName.value.trim() || '默认令牌',
          expires_in_days: newTokenExpiry.value,
        }),
      })
      const data = await response.json()
      if (!response.ok) {
        tokenErrorTraceId.value = extractTraceId(response) || String(data.trace_id || data.traceId || '')
        throw new Error(data.error || data.detail || '创建令牌失败')
      }
      newlyCreatedToken.value = data.token || ''
      newTokenName.value = ''
      tokenMessage.value = '令牌创建成功'
      statusFilter.value = 'active'
      currentPage.value = 1
      await fetchAccessTokens()
    } catch (error) {
      console.error('创建访问令牌失败:', error)
      tokenErrorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '创建令牌失败'
    } finally {
      isCreatingToken.value = false
    }
  })
}

const revokeAccessToken = async (tokenId) => {
  await revokeGuard.run(async ({ idempotencyKey }) => {
    tokenErrorMessage.value = ''
    tokenErrorTraceId.value = ''
    tokenMessage.value = ''
    isRevokingToken.value = tokenId
    try {
      const response = await apiFetch(`/api/accounts/users/access-tokens/${tokenId}/`, {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        tokenErrorTraceId.value = extractTraceId(response) || String(data.trace_id || data.traceId || '')
        throw new Error(data.error || data.detail || '吊销令牌失败')
      }
      tokenMessage.value = '令牌已吊销'
      await fetchAccessTokens()
    } catch (error) {
      console.error('吊销访问令牌失败:', error)
      tokenErrorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '吊销令牌失败'
    } finally {
      isRevokingToken.value = null
    }
  })
}

const copyNewToken = async () => {
  if (!newlyCreatedToken.value) return
  try {
    await navigator.clipboard.writeText(newlyCreatedToken.value)
    tokenCopied.value = true
    setTimeout(() => { tokenCopied.value = false }, 3000)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = newlyCreatedToken.value
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    tokenCopied.value = true
    setTimeout(() => { tokenCopied.value = false }, 3000)
  }
}

onMounted(() => {
  fetchAccessTokens()
})
</script>
