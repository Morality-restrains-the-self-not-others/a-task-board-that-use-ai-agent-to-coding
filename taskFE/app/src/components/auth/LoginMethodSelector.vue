<template>
  <div class="login-method-selector">
    <div class="mb-6">
      <div class="flex rounded-lg overflow-hidden border border-gray-200 bg-gray-50/70 p-1">
        <button 
          type="button" 
          @click="switchMethod('emailPassword')"
          :class="[
            'flex-1 py-2.5 px-3 sm:px-4 text-xs sm:text-sm font-medium rounded-md transition-all duration-200',
            currentMethod === 'emailPassword' 
              ? 'bg-primary text-white shadow-sm scale-[1.01]' 
              : 'bg-transparent text-text-light hover:bg-white'
          ]"
        >
          邮箱/密码
        </button>
        <button
          v-if="allowPhoneLogin"
          type="button" 
          @click="switchMethod('phonePassword')"
          :class="[
            'flex-1 py-2.5 px-3 sm:px-4 text-xs sm:text-sm font-medium rounded-md transition-all duration-200',
            currentMethod === 'phonePassword' 
              ? 'bg-primary text-white shadow-sm scale-[1.01]' 
              : 'bg-transparent text-text-light hover:bg-white'
          ]"
        >
          手机号/密码
        </button>
        <button
          type="button"
          @click="switchMethod('accessToken')"
          :class="[
            'flex-1 py-2.5 px-3 sm:px-4 text-xs sm:text-sm font-medium rounded-md transition-all duration-200',
            currentMethod === 'accessToken'
              ? 'bg-primary text-white shadow-sm scale-[1.01]'
              : 'bg-transparent text-text-light hover:bg-white'
          ]"
        >
          访问令牌
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: 'emailPassword'
  },
  allowPhoneLogin: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'method-changed'])

const currentMethod = ref(props.modelValue)

// 监听props变化，更新本地状态
watch(() => props.modelValue, (newValue) => {
  currentMethod.value = newValue
})

// 切换登录方式
const switchMethod = (method) => {
  currentMethod.value = method
  emit('update:modelValue', method)
  emit('method-changed', method)
}
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>
