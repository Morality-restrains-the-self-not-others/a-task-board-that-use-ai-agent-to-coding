<template>
  <div
    :id="formId"
    ref="rootEl"
    class="sticky top-0 z-10 p-3 border border-gray-200 rounded-lg bg-gray-50 space-y-2 shadow-sm"
    :data-testid="`${testIdPrefix}-share-form`"
  >
    <p class="text-sm text-gray-700">
      为订单 {{ orderLabel }} 向微信发起分账。缘由将记入审计。
    </p>
    <label class="block text-xs text-gray-500" :for="reasonId">分账缘由</label>
    <textarea
      :id="reasonId"
      ref="reasonEl"
      :value="modelValue"
      :data-testid="`${testIdPrefix}-share-reason`"
      class="w-full text-sm border border-gray-300 rounded px-2 py-1.5"
      rows="3"
      maxlength="500"
      placeholder="至少 8 个字，说明为何由超管发起"
      @input="$emit('update:modelValue', $event.target.value)"
    />
    <div class="flex gap-2">
      <button
        type="button"
        class="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-white"
        :data-testid="`${testIdPrefix}-share-cancel`"
        :disabled="shareBusy"
        @click="$emit('cancel')"
      >
        取消
      </button>
      <button
        type="button"
        class="px-3 py-1 text-sm bg-gray-900 text-white rounded disabled:opacity-40"
        :data-testid="`${testIdPrefix}-share-confirm`"
        :disabled="shareBusy"
        :aria-busy="shareBusy ? 'true' : 'false'"
        @click="$emit('confirm')"
      >
        {{ shareBusy ? '提交中…' : '确认分账' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'

const props = defineProps({
  orderLabel: { type: String, required: true },
  modelValue: { type: String, default: '' },
  shareBusy: { type: Boolean, default: false },
  formId: { type: String, required: true },
  testIdPrefix: { type: String, required: true },
})

defineEmits(['update:modelValue', 'confirm', 'cancel'])

const rootEl = ref(null)
const reasonEl = ref(null)
const reasonId = computed(() => `${props.testIdPrefix}-share-reason`)

onMounted(() => {
  nextTick(() => {
    const form = rootEl.value
    if (form && typeof form.scrollIntoView === 'function') {
      form.scrollIntoView({ block: 'nearest', inline: 'nearest' })
    }
    const input = reasonEl.value
    if (input && typeof input.focus === 'function') {
      input.focus()
    }
  })
})
</script>
