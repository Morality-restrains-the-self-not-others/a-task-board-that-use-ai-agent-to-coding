<template>
  <div
    data-testid="system-admin-referral-applications-panel"
    class="bg-white rounded-lg shadow-sm border border-gray-200 p-6"
  >
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-gray-900">推荐码申请列表</h2>
      <div class="flex items-center gap-3">
        <select
          v-model="statusFilter"
          class="px-3 py-2 border border-gray-300 rounded-lg text-sm"
          @change="loadApplications"
        >
          <option value="">全部状态</option>
          <option value="pending">待审批</option>
          <option value="approved">已通过</option>
          <option value="rejected">已拒绝</option>
          <option value="expired">已过期</option>
          <option value="revoked">已取消资格</option>
        </select>
        <button
          class="text-sm text-primary hover:underline"
          :disabled="loadingApps"
          @click="loadApplications"
        >
          <!-- Anti-Replay-OK: read-refresh -->
          {{ loadingApps ? '刷新中...' : '刷新' }}
        </button>
      </div>
    </div>

    <!-- 推荐码反查：输入推荐码反向定位具体用户（管理员排查用） -->
    <div class="mb-6 p-4 bg-gray-50 rounded-lg border border-gray-200" data-testid="referral-code-lookup">
      <h3 class="text-sm font-semibold text-gray-900 mb-1">推荐码反查</h3>
      <p class="text-xs text-gray-500 mb-3">输入推广链接中的推荐码，反查持有该码的用户</p>
      <div class="flex items-center gap-2">
        <input
          v-model="codeQuery"
          class="px-3 py-2 border border-gray-300 rounded-lg text-sm flex-1 max-w-xs font-mono"
          placeholder="推荐码，如 DR2AKvP9J9"
          data-testid="referral-code-lookup-input"
          @keyup.enter="lookupShareCode"
        />
        <button
          class="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-white disabled:opacity-50"
          :disabled="codeSearching"
          data-testid="referral-code-lookup-btn"
          @click="lookupShareCode"
        >
          {{ codeSearching ? '查询中...' : '反查' }}
        </button>
      </div>

      <div
        v-if="codeError"
        class="mt-3 p-3 bg-red-50 text-red-700 rounded-lg text-sm"
        :data-traceId="codeErrorTraceId || undefined"
        data-testid="referral-code-lookup-error"
      >
        {{ codeError }}
      </div>

      <div
        v-if="codeResult && codeResult.found"
        class="mt-3 p-3 bg-green-50 border border-green-200 rounded-lg text-sm"
        data-testid="referral-code-lookup-result"
      >
        <div class="flex items-center gap-2 mb-2">
          <span class="text-xs text-green-700 font-medium bg-green-100 px-2 py-0.5 rounded-full">已找到</span>
          <span class="font-mono text-xs text-gray-700">{{ codeResult.code }}</span>
          <span
            v-if="!codeResult.is_default"
            class="px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-600"
          >渠道</span>
        </div>
        <dl class="grid grid-cols-2 gap-x-6 gap-y-1 text-xs">
          <div class="flex gap-2">
            <dt class="text-gray-500 w-16 shrink-0">用户 ID</dt>
            <dd class="font-mono break-all" data-testid="referral-code-lookup-user-id">{{ codeResult.user_id }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="text-gray-500 w-16 shrink-0">个人名称</dt>
            <dd data-testid="referral-code-lookup-username">{{ codeResult.username || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="text-gray-500 w-16 shrink-0">渠道名称</dt>
            <dd>{{ codeResult.channel_name || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="text-gray-500 w-16 shrink-0">状态</dt>
            <dd class="capitalize">{{ codeResult.status }}</dd>
          </div>
        </dl>
      </div>

      <div
        v-if="codeResult && !codeResult.found"
        class="mt-3 p-3 bg-gray-100 text-gray-600 rounded-lg text-sm"
        data-testid="referral-code-lookup-not-found"
      >
        未找到使用该推荐码的用户
      </div>
    </div>

    <div v-if="loadingApps" class="text-sm text-gray-500 py-8 text-center">加载中...</div>
    <div v-else-if="!applications.length" class="text-sm text-gray-500 py-8 text-center">暂无申请记录</div>

    <div v-else class="overflow-x-auto">
      <table class="min-w-full text-sm" data-testid="referral-applications-table">
        <thead>
          <tr class="text-left text-gray-500 border-b">
            <th class="py-2 pr-3 font-medium">ID</th>
            <th class="py-2 pr-3 font-medium">用户 ID</th>
            <th class="py-2 pr-3 font-medium">推荐码</th>
            <th class="py-2 pr-3 font-medium">个人名称</th>
            <th class="py-2 pr-3 font-medium">个人介绍</th>
            <th class="py-2 pr-3 font-medium">状态</th>
            <th class="py-2 pr-3 font-medium">分成比例</th>
            <th class="py-2 pr-3 font-medium">申请时间</th>
            <th class="py-2 pr-3 font-medium">过期时间</th>
            <th class="py-2 pr-3 font-medium">审批人</th>
            <th class="py-2 pr-3 font-medium">操作理由</th>
            <th class="py-2 pr-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="app in applications"
            :key="app.id"
            :ref="(el) => setRowRef(app.user_id, el)"
            class="border-b hover:bg-gray-50"
            :class="{
              'bg-yellow-50': app.status === 'pending',
              'bg-green-50': app.status === 'approved' && app.is_active,
              'bg-gray-50': app.status === 'rejected' || app.status === 'expired' || app.status === 'revoked',
              'ring-2 ring-primary ring-inset': highlightUserId === app.user_id,
            }"
          >
            <td class="py-2 pr-3">{{ app.id }}</td>
            <td class="py-2 pr-3 font-mono text-xs">{{ app.user_id }}</td>
            <td class="py-2 pr-3">
              <div class="flex items-center gap-2">
                <span class="font-mono text-xs" data-testid="referral-app-share-code">{{ app.share_code || '—' }}</span>
                <button
                  v-if="app.share_code"
                  class="text-xs text-primary hover:underline"
                  data-testid="referral-app-copy-code"
                  @click="copyShareCode(app.share_code)"
                >
                  {{ copiedCode === app.share_code ? '已复制' : '复制' }}
                </button>
              </div>
            </td>
            <td class="py-2 pr-3 text-xs" data-testid="referral-app-legal-name">{{ app.legal_name || '—' }}</td>
            <td class="py-2 pr-3 text-xs max-w-[16rem]">
              <span
                class="block truncate"
                data-testid="referral-app-intro"
                :title="app.personal_intro || ''"
              >{{ app.personal_intro || '—' }}</span>
            </td>
            <td class="py-2 pr-3">
              <span
                class="px-2 py-0.5 rounded-full text-xs font-medium"
                :class="statusClass(app)"
              >
                {{ app.status_display || app.status }}
              </span>
            </td>
            <td class="py-2 pr-3 text-xs" data-testid="referral-app-ratio">
              {{ app.referral_rate_display || app.profit_sharing_ratio_display || '—' }}
            </td>
            <td class="py-2 pr-3 text-xs">{{ formatDate(app.applied_at) }}</td>
            <td class="py-2 pr-3 text-xs">{{ formatDate(app.expires_at) }}</td>
            <td class="py-2 pr-3 text-xs">{{ app.reviewed_by || '—' }}</td>
            <td class="py-2 pr-3 text-xs">{{ app.last_action_reason || app.reject_reason || '—' }}</td>
            <td class="py-2 pr-3">
              <div class="flex items-center gap-2">
                <template v-if="app.status === 'pending'">
                  <button
                    class="px-3 py-1 text-xs bg-green-500 text-white rounded hover:bg-green-600 disabled:opacity-50"
                    :disabled="actingId === app.id"
                    data-testid="referral-app-approve"
                    @click="openAction(app, 'approve')"
                  >
                    {{ actingId === app.id && actingType === 'approve' ? '...' : '通过' }}
                  </button>
                  <button
                    class="px-3 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50"
                    :disabled="actingId === app.id"
                    data-testid="referral-app-reject"
                    @click="openAction(app, 'reject')"
                  >
                    {{ actingId === app.id && actingType === 'reject' ? '...' : '拒绝' }}
                  </button>
                </template>
                <button
                  v-if="app.can_revoke || (app.status === 'approved' && app.is_active)"
                  class="px-3 py-1 text-xs bg-orange-500 text-white rounded hover:bg-orange-600 disabled:opacity-50"
                  :disabled="actingId === app.id"
                  data-testid="referral-app-revoke"
                  @click="openAction(app, 'revoke')"
                >
                  {{ actingId === app.id && actingType === 'revoke' ? '...' : '取消资格' }}
                </button>
                <button
                  class="px-3 py-1 text-xs border border-gray-300 rounded hover:bg-gray-50"
                  data-testid="referral-app-audit"
                  @click="openAudit(app)"
                >
                  审计
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="actionTarget"
      class="app-modal-overlay z-50 flex items-center justify-center bg-black/30"
      @click.self="closeAction"
    >
      <div class="bg-white rounded-xl shadow-xl p-6 w-full max-w-md">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">{{ actionTitle() }}</h3>
        <p class="text-sm text-gray-500 mb-3">
          用户 <code class="bg-gray-100 px-1 rounded">{{ actionTarget.user_id }}</code>
        </p>
        <label class="block text-sm font-medium text-gray-700 mb-1">操作理由（必填，8–500 字）</label>
        <textarea
          v-model="actionReason"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm mb-2"
          rows="3"
          data-testid="referral-action-reason"
          placeholder="填写理由以便审计..."
        ></textarea>
        <p v-if="actionReasonError" class="text-xs text-red-500 mb-3">{{ actionReasonError }}</p>
        <div class="flex justify-end gap-3">
          <button
            class="px-4 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50"
            @click="closeAction"
          >
            取消
          </button>
          <button
            class="px-4 py-2 text-sm bg-primary text-white rounded-lg hover:opacity-90 disabled:opacity-50"
            :disabled="actingId === actionTarget.id"
            data-testid="referral-action-confirm"
            @click="submitAction"
          >
            {{ actingId === actionTarget.id ? '处理中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>

    <div
      v-if="auditTarget"
      class="app-modal-overlay z-50 flex items-center justify-center bg-black/30"
      @click.self="auditTarget = null"
    >
      <div class="bg-white rounded-xl shadow-xl p-6 w-full max-w-lg" data-testid="referral-audit-drawer">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">操作审计</h3>
        <p v-if="auditLoading" class="text-sm text-gray-500">加载中...</p>
        <p v-else-if="!auditItems.length" class="text-sm text-gray-500">暂无记录</p>
        <ul v-else class="space-y-3 max-h-80 overflow-y-auto text-sm">
          <li v-for="row in auditItems" :key="row.id" class="border-b pb-2">
            <div class="font-medium">{{ row.action }} · {{ row.operator_id }}</div>
            <div class="text-gray-500 text-xs">{{ formatDate(row.created_at) }}</div>
            <div class="mt-1">{{ row.reason }}</div>
          </li>
        </ul>
        <div class="flex justify-end mt-4">
          <button
            class="px-4 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50"
            @click="auditTarget = null"
          >
            关闭
          </button>
        </div>
      </div>
    </div>

    <div
      v-if="appError"
      class="mt-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm"
      :data-traceId="appErrorTraceId || undefined"
    >
      {{ appError }}
    </div>

    <div v-if="totalApps > limit" class="mt-4 flex items-center justify-between text-sm">
      <span class="text-gray-500">共 {{ totalApps }} 条</span>
      <div class="flex gap-2">
        <button
          class="px-3 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
          :disabled="offset === 0"
          @click="prevPage"
        >
          上一页
        </button>
        <button
          class="px-3 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
          :disabled="offset + limit >= totalApps"
          @click="nextPage"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useReferralApplications } from '../composables/useReferralApplications.js'

const copiedCode = ref('')

// OPT-20260824-007：反查命中用户在列表时滚动定位对应行
const rowEls = {}
const setRowRef = (userId, el) => {
  if (el) rowEls[userId] = el
  else delete rowEls[userId]
}

const copyShareCode = async (code) => {
  try {
    await navigator.clipboard.writeText(code)
    copiedCode.value = code
    setTimeout(() => {
      if (copiedCode.value === code) copiedCode.value = ''
    }, 1500)
  } catch {
    // clipboard API 在非安全上下文可能不可用；静默降级（码仍可见可手选）
  }
}

const {
  loadingApps,
  appError,
  appErrorTraceId,
  applications,
  totalApps,
  statusFilter,
  limit,
  offset,
  actingId,
  actingType,
  actionTarget,
  actionReason,
  actionReasonError,
  auditTarget,
  auditItems,
  auditLoading,
  statusClass,
  formatDate,
  loadApplications,
  openAction,
  closeAction,
  submitAction,
  openAudit,
  prevPage,
  nextPage,
  actionTitle,
  codeQuery,
  codeSearching,
  codeResult,
  codeError,
  codeErrorTraceId,
  highlightUserId,
  lookupShareCode,
} = useReferralApplications()

watch(highlightUserId, (id) => {
  if (id && rowEls[id]) {
    try {
      rowEls[id].scrollIntoView?.({ behavior: 'smooth', block: 'center' })
    } catch {
      // jsdom / 无 scrollIntoView 环境静默降级，高亮仍可见
    }
  }
})

onMounted(() => {
  loadApplications()
})
</script>
