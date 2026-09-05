<template>
  <div
    v-if="showPrivacyReader"
    class="fixed inset-0 z-[10040] flex items-end sm:items-center justify-center bg-black/50 p-0 sm:p-4"
    @click.self="$emit('close-privacy')"
  >
    <div
      class="w-full sm:max-w-2xl max-h-[85vh] flex flex-col rounded-t-2xl sm:rounded-2xl bg-white shadow-2xl"
      role="dialog"
      aria-modal="true"
    >
      <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
        <h2 class="text-base font-semibold text-gray-900">
          {{ currentPrivacyPolicy?.title || '隐私政策' }}
        </h2>
        <button type="button" class="text-sm text-primary" @click="$emit('close-privacy')">
          关闭
        </button>
      </div>
      <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-800 whitespace-pre-wrap">
        <p v-if="!currentPrivacyPolicy" class="text-gray-500">正在加载或暂无内容…</p>
        <template v-else>{{ currentPrivacyPolicy.content }}</template>
      </div>
    </div>
  </div>

  <div
    v-if="showLicenseReader"
    class="fixed inset-0 z-[10040] flex items-end sm:items-center justify-center bg-black/50 p-0 sm:p-4"
    @click.self="$emit('close-license')"
  >
    <div
      class="w-full sm:max-w-2xl max-h-[85vh] flex flex-col rounded-t-2xl sm:rounded-2xl bg-white shadow-2xl"
      role="dialog"
      aria-modal="true"
    >
      <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
        <h2 class="text-base font-semibold text-gray-900">
          {{ currentLicenseAgreement?.title || '软件许可及服务协议' }}
        </h2>
        <button type="button" class="text-sm text-primary" @click="$emit('close-license')">
          关闭
        </button>
      </div>
      <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-800 whitespace-pre-wrap">
        <p v-if="!currentLicenseAgreement" class="text-gray-500">正在加载或暂无内容…</p>
        <template v-else>{{ currentLicenseAgreement.content }}</template>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  showPrivacyReader: { type: Boolean, default: false },
  showLicenseReader: { type: Boolean, default: false },
  currentPrivacyPolicy: { type: Object, default: null },
  currentLicenseAgreement: { type: Object, default: null },
})

defineEmits(['close-privacy', 'close-license'])
</script>
