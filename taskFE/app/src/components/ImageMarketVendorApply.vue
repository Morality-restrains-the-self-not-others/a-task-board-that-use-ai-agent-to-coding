<template>
  <div class="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-4" data-testid="vendor-app-form">
    <h3 class="text-sm font-semibold text-gray-900">{{ heading }}</h3>
    <p v-if="rejectNote" class="mt-1 text-sm text-red-600">上次驳回原因：{{ rejectNote }}</p>
    <p v-if="!hasEmail" class="mt-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
      当前账号没有可投递邮箱（微信扫码为合成邮箱），须先绑定邮箱才能提交申请。
      <a
        :href="emailBindingHref"
        class="ml-1 font-medium text-primary underline"
      >去绑定邮箱</a>
    </p>
    <p class="mt-1 text-xs text-gray-500">提交后进入审核；通过前不会出现厂商门户 SSO。</p>

    <label class="mt-3 block text-xs font-medium text-gray-700">公司名称</label>
    <input v-model.trim="companyName" type="text" maxlength="255" class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" placeholder="请输入公司名称（必填）">
    <label class="mt-3 block text-xs font-medium text-gray-700">联系人</label>
    <input v-model.trim="contactName" type="text" maxlength="255" class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" placeholder="请输入联系人姓名（必填）">
    <label class="mt-3 block text-xs font-medium text-gray-700">身份证</label>
    <input type="file" accept="image/jpeg,image/png,image/webp,application/pdf" class="mt-1 text-sm" @change="onFileChange($event, 'id_card')">
    <p v-if="idCardKey" class="text-xs text-green-700">已上传</p>
    <label class="mt-3 block text-xs font-medium text-gray-700">营业执照</label>
    <input type="file" accept="image/jpeg,image/png,image/webp,application/pdf" class="mt-1 text-sm" @change="onFileChange($event, 'business_license')">
    <p v-if="licenseKey" class="text-xs text-green-700">已上传</p>

    <div class="mt-3 rounded-lg border border-amber-200 bg-amber-50 p-3">
      <p class="text-xs text-amber-800">申请前需完成手机号短信验证。</p>
      <p v-if="phoneVerified" class="mt-1 text-xs text-green-700">已验证 {{ phoneMasked || contactPhone }}</p>
      <template v-else>
        <p v-if="hasBoundPhone" class="mt-1 text-xs text-gray-600">已绑定手机：{{ phoneMasked || '—' }}</p>
        <input
          v-else
          v-model.trim="phoneDraft"
          type="tel"
          class="mt-2 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm"
          placeholder="+86 手机号"
        >
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <button type="button" class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm" :disabled="sendingSms" :aria-busy="sendingSms ? 'true' : 'false'" @click="sendSms">
            {{ sendingSms ? '发送中…' : '发送验证码' }}
          </button>
          <input v-model="smsCode" type="text" maxlength="6" class="w-28 rounded-lg border border-gray-200 px-3 py-1.5 text-sm" placeholder="6 位验证码">
          <button type="button" class="rounded-lg bg-primary px-3 py-1.5 text-sm text-white disabled:opacity-60" :disabled="verifyingSms || smsCode.trim().length !== 6" :aria-busy="verifyingSms ? 'true' : 'false'" @click="verifySms">
            {{ verifyingSms ? '验证中…' : '验证' }}
          </button>
        </div>
      </template>
    </div>

    <div class="mt-4 flex flex-wrap items-center gap-2">
      <button
        type="button"
        class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
        data-testid="vendor-apply-submit"
        :disabled="!canSubmit"
        :aria-busy="submitting ? 'true' : 'false'"
        @click="submit"
      >
        {{ submitting ? '提交中…' : '提交申请' }}
      </button>
      <!-- Anti-Replay-OK: pure UI collapse; no network -->
      <button
        type="button"
        class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700"
        data-testid="vendor-apply-cancel"
        @click="emit('cancel')"
      >
        取消
      </button>
    </div>
    <p v-if="banner" class="mt-2 text-sm text-red-600" :data-traceId="bannerTraceId || undefined">{{ banner }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch, extractErrorMessage, safeParseResponse } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { uploadIssuedFile } from '../utils/vendorDocsDirectUpload.js'

const props = defineProps({
  rejectNote: { type: String, default: '' },
  heading: { type: String, default: '申请成为厂商门户' },
  hasEmail: { type: Boolean, default: true },
  emailBindingHref: { type: String, default: '/profile/?sso_error=email_required#rg=profile.email_binding' },
})
const emit = defineEmits(['applied', 'cancel'])

const companyName = ref('')
const contactName = ref('')
const idCardKey = ref('')
const licenseKey = ref('')
const phoneVerified = ref(false)
const contactPhone = ref('')
const phoneMasked = ref('')
const hasBoundPhone = ref(false)
const smsCode = ref('')
const phoneDraft = ref('+86')
const uploading = ref(false)
const submitting = ref(false)
const sendingSms = ref(false)
const verifyingSms = ref(false)
const banner = ref('')
const bannerTraceId = ref('')

const submitGuard = createClickGuard()
const smsGuard = createClickGuard()
const verifyGuard = createClickGuard()
const uploadGuard = createClickGuard()

const canSubmit = computed(() =>
  props.hasEmail &&
  !submitting.value &&
  !uploading.value &&
  !!companyName.value &&
  !!contactName.value &&
  !!idCardKey.value &&
  !!licenseKey.value &&
  phoneVerified.value &&
  !!contactPhone.value,
)

function setBanner(message, source) {
  banner.value = message || ''
  bannerTraceId.value = source?.traceId || ''
}

async function jsonCall(url, options = {}) {
  const response = await apiFetch(url, { credentials: 'include', ...options })
  return safeParseResponse(response)
}

async function loadPhoneStatus() {
  try {
    const d = await jsonCall('/api/ai-provider/vendor-application/phone-status/')
    hasBoundPhone.value = !!d.has_phone
    phoneMasked.value = d.phone_masked || ''
    if (d.sms_verified && d.phone_e164) {
      phoneVerified.value = true
      contactPhone.value = d.phone_e164
    } else if (d.phone_e164) {
      contactPhone.value = d.phone_e164
    }
  } catch {
    /* 门禁仍展示 */
  }
}

async function onFileChange(ev, kind) {
  const file = ev?.target?.files?.[0]
  if (!file) return
  await uploadGuard.run(async ({ idempotencyKey }) => {
    uploading.value = true
    setBanner('')
    try {
      const issued = await jsonCall('/api/ai-provider/vendor-application/upload-url/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          kind,
          filename: file.name,
          content_type: file.type,
          size: file.size,
        }),
      })
      await uploadIssuedFile(file, issued, { fetchImpl: fetch })
      await jsonCall('/api/ai-provider/vendor-application/upload-complete/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ kind, file_key: issued.file_key }),
      })
      if (kind === 'id_card') idCardKey.value = issued.file_key || ''
      else licenseKey.value = issued.file_key || ''
    } catch (e) {
      setBanner(e.message || '上传失败', e)
      if (kind === 'id_card') idCardKey.value = ''
      else licenseKey.value = ''
    } finally {
      uploading.value = false
    }
  })
}

async function sendSms() {
  await smsGuard.run(async ({ idempotencyKey }) => {
    const phone = hasBoundPhone.value ? contactPhone.value : String(phoneDraft.value || '').trim()
    if (!phone) {
      setBanner('请输入手机号')
      return
    }
    sendingSms.value = true
    setBanner('')
    try {
      await jsonCall('/api/ai-provider/vendor-application/send-sms/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ phone }),
      })
      contactPhone.value = phone
    } catch (e) {
      setBanner(e.message || extractErrorMessage(e.response?._errorData, e.response, '发送失败'), e)
    } finally {
      sendingSms.value = false
    }
  })
}

async function verifySms() {
  await verifyGuard.run(async ({ idempotencyKey }) => {
    const phone = contactPhone.value || String(phoneDraft.value || '').trim()
    verifyingSms.value = true
    setBanner('')
    try {
      const d = await jsonCall('/api/ai-provider/vendor-application/verify-phone/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ phone, code: smsCode.value.trim() }),
      })
      phoneVerified.value = true
      contactPhone.value = phone
      if (d?.phone_masked) phoneMasked.value = d.phone_masked
    } catch (e) {
      setBanner(e.message || '验证失败', e)
    } finally {
      verifyingSms.value = false
    }
  })
}

async function submit() {
  if (!canSubmit.value) return
  await submitGuard.run(async ({ idempotencyKey }) => {
    submitting.value = true
    setBanner('')
    try {
      await jsonCall('/api/ai-provider/vendor-application/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          company_name: companyName.value,
          contact_name: contactName.value,
          id_card_file_key: idCardKey.value,
          business_license_file_key: licenseKey.value,
          contact_phone: contactPhone.value,
        }),
      })
      emit('applied')
    } catch (e) {
      setBanner(e.message || '提交申请失败', e)
    } finally {
      submitting.value = false
    }
  })
}

onMounted(loadPhoneStatus)
</script>
