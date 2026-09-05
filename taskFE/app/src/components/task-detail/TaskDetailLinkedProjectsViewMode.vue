<template>
  <div>
    <div
      v-if="displayProjects.length > 0"
      class="space-y-2"
      data-testid="task-linked-projects-display"
    >
      <div
        v-for="tp in displayProjects"
        :key="tp.id"
        class="px-2 py-1.5 bg-gray-50 rounded border border-gray-100/80"
      >
        <div class="flex flex-wrap items-center gap-1.5 mb-1">
          <span
            v-if="tp.project_missing"
            class="inline-flex items-center px-1.5 py-0 rounded-full text-[11px] leading-tight bg-gray-100 text-gray-600 border border-gray-200/80"
            data-testid="task-linked-project-missing"
            title="工作区项目目录中没有这条记录，详情页无法打开"
          >
            项目已删除
          </span>
          <!-- Anti-Replay-OK: navigation -->
          <router-link
            v-else-if="tp.project_id && tenantId"
            :to="`/tenant/${tenantId}/projects/${tp.project_id}/`"
            class="inline-flex items-center px-1.5 py-0 rounded-full text-[11px] leading-tight bg-blue-50 text-blue-700 border border-blue-200/80 hover:bg-blue-100 hover:underline cursor-pointer"
            data-testid="task-linked-project-name-link"
            title="查看项目详情"
          >
            {{ tp.project_name }}
          </router-link>
          <span
            v-else
            class="inline-flex items-center px-1.5 py-0 rounded-full text-[11px] leading-tight bg-blue-50 text-blue-700 border border-blue-200/80"
          >
            {{ tp.project_name }}
          </span>
          <span
            v-if="tp.repo_address_mismatch"
            class="inline-flex items-center px-1.5 py-0 rounded-full text-[11px] leading-tight bg-orange-50 text-orange-800 border border-orange-200/80"
            data-testid="task-repo-address-mismatch-badge"
            :title="`任务记录：${tp.stored_repo_address}；当前项目：${(tp.project && tp.project.git_repos && tp.project.git_repos[0]) || ''}。同步后请重新克隆，容器推送才会改到新地址。`"
          >
            仓库地址已变更
          </span>
          <button
            v-if="tp.repo_address_mismatch"
            type="button"
            class="text-[11px] px-1.5 py-0 rounded border border-orange-300 bg-white text-orange-900 hover:bg-orange-100 disabled:opacity-50 disabled:cursor-not-allowed"
            data-testid="task-repo-address-sync-btn"
            :disabled="staleRepoSyncLoading"
            @click="emit('sync-repo-address')"
          >
            {{ staleRepoSyncLoading ? '同步中…' : '同步仓库地址' }}
          </button>
          <span
            v-if="staleRepoSyncNeedsReclone"
            class="inline-flex items-center px-1.5 py-0 rounded-full text-[11px] leading-tight bg-blue-50 text-blue-700 border border-blue-200/80"
            data-testid="task-repo-address-needs-reclone-badge"
            title="地址已同步，正在重新克隆容器仓库，推送将改到新地址"
          >
            需重新克隆
          </span>
        </div>
        <div
          v-if="tp.project && Array.isArray(tp.project.git_repos) && tp.project.git_repos.length > 0"
          class="ml-1 pl-1 border-l border-gray-200/60"
        >
          <p class="text-[11px] text-gray-500 mb-0.5 leading-tight">仓库基准分支：</p>
          <ul class="space-y-0.5 list-none p-0 m-0">
            <li
              v-for="repoUrl in tp.project.git_repos"
              :key="repoUrl"
              class="border border-gray-100 rounded px-1.5 py-0.5 text-xs list-none"
              data-testid="task-linked-repo-row"
            >
              <div class="flex flex-wrap items-center gap-1.5 text-xs leading-snug">
                <!-- Anti-Replay-OK: navigation -->
                <a
                  v-if="isHttpRepoUrl(repoUrl)"
                  :href="String(repoUrl).trim()"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-gray-600 truncate max-w-xs hover:underline hover:text-blue-600"
                  :title="repoUrl"
                  data-testid="task-linked-repo-url"
                >{{ repoUrl }}</a>
                <span
                  v-else
                  class="text-gray-600 truncate max-w-xs"
                  :title="repoUrl"
                >{{ repoUrl }}</span>
                <span class="text-gray-400 shrink-0">→</span>
                <span class="font-mono text-[11px] text-gray-700">{{ (tp.repo_branches && tp.repo_branches[repoUrl]) || '未配置' }}</span>
                <span
                  class="inline-flex items-center px-1.5 py-0 rounded-full text-[11px] leading-tight border"
                  :class="gitRepoOAuthStatusBadgeClass(gitRepoTokenStatus[repoUrl], { loading: !!gitRepoStatusLoading[repoUrl] })"
                  data-testid="task-linked-repo-git-probe-badge"
                  :title="gitRepoProbeErrorByUrl[repoUrl] || gitRepoOAuthStatusHint(gitRepoTokenStatus[repoUrl])"
                  v-bind="gitRepoProbeTraceIdByUrl[repoUrl] ? { 'data-traceId': gitRepoProbeTraceIdByUrl[repoUrl] } : {}"
                >{{ resolveGitRepoOAuthStatusLabel(gitRepoTokenStatus[repoUrl], { loading: !!gitRepoStatusLoading[repoUrl] }) }}</span>
              </div>
            </li>
          </ul>
        </div>
        <div v-else class="ml-1 text-[11px] text-gray-400 leading-tight">
          该项目暂无仓库
        </div>
      </div>
    </div>
    <p v-else class="text-gray-500 text-xs">暂无关联项目</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { isHttpRepoUrl } from '../../utils/repoUrl.js'
import { buildTaskProjectsWithDetails } from '../../utils/taskProjectsWithDetails.js'
import { useTaskLinkedRepoGitProbe } from '../../composables/taskDetail/useTaskLinkedRepoGitProbe.js'
import {
  gitRepoOAuthStatusBadgeClass,
  gitRepoOAuthStatusHint,
  resolveGitRepoOAuthStatusLabel,
} from '../../utils/gitRepoOAuthStatusUtils.js'

const props = defineProps({
  tenantId: { type: String, default: '' },
  taskProjectsWithDetails: { type: Array, required: true },
  fallbackApiProjects: { type: Array, default: () => [] },
  workspaceProjects: { type: Array, default: () => [] },
  staleRepoSyncLoading: { type: Boolean, default: false },
  staleRepoSyncNeedsReclone: { type: Boolean, default: false },
})

const displayProjects = computed(() => {
  if (Array.isArray(props.taskProjectsWithDetails) && props.taskProjectsWithDetails.length > 0) {
    return props.taskProjectsWithDetails
  }
  return buildTaskProjectsWithDetails(props.fallbackApiProjects, props.workspaceProjects)
})

const linkedRepoUrls = computed(() => {
  const out = []
  for (const tp of displayProjects.value) {
    const repos = tp?.project?.git_repos
    if (!Array.isArray(repos)) continue
    for (const u of repos) {
      const s = String(u || '').trim()
      if (s) out.push(s)
    }
  }
  return out
})
const tenantIdRef = computed(() => props.tenantId)
const {
  gitRepoTokenStatus,
  gitRepoStatusLoading,
  gitRepoProbeErrorByUrl,
  gitRepoProbeTraceIdByUrl,
} = useTaskLinkedRepoGitProbe({ tenantId: tenantIdRef, repoUrls: linkedRepoUrls })

const emit = defineEmits(['sync-repo-address'])
</script>
