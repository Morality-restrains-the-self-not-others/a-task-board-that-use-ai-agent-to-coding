<template>
  <div>
    <div class="mb-4">
      <label for="access-token-username" class="block text-sm font-medium text-text-light mb-2">账号</label>
      <input
        type="text"
        id="access-token-username"
        :value="accessTokenUsernameInput"
        autocomplete="username"
        class="input w-full"
        placeholder="用户名或邮箱"
        @input="emit('update:accessTokenUsernameInput', $event.target.value)"
      >
    </div>
    <div>
      <label for="access-token-input" class="block text-sm font-medium text-text-light mb-2">访问令牌</label>
      <div class="relative">
        <input
          :type="showAccessToken ? 'text' : 'password'"
          id="access-token-input"
          :value="accessTokenInput"
          autocomplete="off"
          class="input w-full pr-16"
          placeholder="请输入您的访问令牌"
          @input="emit('update:accessTokenInput', $event.target.value)"
        >
        <button
          type="button"
          class="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-primary hover:text-primary-dark"
          @click="emit('update:showAccessToken', !showAccessToken)"
        >
          {{ showAccessToken ? '隐藏' : '显示' }}
        </button>
      </div>
      <p class="mt-2 text-xs text-text-light">
        请填写与令牌对应的账号，并在「个人资料 → 访问令牌」创建令牌。令牌仅创建时完整展示一次，请妥善保管。
      </p>
    </div>
  </div>
</template>

<script setup>
defineProps({
  accessTokenUsernameInput: { type: String, default: '' },
  accessTokenInput: { type: String, default: '' },
  showAccessToken: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:accessTokenUsernameInput',
  'update:accessTokenInput',
  'update:showAccessToken',
])
</script>
