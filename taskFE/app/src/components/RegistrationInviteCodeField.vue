<template>
  <div v-if="enabled" data-testid="registration-invite-code-field" class="space-y-1">
    <label :for="inputId" class="block text-sm font-medium text-gray-700">
      邀请码<span class="text-red-500">*</span>
    </label>
    <input
      :id="inputId"
      :value="modelValue"
      type="text"
      autocomplete="off"
      maxlength="16"
      class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/40 uppercase tracking-wider font-mono"
      :placeholder="placeholder"
      data-testid="registration-invite-code-input"
      @input="onInput"
    >
    <p v-if="hint" class="text-xs text-gray-500">{{ hint }}</p>
    <p
      v-if="errorMessage"
      class="text-sm text-red-600"
      data-testid="registration-invite-code-error"
      :data-traceId="errorTraceId || undefined"
    >
      {{ errorMessage }}
    </p>
  </div>
</template>

<script setup>
/* @alias:comp-registration-invite-code-field */
defineProps({
  modelValue: { type: String, default: '' },
  enabled: { type: Boolean, default: false },
  inputId: { type: String, default: 'registration-invite-code' },
  placeholder: { type: String, default: '请输入邀请码' },
  hint: { type: String, default: '' },
  errorMessage: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const onInput = (event) => {
  emit('update:modelValue', String(event?.target?.value || '').toUpperCase())
}
</script>
