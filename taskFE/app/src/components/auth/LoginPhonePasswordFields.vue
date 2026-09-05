<template>
  <div>
    <div>
      <label for="login-phone-national-pwd" class="block text-sm font-medium text-text-light mb-2">手机号</label>
      <div class="flex rounded-lg border border-gray-300 focus-within:ring-2 focus-within:ring-primary relative">
        <div class="relative z-30 shrink-0 border-r border-gray-200 bg-gray-50" :title="countryErrorHint || ''">
          <select
            :value="loginPhonePrefix"
            class="block w-full min-w-[6.5rem] max-h-48 py-2.5 pl-2 pr-8 text-sm text-gray-700 bg-transparent border-0 appearance-none cursor-pointer"
            aria-label="国家或地区代码"
            @change="emit('update:loginPhonePrefix', $event.target.value)"
          >
            <option
              v-for="c in countryDialOptions"
              :key="c.code"
              :value="c.code"
            >
              {{ c.code }} {{ c.name }}
            </option>
          </select>
          <span v-if="countryLoading" class="absolute right-1 top-1/2 -translate-y-1/2 flex items-center" aria-label="加载区域限制中" title="区域限制加载中">
            <svg class="animate-spin h-3.5 w-3.5 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
          </span>
        </div>
        <input
          id="login-phone-national-pwd"
          :value="phoneNationalPassword"
          type="tel"
          name="phone_national"
          autocomplete="tel-national"
          required
          class="flex-1 min-w-0 px-3 py-2.5 border-0 text-sm focus:outline-none bg-white"
          :placeholder="nationalPlaceholder"
          :maxlength="maxNationalLen"
          inputmode="numeric"
          @input="emit('national-pwd-input', $event)"
          @paste="emit('national-pwd-paste', $event)"
        >
      </div>
      <p class="mt-1 text-xs text-text-light">默认区号 +86；可粘贴含 +86 / 86 前缀的完整号码。</p>
    </div>
    <div>
      <label for="password" class="block text-sm font-medium text-text-light mb-2">密码</label>
      <div class="relative">
        <input
          :type="showPhonePassword ? 'text' : 'password'"
          id="password"
          name="password"
          autocomplete="current-password"
          required
          class="input w-full pr-16"
          placeholder="请输入密码"
        >
        <button
          type="button"
          class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-primary hover:text-primary-dark"
          @click="emit('update:showPhonePassword', !showPhonePassword)"
        >
          {{ showPhonePassword ? '隐藏' : '显示' }}
        </button>
      </div>
    </div>

    <div class="flex items-center justify-between">
      <label for="remember-me" class="inline-flex items-center gap-2 px-2 py-1 -ml-2 rounded-md cursor-pointer select-none hover:bg-gray-50">
        <input
          id="remember-me"
          name="remember-me"
          type="checkbox"
          class="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
        >
        <span class="block text-sm text-text">记住我</span>
      </label>
      <div class="text-sm">
        <a href="/auth/reset-password-request/" class="font-medium text-primary hover:text-primary-dark transition-colors duration-300">忘记密码？</a>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  loginPhonePrefix: { type: String, required: true },
  phoneNationalPassword: { type: String, required: true },
  showPhonePassword: { type: Boolean, default: false },
  countryDialOptions: { type: Array, required: true },
  countryLoading: { type: Boolean, default: false },
  countryErrorHint: { type: String, default: '' },
  nationalPlaceholder: { type: String, required: true },
  maxNationalLen: { type: Number, required: true },
})

// 模板事件透传依赖 emit 返回值：此前裸 defineEmits([...]) 调用导致 _ctx.emit 为
// undefined，任何输入/点击抛 "TypeError: i.emit is not a function" → 父组件
// phoneNationalPassword 永不更新 → 登录按钮恒禁用（OPT-20260810-007 修复）
const emit = defineEmits([
  'update:loginPhonePrefix',
  'update:phoneNationalPassword',
  'update:showPhonePassword',
  'national-pwd-input',
  'national-pwd-paste',
])
</script>
