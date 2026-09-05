<template>
  <div
    v-if="visible"
    class="space-y-2"
    data-testid="create-task-git-identity-section"
  >
    <div
      v-for="url in repoUrls"
      :key="url"
      class="rounded-md border border-gray-200 p-2"
    >
      <p class="text-xs text-gray-600 break-all">{{ url }}</p>
      <CreateTaskRepoGitIdentitySelect
        :visible="true"
        :model-value="gitIdentityIdForUrl(url)"
        :identities="gitIdentities"
        :settings-href="gitIdentitySettingsHref"
        @update:model-value="(value) => setGitIdentityIdForUrl(url, value)"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import CreateTaskRepoGitIdentitySelect from './CreateTaskRepoGitIdentitySelect.vue'
import { useCreateTaskGitIdentities } from '../composables/useCreateTaskGitIdentities.js'
import { collectCreateTaskRepoUrls } from '../utils/createTaskGitIdentityGate.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  projects: {
    type: Array,
    default: () => [],
  },
  tenantId: {
    type: [String, Number],
    default: null,
  },
  show: {
    type: Boolean,
    default: false,
  },
})

const repoUrls = computed(() => collectCreateTaskRepoUrls(props.editingTask, props.projects))
const visible = computed(() => Boolean(props.editingTask?.auto_run) && repoUrls.value.length > 0)

const {
  gitIdentities,
  settingsHref: gitIdentitySettingsHref,
  gitIdentityIdForUrl,
  setGitIdentityIdForUrl,
} = useCreateTaskGitIdentities({
  editingTask: () => props.editingTask,
  projects: () => props.projects,
  tenantId: () => props.tenantId,
  enabled: () => Boolean(props.show && props.editingTask?.auto_run),
})
</script>
