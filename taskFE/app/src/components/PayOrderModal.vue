<template>
  <div class="app-modal-overlay z-50 flex items-center justify-center bg-black/50 p-4" @click.self="$emit('close')">
    <!-- OPT-20260726-034: 支付前手机号验证门禁 — 策略开启且用户未验证时先展示验证 -->
    <div class="bg-white rounded-lg max-w-sm w-full p-6 shadow-xl" v-if="showPhoneGate">
      <h3 class="text-lg font-semibold mb-4">微信扫码支付</h3>
      <p class="text-sm text-gray-600 mb-2">
        订单号：<strong class="break-all">{{ order?.order_number }}</strong>
      </p>
      <p class="text-sm text-gray-600 mb-4">
        金额：<strong class="text-primary">{{ order?.total_yuan }} 元</strong>
      </p>
      <PhoneVerificationGate
        :tenant-id="tenantId"
        :active="showPhoneGate"
        :phone-status="phoneVerificationStatus"
        @verified="$emit('phone-verified')"
      />
      <div class="mt-4">
        <button
          type="button"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
          @click="$emit('close')"
        >
          关闭
        </button>
      </div>
    </div>

    <!-- 支付服务条款签署门禁 -->
    <div
      v-else-if="showPaymentTermsGate"
      class="bg-white rounded-lg max-w-lg w-full p-6 shadow-xl"
      data-testid="payment-terms-consent-gate"
    >
      <h3 class="text-lg font-semibold mb-2">支付服务条款</h3>
      <p class="text-sm text-gray-600 mb-3">
        支付前请阅读并同意《{{ paymentTermsDoc?.title || '支付服务条款协议' }}》
        <span v-if="paymentTermsDoc?.version" class="text-gray-400">（{{ paymentTermsDoc.version }}）</span>
      </p>
      <div class="max-h-56 overflow-y-auto rounded border border-gray-200 bg-gray-50 p-3 text-xs text-gray-700 whitespace-pre-wrap mb-4">
        {{ paymentTermsDoc?.content || '正在加载条款…' }}
      </div>
      <p
        v-if="pollingErr"
        class="text-sm text-red-500 mb-3"
        :data-traceId="pollingErrTraceId || undefined"
      >{{ pollingErr }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          class="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
          :disabled="paymentTermsConsenting"
          @click="$emit('close')"
        >
          取消
        </button>
        <button
          type="button"
          class="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 text-sm disabled:opacity-50"
          data-testid="payment-terms-accept-btn"
          :disabled="paymentTermsConsenting || !paymentTermsDoc?.id"
          @click="$emit('payment-terms-accepted')"
        >
          {{ paymentTermsConsenting ? '提交中…' : '同意并继续支付' }}
        </button>
      </div>
    </div>

    <!-- QR Code Payment (shown when no phone verification needed or after verification) -->
    <div v-else class="bg-white rounded-lg max-w-sm w-full p-6 shadow-xl">
      <h3 class="text-lg font-semibold mb-4">微信扫码支付</h3>
      <p class="text-sm text-gray-600 mb-2">
        订单号：<strong class="break-all">{{ order?.order_number }}</strong>
      </p>
      <p class="text-sm text-gray-600 mb-4">
        金额：<strong class="text-primary">{{ order?.total_yuan }} 元</strong>
      </p>
      <!-- OPT-20260726-019/028: 本地 Canvas 渲染 QR 码 -->
      <div v-if="codeUrl" class="flex justify-center mb-4" data-testid="wechat-pay-qr">
        <div class="bg-white border border-gray-200 rounded p-2">
          <canvas ref="qrCanvas" width="200" height="200"></canvas>
        </div>
      </div>
      <p v-if="pollingErr" class="text-sm text-red-500 text-center mb-3" :data-traceId="pollingErrTraceId || undefined">{{ pollingErr }}</p>
      <p class="text-xs text-gray-400 text-center mb-4">请使用微信扫描二维码完成支付</p>
      <div class="flex gap-2">
        <button
          type="button"
          class="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
          @click="$emit('close')"
        >
          关闭
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import PhoneVerificationGate from './PhoneVerificationGate.vue'
import { renderQrToCanvas } from '../composables/useQrCodeCanvas'

const props = defineProps({
  order: { type: Object, default: null },
  codeUrl: { type: String, default: '' },
  mode: { type: String, default: '' },
  pollingErr: { type: String, default: '' },
  pollingErrTraceId: { type: String, default: '' },
  // OPT-20260726-034: 手机号验证门禁 props
  tenantId: { type: String, default: '' },
  phoneGateActive: { type: Boolean, default: false },
  phoneVerificationStatus: {
    type: Object,
    default: () => ({
      has_phone: false,
      phone_masked: '',
      phone_e164: '',
      sms_verified: false,
      required: false,
    }),
  },
  paymentTermsGateActive: { type: Boolean, default: false },
  paymentTermsDoc: { type: Object, default: null },
  paymentTermsConsenting: { type: Boolean, default: false },
})

defineEmits(['close', 'phone-verified', 'payment-terms-accepted'])

const showPhoneGate = computed(() => props.phoneGateActive)
const showPaymentTermsGate = computed(() => props.paymentTermsGateActive)

const qrCanvas = ref(null)

const renderQR = async () => {
  if (!props.codeUrl || !qrCanvas.value) return
  await renderQrToCanvas(qrCanvas.value, props.codeUrl, { width: 200 })
}

watch(
  [qrCanvas, () => props.codeUrl, () => props.phoneGateActive, () => props.paymentTermsGateActive],
  () => {
    if (props.phoneGateActive || props.paymentTermsGateActive) return
    renderQR()
  },
  { flush: 'post' },
)
onMounted(() => { renderQR() })
</script>
