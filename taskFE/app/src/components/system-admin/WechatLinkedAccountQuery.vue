<template>
  <div class="bg-white border border-gray-200 rounded-lg p-6 mb-6" data-testid="wechat-linked-account-box">
    <label class="block text-sm font-medium text-gray-700 mb-1" for="wechat-linked-account">微信关联账号查询</label>
    <p class="text-xs text-gray-500 mb-3">
      输入微信昵称、openid、unionid，或已绑定的手机号、邮箱、用户名，精确查找，无需先选租户。
    </p>
    <div class="flex gap-2">
      <input
        id="wechat-linked-account"
        v-model="accountInput"
        type="text"
        class="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm"
        placeholder="例如微信昵称、openid 或绑定手机号"
        data-testid="wechat-linked-account-input"
        @keydown.enter="submitSearch"
      >
      <!-- Anti-Replay-OK: 只读列表查询；同步门闩防连点，无 Idempotency-Key -->
      <button
        type="button"
        class="px-4 py-2 bg-gray-800 text-white text-sm rounded-lg hover:bg-gray-700 disabled:opacity-50"
        :disabled="searching"
        :aria-busy="searching ? 'true' : 'false'"
        data-testid="wechat-linked-account-search"
        @click="submitSearch"
      >
        {{ searching ? '查询中...' : '查询' }}
      </button>
    </div>
    <p
      v-if="accountError"
      class="mt-2 text-sm text-red-700"
      data-testid="wechat-linked-account-error"
    >{{ accountError }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { createClickGuard } from '../../utils/clickGuard.js'

defineProps({
  searching: { type: Boolean, default: false },
})

const emit = defineEmits(['search'])

const accountInput = ref('')
const accountError = ref('')
const searchGuard = createClickGuard()

const submitSearch = async () => {
  await searchGuard.run(async () => {
    const raw = String(accountInput.value || '').trim()
    accountError.value = ''
    if (!raw) {
      accountError.value = '请输入微信关联账号'
      return
    }
    emit('search', raw)
  })
}
</script>
