<template>
  <div class="email-register">
    <div>
      <label for="email" class="block text-sm font-medium text-text-light mb-2">邮箱</label>
      <input 
        type="email" 
        id="email" 
        :value="modelValue.email" 
        name="email" 
        placeholder="请输入邮箱地址" 
        required 
        class="input w-full"
        @input="handleEmailInput"
      >
      <div id="email-error" class="text-error text-sm mt-1" :class="{ 'hidden': !errors.email }" :data-traceId="traceId || undefined">{{ errors.email }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      email: '',
      password: ''
    })
  },
  errors: {
    type: Object,
    default: () => ({
      email: ''
    })
  },
  /** 本次提交 API 失败 traceId，挂到字段级错误节点供 trace-first 排障（OPT-20260821-025） */
  traceId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'update:errors'])

// 处理邮箱输入
const handleEmailInput = (event) => {
  const email = event.target.value
  const newFormData = { ...props.modelValue, email }
  emit('update:modelValue', newFormData)
  validateEmail(email)
}

// 验证邮箱
const validateEmail = (email) => {
  const trimmedEmail = email.trim()
  if (trimmedEmail && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail)) {
    emit('update:errors', { ...props.errors, email: '请输入正确的邮箱地址' })
  } else {
    emit('update:errors', { ...props.errors, email: '' })
  }
}
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>