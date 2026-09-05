<template>
  <div class="mt-4 pt-4 border-t border-green-200" data-testid="order-invoice-section">
    <h3 class="text-sm font-medium text-green-800 mb-2">电子发票</h3>
    <!-- Anti-Replay-OK: display-only 冲红 72h 确认提醒 -->
    <div
      v-if="reverseConfirm"
      class="mb-3 p-3 rounded-md text-sm border"
      :class="reverseConfirm.expired ? 'bg-red-50 text-red-800 border-red-200' : 'bg-amber-50 text-amber-900 border-amber-200'"
      data-testid="order-invoice-reverse-confirm-hint"
    >
      {{ reverseConfirm.message }}
      <span v-if="reverseConfirm.deadline" class="block mt-1 text-xs opacity-80">确认截止：{{ reverseConfirm.deadline }}</span>
    </div>
    <p
      v-if="pending"
      class="text-sm text-orange-600"
      data-testid="order-invoice-pending-label"
    >开票处理中，请耐心等待</p>
    <button
      v-else-if="canApply"
      type="button"
      class="px-3 py-1.5 text-sm font-medium text-green-800 border border-green-300 rounded-md hover:bg-green-100"
      data-testid="order-invoice-apply-btn"
      :disabled="guardBusy"
      :aria-busy="guardBusy ? 'true' : 'false'"
      @click="openApply"
    >
              <!-- Anti-Replay-OK: ui-only 打开开票抬头弹层 -->
              申请开票
    </button>
    <p
      v-else-if="zeroAmount"
      class="text-sm text-gray-600"
      data-testid="order-invoice-zero-amount-label"
    >
      <!-- Anti-Replay-OK: display-only 零额订单不可开票说明 -->
      订单金额为 0 元，无法申请开票
    </p>
    <p
      v-else-if="notWechatChannel"
      class="text-sm text-gray-600"
      data-testid="order-invoice-non-wechat-label"
    >
      <!-- Anti-Replay-OK: display-only 非微信渠道不可开票说明（与后端 applyInvoiceApplication 对齐） -->
      仅微信支付订单可申请开票
    </p>
    <p v-else-if="hasIssuedBlue" class="text-sm text-green-700">已开具电子发票</p>

    <ul v-if="invoices.length" class="mt-3 space-y-1" data-testid="order-invoice-list">
      <li
        v-for="inv in invoices"
        :key="inv.id"
        class="text-sm text-gray-700"
      >
        {{ purposeLabel(inv) }} · {{ inv.amount_yuan }} 元 · {{ statusLabel(inv.status) }}
        <span v-if="inv.invoice_type" class="text-xs text-gray-500">{{ inv.invoice_type === 'special' ? '专票' : '普票' }}</span>
        <span v-if="inv.wechat_fapiao_number" class="font-mono text-xs text-gray-500">{{ inv.wechat_fapiao_number }}</span>
        <a
          v-if="inv.invoice_file_url"
          :href="inv.invoice_file_url"
          target="_blank"
          rel="noopener"
          class="ml-2 text-blue-600 underline"
          data-testid="order-invoice-file-link"
        >查看发票文件</a>
      </li>
    </ul>
    <p
      v-if="sectionError"
      class="mt-2 text-sm text-red-700"
      :data-traceId="sectionErrorTraceId || undefined"
    >{{ sectionError }}</p>

    <div
      v-if="applyOpen"
      class="app-modal-overlay z-50 flex items-center justify-center bg-black/50 p-4"
      data-testid="order-invoice-apply-modal"
      @click.self="closeApply"
    >
      <div class="bg-white rounded-lg max-w-md w-full p-6 shadow-xl">
        <h3 class="text-lg font-semibold text-gray-900 mb-3">申请开具电子发票</h3>
        <p class="text-sm text-gray-500 mb-4">提交后由平台管理员手动开具电子发票。专票仅支持企业抬头，须填写完整单位信息。</p>
        <label class="block text-sm font-medium text-gray-700 mb-1">发票类型</label>
        <select
          v-model="invoiceType"
          class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
          data-testid="order-invoice-type-select"
        >
          <option value="general">增值税普通发票（普票）</option>
          <option value="special">增值税专用发票（专票）</option>
        </select>
        <label class="block text-sm font-medium text-gray-700 mb-1">抬头类型</label>
        <select
          v-model="buyerType"
          :disabled="invoiceType === 'special'"
          class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3 disabled:bg-gray-100"
        >
          <option value="INDIVIDUAL">个人</option>
          <option value="ORGANIZATION">企业</option>
        </select>
        <label class="block text-sm font-medium text-gray-700 mb-1">发票抬头</label>
        <input
          v-model.trim="buyerName"
          class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
          placeholder="姓名或公司全称"
        >
        <template v-if="buyerType === 'ORGANIZATION'">
          <label class="block text-sm font-medium text-gray-700 mb-1">纳税人识别号</label>
          <input
            v-model.trim="taxpayerId"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
            placeholder="统一社会信用代码"
          >
        </template>
        <template v-if="invoiceType === 'special'">
          <label class="block text-sm font-medium text-gray-700 mb-1">注册地址</label>
          <input
            v-model.trim="buyerAddress"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
            placeholder="单位注册地址"
            data-testid="order-invoice-address"
          >
          <label class="block text-sm font-medium text-gray-700 mb-1">注册电话</label>
          <input
            v-model.trim="buyerTelephone"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
            placeholder="单位联系电话"
            data-testid="order-invoice-telephone"
          >
          <label class="block text-sm font-medium text-gray-700 mb-1">开户银行</label>
          <input
            v-model.trim="bankName"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
            placeholder="开户银行全称"
            data-testid="order-invoice-bank-name"
          >
          <label class="block text-sm font-medium text-gray-700 mb-1">银行账号</label>
          <input
            v-model.trim="bankAccount"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
            placeholder="银行账号"
            data-testid="order-invoice-bank-account"
          >
        </template>
        <p v-if="applyError" class="mb-3 text-sm text-red-700" :data-traceId="applyErrorTraceId || undefined">{{ applyError }}</p>
        <div class="flex justify-end gap-3">
          <button type="button" class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg" :disabled="applying" @click="closeApply">取消</button>
          <button
            type="button"
            data-testid="order-invoice-apply-confirm"
            class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
            :disabled="!canSubmit || applying"
            :aria-busy="applying ? 'true' : 'false'"
            @click="submitApply"
          >{{ applying ? '提交中…' : '提交申请' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils'
import { createClickGuard } from '../utils/clickGuard.js'
import { extractTraceId } from '../utils/traceId.js'
import { billingOrderIsWechatChannel } from '../utils/billingOrderDisplay.js'

const props = defineProps({
  tenantId: { type: String, required: true },
  order: { type: Object, default: null },
})
const emit = defineEmits(['updated'])

const applyOpen = ref(false)
const applying = ref(false)
const applyError = ref('')
const applyErrorTraceId = ref('')
const sectionError = ref('')
const sectionErrorTraceId = ref('')
const invoiceType = ref('general')
const buyerType = ref('INDIVIDUAL')
const buyerName = ref('')
const taxpayerId = ref('')
const buyerAddress = ref('')
const buyerTelephone = ref('')
const bankName = ref('')
const bankAccount = ref('')
watch(invoiceType, (v) => {
  if (v === 'special') buyerType.value = 'ORGANIZATION'
})
const guard = createClickGuard()
const guardBusy = ref(false)

const invoices = computed(() => Array.isArray(props.order?.invoices) ? props.order.invoices : [])
const appStatus = computed(() => props.order?.invoice_application?.status || '')
const pending = computed(() => appStatus.value === 'pending')
const hasIssuedBlue = computed(() => invoices.value.some((i) => i.kind === 'blue' && (i.status === 'issued' || i.status === 'issuing')))
const orderTotalYuanCents = computed(() => {
  const order = props.order
  if (!order) return null
  const rawCents = order.total_yuan_cents
  if (rawCents !== undefined && rawCents !== null && rawCents !== '') {
    const n = Number(rawCents)
    if (Number.isFinite(n)) return n
  }
  const rawYuan = order.total_yuan
  if (rawYuan !== undefined && rawYuan !== null && rawYuan !== '') {
    const n = Number(rawYuan)
    if (Number.isFinite(n)) return Math.round(n * 100)
  }
  return null
})
const zeroAmount = computed(() => {
  const cents = orderTotalYuanCents.value
  return cents !== null && cents <= 0
})
const reverseConfirm = computed(() => {
  const c = props.order?.invoice_reverse_confirm
  if (c && c.required) return c
  const pendingInv = invoices.value.find((i) => i.status === 'reverse_pending' || i.status === 'reverse_expired')
  if (!pendingInv) return null
  const expired = pendingInv.status === 'reverse_expired' || pendingInv.reverse_confirm_expired === true
  return {
    required: true,
    hours: pendingInv.reverse_confirm_hours || 72,
    deadline: pendingInv.reverse_confirm_deadline,
    expired,
    message: expired
      ? '冲红确认已超过72小时，冲红可能已失效，请联系平台处理'
      : '请在微信卡包于72小时内确认冲红，逾期冲红将失效',
  }
})
const notWechatChannel = computed(
  () =>
    props.order?.status === 'paid' &&
    !pending.value &&
    !hasIssuedBlue.value &&
    !zeroAmount.value &&
    !billingOrderIsWechatChannel(props.order),
)
const canApply = computed(() => props.order?.status === 'paid' && !pending.value && !hasIssuedBlue.value && !zeroAmount.value && billingOrderIsWechatChannel(props.order))
const canSubmit = computed(() => {
  if (!buyerName.value) return false
  if (buyerType.value === 'ORGANIZATION' && !taxpayerId.value) return false
  if (invoiceType.value === 'special') {
    return buyerType.value === 'ORGANIZATION' &&
      !!(taxpayerId.value && buyerAddress.value && buyerTelephone.value && bankName.value && bankAccount.value)
  }
  return true
})

const purposeLabel = (inv) => {
  if (inv.kind === 'red') return '红字发票（冲红）'
  if (inv.purpose === 'reissue') return '蓝字发票（重开）'
  return '蓝字发票'
}
const statusLabel = (s) => ({
  issuing: '开具中',
  issued: '已开具',
  reverse_pending: '待确认冲红',
  reverse_expired: '冲红确认已超时',
  reversed: '已冲红',
  failed: '失败',
}[s] || s)

const openApply = () => {
  applyError.value = ''
  applyErrorTraceId.value = ''
  invoiceType.value = 'general'
  buyerType.value = 'INDIVIDUAL'
  buyerName.value = ''
  taxpayerId.value = ''
  buyerAddress.value = ''
  buyerTelephone.value = ''
  bankName.value = ''
  bankAccount.value = ''
  applyOpen.value = true
}
const closeApply = () => {
  if (applying.value) return
  applyOpen.value = false
}

const submitApply = async () => {
  const result = await guard.run(async ({ idempotencyKey }) => {
    applying.value = true
    guardBusy.value = true
    applyError.value = ''
    applyErrorTraceId.value = ''
    try {
      const body = {
        invoice_type: invoiceType.value,
        type: buyerType.value,
        name: buyerName.value,
      }
      if (buyerType.value === 'ORGANIZATION') body.taxpayer_id = taxpayerId.value
      if (invoiceType.value === 'special') {
        body.address = buyerAddress.value
        body.telephone = buyerTelephone.value
        body.bank_name = bankName.value
        body.bank_account = bankAccount.value
      }
      const r = await apiFetch(`/api/tenant/${props.tenantId}/billing/orders/${props.order.id}/invoice-applications/`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify(body),
      })
      const data = await r.json().catch(() => ({}))
      if (!r.ok) {
        applyError.value = extractErrorMessage(data, r) || data.error || '申请失败'
        applyErrorTraceId.value = extractTraceId(r) || extractTraceId(data) || ''
        return false
      }
      applyOpen.value = false
      emit('updated')
      return true
    } catch (e) {
      applyError.value = e.message || '申请失败'
      applyErrorTraceId.value = extractTraceId(e) || ''
      return false
    } finally {
      applying.value = false
      guardBusy.value = false
    }
  })
  if (result.skipped) return
}

defineExpose({ sectionError, sectionErrorTraceId })
</script>
