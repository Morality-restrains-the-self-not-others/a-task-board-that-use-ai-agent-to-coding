<template>
  <div
    class="p-2 bg-white border border-gray-200 rounded-md"
    data-testid="task-linked-projects-panel"
  >
    <TaskDetailLinkedProjectsToolbar
      :is-editing="isEditing"
      :stale-repo-sync-error="staleRepoSyncError"
      :stale-repo-sync-error-trace-id="staleRepoSyncErrorTraceId"
      :default-image-label="defaultImageLabel"
      :default-image-id="defaultImageId"
      :default-image-removed="defaultImageRemoved"
    />
    <TaskDetailLinkedProjectsViewMode
      v-if="!isEditing"
      :tenant-id="tenantId"
      :task-projects-with-details="taskProjectsWithDetails"
      :fallback-api-projects="fallbackApiProjects"
      :workspace-projects="workspaceProjects"
      :stale-repo-sync-loading="staleRepoSyncLoading"
      :stale-repo-sync-needs-reclone="staleRepoSyncNeedsReclone"
      @sync-repo-address="emit('sync-repo-address')"
    />
    <TaskDetailLinkedProjectsEditMode
      v-if="isEditing"
      :editing-task="editingTask"
      :workspace-projects="workspaceProjects"
      :workspace-projects-loading="workspaceProjectsLoading"
      :get-project-repos="getProjectRepos"
      :get-repo-branch-value="getRepoBranchValue"
      :get-repo-branches="getRepoBranches"
      :get-repo-branch-error="getRepoBranchError"
      :is-loading-repo-branches="isLoadingRepoBranches"
      :repo-clone-field-id="repoCloneFieldId"
      :gitOAuthCatalogVersion="gitOAuthCatalogVersion"
      :editRepoOAuthActionLoadingByUrl="editRepoOAuthActionLoadingByUrl"
      :shouldShowEditRepoOAuthAuthorize="shouldShowEditRepoOAuthAuthorize"
      :startEditRepoOAuthConnect="startEditRepoOAuthConnect"
      :normalize-repo-url-key="normalizeRepoUrlKey"
      @project-change="emit('project-change', $event)"
      @set-repo-branch="(project, repoUrl, value) => emit('set-repo-branch', project, repoUrl, value)"
      @fetch-repo-branches="(projectId, repoUrl, force) => emit('fetch-repo-branches', projectId, repoUrl, force)"
      @add-project-association="emit('add-project-association')"
    />
  </div>
</template>

<script setup>
import { useLinkedProjectsRepoOAuth } from '../../composables/taskDetail/useLinkedProjectsRepoOAuth.js'
import TaskDetailLinkedProjectsToolbar from './TaskDetailLinkedProjectsToolbar.vue'
import TaskDetailLinkedProjectsViewMode from './TaskDetailLinkedProjectsViewMode.vue'
import TaskDetailLinkedProjectsEditMode from './TaskDetailLinkedProjectsEditMode.vue'

const props = defineProps({
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  isEditing: { type: Boolean, required: true },
  taskProjectsWithDetails: { type: Array, required: true },
  fallbackApiProjects: { type: Array, default: () => [] },
  editingTask: { type: Object, required: true },
  workspaceProjects: { type: Array, required: true },
  workspaceProjectsLoading: { type: Boolean, required: true },
  getProjectRepos: { type: Function, required: true },
  getRepoBranchValue: { type: Function, required: true },
  getRepoBranches: { type: Function, required: true },
  getRepoBranchError: { type: Function, required: true },
  isLoadingRepoBranches: { type: Function, required: true },
  repoCloneFieldId: { type: Function, default: () => '' },
  staleRepoSyncLoading: { type: Boolean, default: false },
  staleRepoSyncError: { type: String, default: '' },
  staleRepoSyncErrorTraceId: { type: String, default: '' },
  staleRepoSyncNeedsReclone: { type: Boolean, default: false },
  gitOAuthCatalogVersion: { type: Number, default: 0 },
  onReadinessChange: { type: Function, default: null },
  defaultImageLabel: { type: String, default: '' },
  defaultImageId: { type: String, default: '' },
  defaultImageRemoved: { type: Boolean, default: false },
})

const emit = defineEmits([
  'project-change',
  'remove-project-association',
  'set-repo-branch',
  'fetch-repo-branches',
  'add-project-association',
  'sync-repo-address',
  'repo-oauth-readiness',
])

const {
  normalizeRepoUrlKey,
  editRepoOAuthActionLoadingByUrl,
  shouldShowEditRepoOAuthAuthorize,
  startEditRepoOAuthConnect,
} = useLinkedProjectsRepoOAuth({ props, emit })

defineExpose({
  refreshGithubRepoBindingStatus: () => {},
})
</script>
