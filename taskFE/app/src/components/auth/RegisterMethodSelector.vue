<template>
  <div class="register-method-selector">
    <div class="flex border-b border-gray-200">
      <button 
        type="button" 
        class="py-2 px-4 font-medium" 
        data-testid="register-method-phone"
        :class="{
          'text-primary border-b-2 border-primary': currentMethod === 'phone',
          'text-gray-500 hover:text-primary': currentMethod !== 'phone'
        }"
        @click="switchMethod('phone')"
      >
        手机号注册
      </button>
      <button
        v-if="allowEmail"
        type="button" 
        class="py-2 px-4 font-medium" 
        data-testid="register-method-email"
        :class="{
          'text-primary border-b-2 border-primary': currentMethod === 'email',
          'text-gray-500 hover:text-primary': currentMethod !== 'email'
        }"
        @click="switchMethod('email')"
      >
        邮箱注册
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: 'email'
  },
  allowEmail: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['update:modelValue', 'method-changed'])

const currentMethod = ref(props.modelValue)

watch(() => props.modelValue, (newValue) => {
  currentMethod.value = newValue
})

watch(
  () => props.allowEmail,
  (allowed) => {
    if (!allowed && currentMethod.value === 'email') {
      switchMethod('phone')
    }
  }
)

const switchMethod = (method) => {
  if (method === 'email' && !props.allowEmail) {
    return
  }
  currentMethod.value = method
  emit('update:modelValue', method)
  emit('method-changed', method)
}
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>
