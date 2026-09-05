<template>
  <select
    :value="modelValue"
    class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
    data-testid="invite-expiration-days"
    @change="$emit('update:modelValue', Number($event.target.value))"
  >
    <option v-for="d in INVITE_EXPIRATION_DAY_OPTIONS" :key="d" :value="d">{{ d }}天</option>
  </select>
</template>

<script setup>
import {
  DEFAULT_INVITE_EXPIRATION_DAYS,
  INVITE_EXPIRATION_DAY_OPTIONS,
} from '../utils/peopleInviteExpiration.js'

// 邀请有效期下拉共用控件：复制邀请链接 / 邮箱 / 电话三种邀请方式统一选项与默认值。
defineProps({
  modelValue: { type: [Number, String], default: DEFAULT_INVITE_EXPIRATION_DAYS },
})
defineEmits(['update:modelValue'])
</script>
