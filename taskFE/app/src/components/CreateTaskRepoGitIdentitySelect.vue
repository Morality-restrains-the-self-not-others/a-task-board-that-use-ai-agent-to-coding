<template>
  <div
    v-if="visible"
    class="mt-2 space-y-1"
    data-testid="create-task-repo-git-identity-row"
  >
    <label class="block text-xs font-medium text-gray-700">Git 提交身份</label>
    <select
      class="block w-full px-2 py-1.5 text-xs border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-primary focus:border-primary"
      data-testid="create-task-repo-git-identity-select"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option value="">请选择用于该仓库提交的身份</option>
      <option
        v-for="identity in identities"
        :key="identity.id"
        :value="identity.id"
      >
        {{ formatLabel(identity) }}
      </option>
    </select>
    <a
      v-if="settingsHref"
      class="inline-block text-xs text-primary underline underline-offset-2"
      data-testid="create-task-git-identity-settings-link"
      :href="settingsHref"
    >去账号中心管理 Git 提交身份</a>
  </div>
</template>

<script setup>
import { formatGitIdentityOptionLabel } from '../utils/createTaskGitIdentityGate.js'

defineProps({
  visible: { type: Boolean, default: false },
  modelValue: { type: String, default: '' },
  identities: { type: Array, default: () => [] },
  settingsHref: { type: String, default: '' },
})

defineEmits(['update:modelValue'])

const formatLabel = formatGitIdentityOptionLabel
</script>
