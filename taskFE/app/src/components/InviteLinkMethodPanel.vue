<template>
  <div class="space-y-4">
    <div>
      <label class="block text-sm font-medium text-text mb-1">链接类型</label>
      <div class="grid grid-cols-2 gap-3">
        <label
          class="flex items-center p-3 border border-border rounded-lg cursor-pointer transition-colors"
          :class="{'border-primary bg-primary/5': linkKind === 'single'}"
        >
          <input
            type="radio"
            :checked="linkKind === 'single'"
            class="mr-2"
            data-testid="invite-link-kind-single"
            @change="$emit('update:linkKind', 'single')"
          >
          <span>单次（一人）</span>
        </label>
        <label
          class="flex items-center p-3 border border-border rounded-lg cursor-pointer transition-colors"
          :class="{'border-primary bg-primary/5': linkKind === 'open'}"
        >
          <input
            type="radio"
            :checked="linkKind === 'open'"
            class="mr-2"
            data-testid="invite-link-kind-open"
            @change="$emit('update:linkKind', 'open')"
          >
          <span>开放（多人）</span>
        </label>
      </div>
    </div>
    <div>
      <label class="block text-sm font-medium text-text mb-1">{{ linkKind === 'open' ? '邀请备注（可选）' : '成员名称' }}</label>
      <input
        type="text"
        :value="companyMemberName"
        class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
        :required="linkKind !== 'open'"
        data-testid="invite-link-member-name"
        @input="$emit('update:companyMemberName', $event.target.value)"
      >
    </div>
    <div v-if="linkKind === 'open'">
      <label class="block text-sm font-medium text-text mb-1">最多人数（留空不限制）</label>
      <input
        type="number"
        min="1"
        :value="maxUses"
        class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
        data-testid="invite-link-max-uses"
        placeholder="不限制"
        @input="$emit('update:maxUses', $event.target.value)"
      >
    </div>
    <div>
      <label class="block text-sm font-medium text-text mb-1">有效期</label>
      <div class="grid grid-cols-12 gap-4">
        <div class="col-span-8">
          <InviteExpirationSelect
            :model-value="expirationDays"
            @update:model-value="$emit('update:expirationDays', $event)"
          />
        </div>
        <div class="col-span-4">
          <button
            type="button"
            class="w-full bg-primary text-white px-4 py-2 border border-primary rounded-lg hover:bg-primary/90 transition-colors flex items-center justify-center"
            :disabled="isLoading || !workspaceId || (linkKind !== 'open' && !companyMemberName)"
            data-testid="invite-link-generate-btn"
            @click="$emit('generate')"
          >
            <span v-if="isLoading">生成中...</span>
            <span v-else>生成邀请链接</span>
          </button>
        </div>
      </div>
    </div>
    <div>
      <label class="block text-sm font-medium text-text mb-1">邀请链接</label>
      <div class="flex">
        <input
          type="text"
          :value="inviteLink"
          readonly
          class="flex-1 px-4 py-2 border border-border rounded-l-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
        >
        <button
          type="button"
          class="bg-primary text-white px-4 py-2 border border-primary rounded-r-lg hover:bg-primary/90 transition-colors flex items-center"
          :disabled="isLoading || !inviteToken"
          @click="$emit('copy')"
        >
          <span v-if="isLoading">复制中...</span>
          <span v-else-if="copySuccess">已复制</span>
          <span v-else>复制</span>
        </button>
      </div>
    </div>
    <div class="bg-info/10 border border-info text-info rounded-lg p-4">
      <p class="text-sm">复制以上邀请链接并发送给您想要邀请的人，他们可以通过该链接加入您的团队。</p>
      <p class="text-sm mt-2">{{ hintText }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { DEFAULT_INVITE_EXPIRATION_DAYS } from '../utils/peopleInviteExpiration.js'
import InviteExpirationSelect from './InviteExpirationSelect.vue'

const props = defineProps({
  companyMemberName: { type: String, default: '' },
  expirationDays: { type: [Number, String], default: DEFAULT_INVITE_EXPIRATION_DAYS },
  workspaceId: { type: String, default: '' },
  isLoading: { type: Boolean, default: false },
  inviteLink: { type: String, default: '' },
  inviteToken: { type: String, default: '' },
  copySuccess: { type: Boolean, default: false },
  linkKind: { type: String, default: 'single' },
  maxUses: { type: [Number, String], default: '' },
})
defineEmits([
  'update:companyMemberName',
  'update:expirationDays',
  'update:linkKind',
  'update:maxUses',
  'generate',
  'copy',
])

const hintText = computed(() => {
  const days = props.expirationDays
  if (props.linkKind === 'open') {
    const cap = String(props.maxUses || '').trim()
    if (cap) {
      return `开放链接有效期为${days}天，最多 ${cap} 人加入。`
    }
    return `开放链接有效期为${days}天，人数不限，直到过期或被撤销。`
  }
  return `邀请链接有效期为${days}天，且只能使用一次。`
})
</script>
