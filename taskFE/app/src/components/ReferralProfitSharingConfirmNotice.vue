<template>
  <div>
    <p
      v-if="pendingCount > 0"
      class="text-xs text-amber-800 leading-relaxed rounded-lg border border-amber-200 bg-amber-50 px-3 py-2"
      data-testid="referral-profit-sharing-confirm-notice"
      :data-pending-count="pendingCount"
    >
      <strong>待确认 {{ pendingCount }} 笔</strong
      >{{ deadline ? `，最晚 ${deadline} 前打开微信确认` : '' }}。微信支付向个人分账时，会向您的微信下发确认通知。请在通知所示<strong>有效期内</strong>打开微信并<strong>点击确认</strong>；逾期未确认则该笔分账关闭，资金退回平台，<strong>无法补分</strong>。
    </p>
    <p
      v-else
      class="text-xs text-amber-800 leading-relaxed rounded-lg border border-amber-200 bg-amber-50 px-3 py-2"
      data-testid="referral-profit-sharing-confirm-notice"
    >
      微信支付向个人分账时，会向您的微信下发确认通知。请在通知所示<strong>有效期内</strong>打开微信并<strong>点击确认</strong>；逾期未确认则该笔分账关闭，资金退回平台，<strong>无法补分</strong>。请留意微信「服务通知」。
    </p>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'

const PENDING_URL = '/api/billing/profit-sharing/referrer-pending/'

const pendingCount = ref(0)
const deadline = ref('')

async function loadPending() {
  try {
    const resp = await apiFetch(PENDING_URL, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) return
    pendingCount.value = Number(data.pending_count || 0)
    deadline.value = String(data.deadline || '').trim()
  } catch (error) {
    console.warn('[ReferralProfitSharingConfirmNotice] load pending failed', error)
    pendingCount.value = 0
    deadline.value = ''
  }
}

onMounted(loadPending)
</script>
