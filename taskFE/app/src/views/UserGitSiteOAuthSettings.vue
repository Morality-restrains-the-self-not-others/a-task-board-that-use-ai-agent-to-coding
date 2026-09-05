<template>
  <div data-alias="view-user-git-site-oauth" class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="git-site-oauth" class="shrink-0" />

      <main class="flex-1 max-w-3xl space-y-6">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6">
          <h1 class="text-2xl font-semibold text-text mb-2">Git 网站 OAuth</h1>
          <p class="text-sm text-text-light">
            绑定 GitHub / GitLab 等站点账号，用于 API 访问与自动创建 PR（OAuth 令牌）。
            不替代「Git 提交身份」中的 commit name/email；请到侧栏「Git 提交身份」单独管理署名。
          </p>
        </div>

        <div
          v-if="!isOwnProfile"
          class="bg-amber-50 border border-amber-200 rounded-xl p-6 text-sm text-amber-900"
        >
          仅登录用户本人可在账号中心管理 GitHub 授权。请从本人资料页进入，或确认当前网址中的用户 ID 与登录账号一致。
        </div>

        <template v-else>
          <p
            v-if="errorMessage"
            class="text-sm text-danger"
            :data-traceId="errorTraceId || undefined"
          >{{ errorMessage }}</p>
          <p v-if="successMessage" class="text-sm text-success">{{ successMessage }}</p>

          <div class="bg-white rounded-xl shadow-sm border border-border p-6">
            <div class="mb-4 flex flex-wrap gap-2">
              <button
                v-for="entry in providerOptions"
                :key="entry.provider_key"
                type="button"
                class="px-3 py-1.5 rounded border text-sm"
                :class="selectedProviderKey === entry.provider_key ? 'border-primary text-primary bg-primary/10' : 'border-border text-text hover:bg-gray-50'"
                @click="switchProvider(entry.provider_key)"
              >
                {{ entry.label }}
              </button>
            </div>
            <p v-if="statusLoading" class="text-sm text-text-light">加载中...</p>
            <template v-else-if="!statusLoaded || !status">
              <p class="text-sm text-text-light">暂时无法获取绑定状态，请稍后重试。</p>
              <button
                type="button"
                class="mt-3 px-4 py-2 rounded-lg border border-border text-sm hover:bg-gray-50"
                @click="fetchStatus"
              >
                重新加载
              </button>
            </template>
            <template v-else>
              <div class="space-y-3 mb-4 pb-4 border-b border-border">
                <p class="text-sm font-medium text-text">授权范围（GitHub App / scope）</p>
                <template v-if="status.connected && scopeTokensGranted.length">
                  <p class="text-xs text-text-light">当前令牌已授予</p>
                  <ul class="flex flex-wrap gap-2 list-none p-0 m-0">
                    <li
                      v-for="s in scopeTokensGranted"
                      :key="'g-' + s"
                      class="text-xs px-2 py-1 rounded-md bg-primary/10 text-primary font-mono"
                    >
                      {{ s }}
                    </li>
                  </ul>
                </template>
                <p v-else-if="status.connected" class="text-xs text-text-light">
                  当前已绑定，但服务端未记录具体 scope；重新授权后可显示完整范围。
                </p>
                <div class="space-y-1">
                  <p class="text-xs text-text-light">
                    网站：{{ selectedProviderEntry?.website || '—' }}
                  </p>
                  <p class="text-xs text-text-light">
                    Client ID：{{ selectedProviderEntry?.clientId || '—' }}
                  </p>
                </div>
                <ul class="flex flex-wrap gap-2 list-none p-0 m-0">
                  <li
                    v-for="s in scopeTokensRequested"
                    :key="'r-' + s"
                    class="text-xs px-2 py-1 rounded-md bg-gray-100 text-text font-mono border border-border"
                  >
                    {{ s }}
                  </li>
                </ul>
              </div>
              <div v-if="status.connected" class="space-y-4">
                <div class="space-y-3">
                  <p class="text-sm text-text">
                    已绑定 {{ selectedProviderLabel }} 账号（{{ connectionList.length }}）
                  </p>
                  <div
                    v-for="(item, idx) in connectionList"
                    :key="`github-connection-${item.github_user_id || item.github_login || 'unknown'}-${idx}`"
                    class="rounded-lg border border-border p-3"
                  >
                    <div class="flex items-center justify-between gap-3">
                      <p class="text-sm text-text">
                        <span class="font-medium">{{ item.github_login || ('用户 #' + item.github_user_id) }}</span>
                        <span v-if="item.github_user_id" class="text-xs text-text-light ml-2">#{{ item.github_user_id }}</span>
                      </p>
                      <button
                        type="button"
                        :disabled="disconnecting"
                        class="px-3 py-1.5 rounded-lg border border-border text-text text-xs hover:bg-gray-50 disabled:opacity-60"
                        @click="disconnectConnection(item)"
                      >
                        {{ disconnecting ? '处理中...' : '取消该账号授权' }}
                      </button>
                    </div>
                  </div>
                </div>
                <div class="flex flex-wrap gap-3">
                  <button
                    type="button"
                    :disabled="connecting"
                    class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
                    @click="startProviderAuthorize"
                  >
                    {{ connecting ? '跳转中...' : `绑定新的${selectedProviderLabel}账号` }}
                  </button>
                </div>
              </div>
              <div v-else class="space-y-4">
                <p class="text-sm text-text-light">尚未绑定 {{ selectedProviderLabel }} 账号。</p>
                <button
                  type="button"
                  :disabled="connecting"
                  class="px-4 py-2 rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-60"
                  @click="startProviderAuthorize"
                >
                  {{ connecting ? '跳转中...' : `使用${selectedProviderLabel}授权` }}
                </button>
              </div>
            </template>
          </div>
        </template>
      </main>
    </div>
  </div>
</template>

<script setup>
import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import { useUserGitSiteOAuthSettings } from '../composables/useUserGitSiteOAuthSettings.js'

const {
  tenantId,
  isOwnProfile,
  providerOptions,
  selectedProviderKey,
  selectedProviderEntry,
  selectedProviderLabel,
  statusLoading,
  statusLoaded,
  status,
  connecting,
  disconnecting,
  errorMessage,
  errorTraceId,
  successMessage,
  scopeTokensGranted,
  scopeTokensRequested,
  connectionList,
  fetchStatus,
  startProviderAuthorize,
  disconnectConnection,
  switchProvider,
} = useUserGitSiteOAuthSettings()
</script>
