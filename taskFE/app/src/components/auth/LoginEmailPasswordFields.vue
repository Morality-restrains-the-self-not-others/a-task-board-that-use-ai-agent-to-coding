<template>
  <div>
    <div>
      <label for="email" class="block text-sm font-medium text-text-light mb-2">邮箱</label>
      <input
        id="email"
        type="email"
        name="email"
        autocomplete="email"
        required
        class="input w-full"
        placeholder="请输入邮箱"
      />
    </div>
    <div>
      <label for="password" class="block text-sm font-medium text-text-light mb-2">密码</label>
      <div class="relative">
        <input
          id="password"
          :type="props.showEmailPassword ? 'text' : 'password'"
          name="password"
          autocomplete="current-password"
          required
          class="input w-full pr-16"
          placeholder="请输入密码"
        />
        <button
          type="button"
          class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-primary hover:text-primary-dark"
          @click="emit('update:showEmailPassword', !props.showEmailPassword)"
        >
          {{ props.showEmailPassword ? '隐藏' : '显示' }}
        </button>
      </div>
    </div>

    <div class="flex items-center justify-between">
      <label
        for="remember-me"
        class="inline-flex items-center gap-2 px-2 py-1 -ml-2 rounded-md cursor-pointer select-none hover:bg-gray-50"
      >
        <input
          id="remember-me"
          name="remember-me"
          type="checkbox"
          class="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
        />
        <span class="block text-sm text-text">记住我</span>
      </label>
      <div class="text-sm">
        <a
          href="/auth/reset-password-request/"
          class="font-medium text-primary hover:text-primary-dark transition-colors duration-300"
        >
          忘记密码？
        </a>
      </div>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  showEmailPassword: { type: Boolean, default: false },
})

const emit = defineEmits(['update:showEmailPassword'])
</script>
