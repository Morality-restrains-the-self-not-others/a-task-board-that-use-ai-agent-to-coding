<template>
  <div class="rounded-lg border border-gray-200 bg-gray-50/50 px-4 py-3 space-y-2">
    <label class="flex items-start gap-2 cursor-pointer select-none">
      <input
        :checked="acceptAll"
        data-testid="login-accept-all"
        type="checkbox"
        class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
        @change="onToggleAll($event.target.checked)"
      />
      <span class="text-sm font-medium text-text">
        我已阅读并同意全部条款
      </span>
    </label>
    <p v-if="privacyLoadError" class="text-sm text-amber-800">{{ privacyLoadError }}</p>
    <label class="flex items-start gap-2 cursor-pointer select-none">
      <input
        :checked="acceptPrivacy"
        data-testid="login-privacy-accept"
        type="checkbox"
        class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
        @change="emit('update:acceptPrivacy', $event.target.checked)"
      />
      <span class="text-sm text-text">
        我已阅读并同意
        <button
          type="button"
          class="text-primary font-medium hover:underline"
          @click="emit('open-privacy')"
        >
          《隐私政策》
        </button>
        <span v-if="currentPrivacyPolicy" class="text-text-light">（{{ currentPrivacyPolicy.version }}）</span>
      </span>
    </label>
    <p v-if="licenseLoadError" class="text-sm text-amber-800">{{ licenseLoadError }}</p>
    <label class="flex items-start gap-2 cursor-pointer select-none">
      <input
        :checked="acceptLicense"
        data-testid="login-license-accept"
        type="checkbox"
        class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
        @change="emit('update:acceptLicense', $event.target.checked)"
      />
      <span class="text-sm text-text">
        我已阅读并同意
        <button
          type="button"
          class="text-primary font-medium hover:underline"
          @click="emit('open-license')"
        >
          《软件许可及服务协议》
        </button>
        <span v-if="currentLicenseAgreement" class="text-text-light">（{{ currentLicenseAgreement.version }}）</span>
      </span>
    </label>
  </div>
</template>

<script setup>
defineProps({
  acceptAll: { type: Boolean, default: false },
  acceptPrivacy: { type: Boolean, default: false },
  acceptLicense: { type: Boolean, default: false },
  privacyLoadError: { type: String, default: '' },
  licenseLoadError: { type: String, default: '' },
  currentPrivacyPolicy: { type: Object, default: null },
  currentLicenseAgreement: { type: Object, default: null },
})

const emit = defineEmits([
  'update:acceptPrivacy',
  'update:acceptLicense',
  'update:acceptAll',
  'open-privacy',
  'open-license',
])

function onToggleAll(checked) {
  emit('update:acceptAll', checked)
}
</script>
