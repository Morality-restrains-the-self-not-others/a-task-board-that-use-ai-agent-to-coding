<template>
  <div>
    <label for="password" class="block text-sm font-medium text-text-light mb-2">密码</label>
    <input
      id="password"
      :value="modelValue"
      type="password"
      name="password"
      placeholder="请输入密码（至少8位，包含字母、数字和特殊字符）"
      required
      class="input w-full"
      @input="onInput"
    >
    <div id="password-error" class="text-error text-sm mt-1" :class="{ hidden: !error }">{{ error }}</div>
    <div id="password-strength" class="mt-2" :class="{ hidden: !modelValue }">
      <div class="flex items-center mb-1">
        <span class="text-xs font-medium text-gray-600 mr-2">密码强度：</span>
        <span id="strength-text" class="text-xs font-medium" :class="strengthTextClass">{{ passwordStrength.level }}</span>
      </div>
      <div class="w-full bg-gray-200 rounded-full h-2">
        <div
          id="strength-bar"
          class="h-2 rounded-full transition-all duration-300"
          :class="strengthBarClass"
          :style="{ width: `${passwordStrength.strength * 20}%` }"
        ></div>
      </div>
      <div id="password-requirements" class="text-xs text-gray-500 mt-1">
        <div v-for="(req, index) in passwordStrength.requirements" :key="index" class="flex items-center">
          {{ req.text }} <span class="ml-1">{{ req.status }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:comp-register-password-field */
import { computed, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  error: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const passwordStrength = ref({ strength: 0, level: '', requirements: [] })

const strengthTextClass = computed(() => {
  if (passwordStrength.value.level === '弱') return 'text-red-600'
  if (passwordStrength.value.level === '中') return 'text-yellow-600'
  if (passwordStrength.value.level === '强') return 'text-green-600'
  return ''
})

const strengthBarClass = computed(() => {
  if (passwordStrength.value.level === '弱') return 'bg-red-500'
  if (passwordStrength.value.level === '中') return 'bg-yellow-500'
  if (passwordStrength.value.level === '强') return 'bg-green-500'
  return ''
})

const checkPasswordStrength = (password) => {
  if (!password) {
    passwordStrength.value = { strength: 0, level: '', requirements: [] }
    return
  }
  let strength = 0
  const requirements = []
  const checks = [
    [password.length >= 8, '至少8位字符'],
    [/[a-z]/.test(password), '包含小写字母'],
    [/[A-Z]/.test(password), '包含大写字母'],
    [/[0-9]/.test(password), '包含数字'],
    [/[^A-Za-z0-9]/.test(password), '包含特殊字符'],
  ]
  for (const [ok, text] of checks) {
    if (ok) strength += 1
    requirements.push({ text, status: ok ? '✅' : '❌' })
  }
  let level = '强'
  if (strength <= 2) level = '弱'
  else if (strength <= 4) level = '中'
  passwordStrength.value = { strength, level, requirements }
}

const onInput = (event) => {
  const value = String(event?.target?.value || '')
  emit('update:modelValue', value)
  checkPasswordStrength(value)
}

watch(
  () => props.modelValue,
  (v) => checkPasswordStrength(v),
  { immediate: true }
)
</script>
