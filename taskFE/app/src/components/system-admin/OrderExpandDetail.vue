<template>
  <div>
    <div v-if="loading" class="text-sm text-gray-400 py-2">加载中…</div>
    <template v-else>
      <div v-if="items.length" class="space-y-2">
        <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">订单行项明细</h4>
        <table class="w-full text-xs">
          <thead>
            <tr class="text-left text-gray-500">
              <th class="pb-1 pr-4 font-medium">资源类型</th>
              <th class="pb-1 pr-4 text-right font-medium">数量</th>
              <th class="pb-1 pr-4 text-right font-medium">单价（元）</th>
              <th class="pb-1 text-right font-medium">小计（元）</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in items" :key="it.id">
              <td class="py-1 pr-4 text-gray-800">{{ resourceTypeLabel(it.resource_type) }}</td>
              <td class="py-1 pr-4 text-right text-gray-700">{{ it.quantity }}</td>
              <td class="py-1 pr-4 text-right font-mono text-gray-600">{{ it.unit_price_yuan }}</td>
              <td class="py-1 text-right font-mono text-gray-900">{{ it.subtotal_yuan }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="text-sm text-gray-400 py-2">无行项数据</div>
      <p
        v-if="buyerNote"
        class="text-sm whitespace-pre-wrap mt-3"
        data-testid="admin-order-buyer-note"
      >施工留言：{{ buyerNote }}</p>
      <dl
        v-if="outTradeNo || wechatTransactionId"
        class="mt-3 text-sm space-y-1"
        data-testid="admin-order-wechat-vouchers"
      >
        <div v-if="outTradeNo" class="flex gap-2">
          <dt class="text-gray-500 shrink-0">{{ outTradeNoLabel }}</dt>
          <dd class="font-mono text-gray-900 break-all" data-testid="admin-order-out-trade-no">{{ outTradeNo }}</dd>
        </div>
        <div v-if="wechatTransactionId" class="flex gap-2">
          <dt class="text-gray-500 shrink-0">{{ wechatTransactionIdLabel }}</dt>
          <dd class="font-mono text-gray-900 break-all" data-testid="admin-order-wechat-transaction-id">{{ wechatTransactionId }}</dd>
        </div>
      </dl>
      <div class="mt-3">
        <OrderResourceConsumption :consumption="consumption" />
      </div>
      <div v-if="Array.isArray(profitSharing)" class="mt-4" data-testid="admin-order-profit-sharing">
        <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">分账</h4>
        <table v-if="profitSharing.length" class="w-full text-xs">
          <thead>
            <tr class="text-left text-gray-500">
              <th class="pb-1 pr-4 font-medium">分账接收方</th>
              <th class="pb-1 pr-4 text-right font-medium">金额（元）</th>
              <th class="pb-1 pr-4 font-medium">AppID / OpenID</th>
              <th class="pb-1 text-right font-medium">状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, idx) in profitSharing" :key="row.receiver_user_id + '-' + idx">
              <td class="py-1 pr-4 text-gray-800 font-mono">{{ row.receiver_display || row.receiver_user_id }}</td>
              <td class="py-1 pr-4 text-right font-mono text-gray-900">{{ row.amount_yuan }}</td>
              <td class="py-1 pr-4 text-gray-700 font-mono">
                {{ row.app_id || '—' }}<span v-if="row.app_id"> / </span>{{ row.openid || '—' }}
              </td>
              <td class="py-1 text-right text-gray-700">{{ profitSharingStatusLabel(row.status) }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="text-sm text-gray-400 py-2">本订单无分账记录</p>
      </div>
    </template>
  </div>
</template>

<script setup>
import { billingResourceTypeLabel as resourceTypeLabel } from '@/utils/billingResourceTypeLabel.js'
import OrderResourceConsumption from '@/components/OrderResourceConsumption.vue'

defineProps({
  loading: { type: Boolean, default: false },
  items: { type: Array, default: () => [] },
  buyerNote: { type: String, default: '' },
  outTradeNo: { type: String, default: '' },
  wechatTransactionId: { type: String, default: '' },
  // 管理端默认文案；租户端可传「商户单号 / 交易单号」与订单详情摘要保持一致
  outTradeNoLabel: { type: String, default: '商户订单号' },
  wechatTransactionIdLabel: { type: String, default: '微信支付单号' },
  consumption: { type: Object, default: null },
  // null：租户端不渲染；数组：管理端展示（可空）
  profitSharing: { type: Array, default: null },
})

const STATUS_LABEL = {
  pending: '待分账',
  processing: '分账中',
  finished: '已完成',
  failed: '失败',
}

function profitSharingStatusLabel(status) {
  const key = String(status || '').trim()
  return STATUS_LABEL[key] || key || '—'
}
</script>
