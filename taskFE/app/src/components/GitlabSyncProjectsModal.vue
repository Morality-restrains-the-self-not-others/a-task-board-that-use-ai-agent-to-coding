<template>
  <div
    v-if="show"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999 p-4"
    data-testid="gitlab-sync-projects-modal"
    @click="emit('close')"
  >
    <div
      class="bg-white rounded-xl shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col z-10000"
      @click.stop
    >
      <div class="border-b border-gray-100">
        <div class="flex justify-between items-start gap-4 p-6 pb-3">
          <div class="min-w-0 flex-1 flex items-center gap-3 flex-wrap">
            <h3 class="text-xl font-bold text-gray-900 shrink-0">从 GitLab 同步项目</h3>
            <div
              v-if="purchasedRegions.length > 0"
              class="min-w-0 flex items-center gap-2 flex-wrap text-sm text-text-light"
              data-testid="gitlab-sync-source"
            >
              <span class="shrink-0">来源：</span>
              <select
                v-if="purchasedRegions.length > 1"
                id="gitlab-sync-region"
                class="min-w-0 max-w-xs sm:max-w-md px-2 py-1 border border-gray-300 rounded-md text-sm text-text focus:outline-none focus:ring-2 focus:ring-primary bg-white"
                :value="selectedRegionSlug"
                :disabled="creating || loading || regionsLoading"
                data-testid="gitlab-sync-region-select"
                aria-label="选择已购买的 GitLab"
                @change="emit('select-region', $event.target.value)"
              >
                <!-- Anti-Replay-OK: region change is read-only reload of remote repos -->
                <option
                  v-for="r in purchasedRegions"
                  :key="r.region"
                  :value="r.region"
                >
                  {{ r.region_name || r.region }}（{{ r.gitlab_web_url }}）
                </option>
              </select>
              <p
                v-else
                class="truncate min-w-0"
                data-testid="gitlab-sync-source-url"
              >
                {{ gitlabWebsite || purchasedRegions[0]?.gitlab_web_url }}
              </p>
              <span v-if="gitlabLogin" class="shrink-0">· 用户 {{ gitlabLogin }}</span>
            </div>
          </div>
          <button
            type="button"
            class="text-gray-500 hover:text-gray-700 shrink-0"
            :disabled="creating"
            aria-label="关闭"
            @click="emit('close')"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div
          v-if="oauthBound"
          class="flex items-end justify-between gap-4 px-6 pb-6 flex-wrap"
        >
          <div class="flex items-end gap-3 min-w-0 flex-1 flex-wrap">
            <div class="min-w-0 max-w-xs sm:max-w-sm">
              <select
                id="gitlab-sync-workspace"
                :value="selectedWorkspaceId"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                :disabled="creating || loading"
                aria-label="目标工作空间"
                @change="emit('update:selectedWorkspaceId', $event.target.value)"
              >
                <option value="">请选择工作空间</option>
                <option v-for="ws in workspaces" :key="ws.id" :value="String(ws.id)">
                  {{ ws.name }}
                </option>
              </select>
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 flex-wrap shrink-0">
            <button
              type="button"
              class="btn-secondary px-3 py-1.5 text-sm inline-flex items-center gap-2 whitespace-nowrap"
              :disabled="loading || creating"
              data-testid="gitlab-sync-refresh-btn"
              @click="emit('refresh')"
            >
              刷新列表
            </button>
            <button
              type="button"
              class="btn-primary px-3 py-1.5 text-sm disabled:opacity-50 whitespace-nowrap"
              :disabled="creating || loading || selectedCount < 2 || !combinedProjectName.trim() || !selectedWorkspaceId"
              data-testid="gitlab-sync-combined-create-btn"
              @click="emit('create-combined')"
            >
              <span v-if="creating">创建中…</span>
              <span v-else>合并为一个项目（{{ selectedCount }}）</span>
            </button>
            <button
              type="button"
              class="btn-primary px-3 py-1.5 text-sm disabled:opacity-50 whitespace-nowrap"
              :disabled="creating || loading || batchEligibleCount === 0 || !selectedWorkspaceId"
              data-testid="gitlab-sync-create-btn"
              @click="emit('create')"
            >
              <span v-if="creating">创建中…</span>
              <span v-else>每个仓库各建一个项目（{{ batchEligibleCount }}）</span>
            </button>
          </div>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto p-6 space-y-4">
        <div
          v-if="regionsLoading"
          class="py-10 text-center text-text-light text-sm"
          aria-busy="true"
          data-testid="gitlab-sync-regions-loading"
        >
          正在加载已购买的 GitLab…
        </div>

        <div
          v-else-if="!purchasedRegions.length"
          class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
          data-testid="gitlab-sync-no-purchased-region"
        >
          <p class="font-medium mb-2">尚未开通可用的 GitLab 区域</p>
          <p class="mb-3 text-amber-800">
            请先购买或获赠带访问地址的 GitLab 区域，再从对应实例同步仓库。未挂载完成的区域不会出现在此列表。
          </p>
          <a
            href="/pricing/"
            class="btn-primary text-sm px-4 py-2 inline-flex"
            data-testid="gitlab-sync-go-pricing"
          >
            <!-- Anti-Replay-OK: real anchor navigation to pricing -->
            前往价格页开通
          </a>
        </div>

        <div
          v-else-if="!oauthBound && !loading"
          class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
        >
          <p class="font-medium mb-2">尚未绑定 GitLab OAuth</p>
          <p class="mb-3 text-amber-800">请先完成所选 GitLab 实例的授权，以便读取仓库列表。</p>
          <!-- Anti-Replay-OK: real <a href> navigation; start API 302s HTML Accept 到授权页 -->
          <a
            v-if="oauthStartHref"
            :href="oauthStartHref"
            class="btn-primary text-sm px-4 py-2 inline-flex"
            data-testid="gitlab-sync-oauth-btn"
          >
            前往 GitLab 授权
          </a>
        </div>

        <div
          v-if="error"
          class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
          role="alert"
          :data-traceId="errorTraceId || undefined"
        >
          {{ error }}
        </div>

        <div v-if="createResult" class="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          已成功创建 {{ createResult.created?.length || 0 }} 个项目
          <template v-if="createResult.skipped?.length">
            ，跳过 {{ createResult.skipped.length }} 个已存在仓库
          </template>
          <template v-if="createResult.errors?.length">
            ，{{ createResult.errors.length }} 个失败
          </template>
        </div>

        <div
          v-if="purchasedRegions.length && loading"
          class="py-10 text-center text-text-light text-sm"
          aria-busy="true"
        >
          正在加载 GitLab 仓库…
        </div>

        <div
          v-else-if="purchasedRegions.length && oauthBound && repos.length === 0 && !error"
          class="py-10 text-center text-text-light text-sm"
        >
          未找到可同步的 GitLab 仓库
        </div>

        <div v-else-if="purchasedRegions.length && oauthBound && repos.length > 0" class="space-y-2">
          <div class="flex items-center gap-3 flex-wrap">
            <label
              class="inline-flex items-center gap-2 text-sm font-medium text-text cursor-pointer whitespace-nowrap shrink-0"
            >
              <input
                type="checkbox"
                class="rounded border-gray-300 text-primary focus:ring-primary"
                :checked="allSelectableChecked"
                :disabled="creating || !repos.length"
                data-testid="gitlab-sync-select-all"
                @change="emit('toggle-select-all', $event.target.checked)"
              >
              全选仓库
            </label>
            <span
              class="text-xs text-text-light shrink-0 whitespace-nowrap"
              data-testid="gitlab-sync-selected-count"
            >
              已选 {{ selectedCount }} / {{ repos.length }}
            </span>
            <div class="flex-1 flex items-center justify-center min-w-0">
              <div
                v-if="selectedCount >= 2"
                class="flex items-center gap-2 shrink-0"
              >
                <label
                  for="gitlab-sync-combined-name"
                  class="text-xs font-medium text-text whitespace-nowrap shrink-0"
                >
                  合并项目名称
                </label>
                <input
                  id="gitlab-sync-combined-name"
                  :value="combinedProjectName"
                  type="text"
                  maxlength="200"
                  placeholder="输入名称"
                  class="min-w-0 w-36 sm:w-44 px-2 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  :disabled="creating"
                  data-testid="gitlab-sync-combined-name-input"
                  @input="emit('update:combinedProjectName', $event.target.value)"
                >
              </div>
            </div>
            <input
              v-model="repoFilter"
              type="search"
              placeholder="快速过滤仓库…"
              class="shrink-0 w-36 sm:w-44 px-2 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              :disabled="creating"
              data-testid="gitlab-sync-repo-filter"
              aria-label="快速过滤仓库"
            >
          </div>

          <ul
            v-if="filteredRepos.length > 0"
            class="divide-y divide-gray-100 rounded-lg border border-gray-200 max-h-72 overflow-y-auto"
          >
            <li
              v-for="repo in filteredRepos"
              :key="repo.http_url_to_repo"
              class="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-50 cursor-pointer select-none"
              :class="{ 'bg-primary/5': selectedRepoKeys[repo.http_url_to_repo] }"
              @click="toggleRepoRow(repo)"
            >
              <input
                type="checkbox"
                class="rounded border-gray-300 text-primary focus:ring-primary shrink-0 pointer-events-none"
                :checked="Boolean(selectedRepoKeys[repo.http_url_to_repo])"
                :disabled="creating"
                tabindex="-1"
                aria-hidden="true"
              >
              <div class="min-w-0 flex-1 leading-tight">
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span class="font-medium text-sm text-text">{{ repo.name }}</span>
                  <span
                    v-if="repo.imported_in_single_repo_project"
                    class="text-[11px] px-1.5 py-px rounded-full bg-amber-100 text-amber-800"
                  >
                    单仓已占用
                  </span>
                  <span
                    v-else-if="repo.imported_in_any_project"
                    class="text-[11px] px-1.5 py-px rounded-full bg-gray-100 text-gray-600"
                  >
                    已在多仓项目中
                  </span>
                </div>
                <p class="text-[11px] text-text-light truncate">{{ repo.path_with_namespace }}</p>
                <p v-if="repo.description" class="text-[11px] text-text-light line-clamp-1">{{ repo.description }}</p>
              </div>
            </li>
          </ul>
          <p
            v-else
            class="py-6 text-center text-sm text-text-light rounded-lg border border-gray-200"
          >
            无匹配的仓库
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
  creating: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
  oauthBound: { type: Boolean, default: false },
  gitlabWebsite: { type: String, default: '' },
  gitlabLogin: { type: String, default: '' },
  oauthStartHref: { type: String, default: '' },
  purchasedRegions: { type: Array, default: () => [] },
  selectedRegionSlug: { type: String, default: '' },
  regionsLoading: { type: Boolean, default: false },
  repos: { type: Array, default: () => [] },
  selectedCount: { type: Number, default: 0 },
  batchEligibleCount: { type: Number, default: 0 },
  allSelectableChecked: { type: Boolean, default: false },
  selectedRepoKeys: { type: Object, default: () => ({}) },
  workspaces: { type: Array, default: () => [] },
  selectedWorkspaceId: { type: String, default: '' },
  createResult: { type: Object, default: null },
  combinedProjectName: { type: String, default: '' },
})

const repoFilter = ref('')

watch(
  () => props.show,
  (visible) => {
    if (!visible) repoFilter.value = ''
  },
)

const filteredRepos = computed(() => {
  const query = repoFilter.value.trim().toLowerCase()
  if (!query) return props.repos
  return props.repos.filter((repo) => {
    const haystack = [
      repo.name,
      repo.path_with_namespace,
      repo.description,
      repo.http_url_to_repo,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    return haystack.includes(query)
  })
})

function toggleRepoRow(repo) {
  if (props.creating) return
  const checked = !props.selectedRepoKeys[repo.http_url_to_repo]
  emit('toggle-repo', repo.http_url_to_repo, checked)
}

const emit = defineEmits([
  'close',
  'refresh',
  'create',
  'create-combined',
  'select-region',
  'toggle-repo',
  'toggle-select-all',
  'update:selectedWorkspaceId',
  'update:combinedProjectName',
])
</script>
