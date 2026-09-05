<template>
  <span
    v-if="items.length"
    class="inline-flex items-center gap-1 max-w-full min-w-0"
    data-testid="comment-execution-git-identity-wrap"
  >
    <span
      v-for="item in items"
      :key="item.id"
      class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-violet-50 text-violet-900 border border-violet-100 max-w-[14rem] truncate"
      :title="item.title"
      data-testid="comment-execution-git-identity"
      :data-git-identity-id="item.id"
    >{{ item.text }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { commentGitIdentitySummaryItems } from '../../utils/commentExecutionGitIdentity.js'

const props = defineProps({
  repoIdentities: { type: Array, default: () => [] },
  gitIdentityOptions: { type: Array, default: () => [] },
  /** 评论 repo_identities 为空时回退展示的任务级仓库身份（OPT-20260821-028） */
  fallbackRepoIdentities: { type: Array, default: () => [] },
})

const items = computed(() =>
  commentGitIdentitySummaryItems(
    props.repoIdentities,
    props.gitIdentityOptions,
    props.fallbackRepoIdentities,
  ),
)
</script>
