<template>
  <div class="max-w-7xl mx-auto p-6 flex flex-col min-h-0">
    <header class="mb-8 shrink-0">
      <div class="flex flex-col gap-4 sm:flex-row sm:justify-between sm:items-start">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">镜像市场</h1>
          <p class="mt-1 text-sm text-gray-600 max-w-2xl">
            浏览并安装租户可用的容器镜像；开发中条目需在厂商门户维护并由账号绑定厂商后可见。
          </p>
          <div v-if="!isLoadingCatalog || !isLoadingInstalled" class="mt-3 flex flex-wrap gap-2 text-xs">
            <span class="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-gray-700">
              目录 {{ catalogImages.length }} 条
            </span>
            <span class="inline-flex items-center rounded-full bg-primary/10 px-2.5 py-0.5 text-primary font-medium">
              已安装 {{ installedImages.length }} 条
            </span>
          </div>
        </div>
        <a
          v-if="showVendorSsoButton"
          :href="ssoAiProviderVendorHref"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex shrink-0 items-center justify-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
        >
          厂商门户（SSO）
        </a>
        <a
          v-else-if="showBindEmailCta"
          :href="emailBindingHref"
          class="inline-flex shrink-0 items-center justify-center rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
        >
          绑定邮箱后申请成为厂商
        </a>
        <button
          v-else-if="showPendingCta"
          type="button"
          disabled
          class="inline-flex shrink-0 cursor-not-allowed items-center justify-center rounded-lg border border-gray-300 bg-gray-100 px-4 py-2 text-sm font-medium text-gray-500"
        >
          审核中
        </button>
        <!-- Anti-Replay-OK: pure UI expand; no write side-effect until form submit -->
        <button
          v-else-if="showVendorApplyOpenCta"
          type="button"
          data-testid="vendor-apply-open"
          class="inline-flex shrink-0 items-center justify-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
          @click="vendorApplyFormOpen = true"
        >
          {{ vendorApplyHeading }}
        </button>
      </div>
      <p
        v-if="vendorStatusError"
        class="mt-3 text-sm text-red-700"
        :data-traceId="vendorStatusErrorTraceId || undefined"
      >{{ vendorStatusError }}</p>
      <ImageMarketVendorApply
        v-if="canOpenVendorApplyForm && vendorApplyFormOpen"
        :heading="vendorApplyHeading"
        :reject-note="vendorRejectNote"
        :has-email="hasEmail"
        :email-binding-href="emailBindingHref"
        @applied="onVendorApplied"
        @cancel="vendorApplyFormOpen = false"
      />
    </header>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2 lg:items-start shrink-0">
      <div class="rounded-xl bg-white p-6 shadow-md ring-1 ring-gray-100">
        <h2 class="text-lg font-semibold text-gray-900">开发中镜像</h2>
        <p class="mt-1 text-sm text-gray-500">
          若当前账号已在镜像市场绑定厂商，将自动列出尚未上架的镜像；安装方式与厂商门户一致。
        </p>

        <div class="mt-4">
          <label class="sr-only">筛选开发中镜像</label>
          <input
            v-model="devSearch"
            type="search"
            placeholder="按名称、描述、供应商筛选…"
            class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
          >
        </div>

        <div v-if="isLoadingDevCatalog" class="mt-8 flex justify-center py-12">
          <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
        <div v-else-if="devCatalogError" class="mt-6 rounded-lg border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-700" :data-traceId="devCatalogErrorTraceId || undefined">
          {{ devCatalogError }}
          <button type="button" class="ml-2 font-medium text-primary underline" @click="loadDevCatalog">重试</button>
        </div>
        <div v-else-if="filteredDevImages.length === 0" class="mt-8 rounded-lg border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500">
          {{ devSearch ? '无匹配的开发中镜像' : '暂无开发中镜像（需绑定厂商后可见）' }}
        </div>
        <ul v-else class="mt-6 space-y-4">
          <!-- 同组不同版本叠列：一组一张卡，卡内为版本列表 -->
          <li
            v-for="group in devImageGroups"
            :key="group.groupKey"
            class="flex flex-col gap-3 rounded-lg border border-gray-100 p-4 transition-shadow hover:shadow-sm"
          >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <ImageGroupIcon :src="imageGroupIconSrc(group)" :alt="group.name" />
                <h3 class="font-medium text-gray-900">{{ group.name }}</h3>
                <span class="rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800">开发中</span>
              </div>
              <p class="mt-1 text-sm text-gray-600 line-clamp-2">{{ group.description || '暂无描述' }}</p>
              <div
                v-for="image in group.versions"
                :key="image.id"
                class="mt-3 border-t border-gray-100 pt-3 first:mt-2 first:border-t-0 first:pt-0"
              >
                <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                  <div class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500">
                    <span>版本 {{ image.version || '—' }}</span>
                    <span>架构 {{ image.target_architectures?.join(', ') || '—' }}</span>
                    <span>供应商 {{ image.vendor?.company_name || '—' }}</span>
                    <span data-testid="image-updated-at">更新时间 {{ formatDateTime(image.updated_at) }}</span>
                  </div>
                  <button
                    v-if="image.is_installed"
                    type="button"
                    disabled
                    class="shrink-0 cursor-not-allowed rounded-lg bg-gray-100 px-3 py-2 text-sm font-medium text-gray-500"
                  >
                    已安装
                  </button>
                  <button
                    v-else
                    type="button"
                    class="shrink-0 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary/90 disabled:opacity-60"
                    :disabled="!!installingKey || !image.vendor?.id"
                    @click="installDevelopmentImage(image)"
                  >
                    {{ installingKey === `dev-${image.id}` ? '安装中…' : '安装' }}
                  </button>
                </div>
                <runtime-deps-block :environments="image.runtime_environments" />
                <ImageAutoRunSteps :image="image" />
                <ImageSkillsList :image="image" />
              </div>
            </div>
          </li>
        </ul>
      </div>

      <div class="rounded-xl bg-white p-6 shadow-md ring-1 ring-gray-100">
        <h2 class="text-lg font-semibold text-gray-900">已发布镜像</h2>
        <p class="mt-1 text-sm text-gray-500">市场目录中已审核上架的镜像，可直接安装到当前租户。</p>

        <div class="mt-4">
          <label class="sr-only">筛选已发布镜像</label>
          <input
            v-model="catalogSearch"
            type="search"
            placeholder="按名称、描述、供应商筛选…"
            class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
          >
        </div>

        <div v-if="isLoadingCatalog" class="mt-8 flex justify-center py-12">
          <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
        <div v-else-if="catalogError" class="mt-6 rounded-lg border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-700" :data-traceId="catalogErrorTraceId || undefined">
          {{ catalogError }}
          <button type="button" class="ml-2 font-medium text-primary underline" @click="loadCatalog">重试</button>
        </div>
        <div v-else-if="filteredCatalogImages.length === 0" class="mt-8 rounded-lg border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500">
          {{ catalogSearch ? '无匹配的已发布镜像' : '暂无已发布镜像' }}
        </div>
        <ul v-else class="mt-6 space-y-4">
          <!-- 同组不同版本叠列：一组一张卡，卡内为版本列表（后端每组仅一激活版本，多版本时同卡展示） -->
          <li
            v-for="group in catalogImageGroups"
            :key="group.groupKey"
            class="flex flex-col gap-3 rounded-lg border border-gray-100 p-4 transition-shadow hover:shadow-sm"
          >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <ImageGroupIcon :src="imageGroupIconSrc(group)" :alt="group.name" />
                <h3 class="font-medium text-gray-900">{{ group.name }}</h3>
                <span class="rounded-md bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800">已发布</span>
              </div>
              <p class="mt-1 text-sm text-gray-600 line-clamp-2">{{ group.description || '暂无描述' }}</p>
              <div
                v-for="image in group.versions"
                :key="image.id"
                class="mt-3 border-t border-gray-100 pt-3 first:mt-2 first:border-t-0 first:pt-0"
              >
                <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
                  <div class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500">
                    <span>版本 {{ image.version || '—' }}</span>
                    <span>架构 {{ image.target_architectures?.join(', ') || '—' }}</span>
                    <span>供应商 {{ image.vendor?.company_name || '—' }}</span>
                    <span data-testid="image-updated-at">更新时间 {{ formatDateTime(image.updated_at) }}</span>
                  </div>
                  <button
                    v-if="image.is_installed"
                    type="button"
                    disabled
                    class="shrink-0 cursor-not-allowed rounded-lg bg-gray-100 px-3 py-2 text-sm font-medium text-gray-500"
                  >
                    已安装
                  </button>
                  <button
                    v-else
                    type="button"
                    class="shrink-0 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary/90 disabled:opacity-60"
                    :disabled="!!installingKey"
                    @click="installPublishedImage(image.id)"
                  >
                    {{ installingKey === pubKey(image.id) ? '安装中…' : '安装' }}
                  </button>
                </div>
                <runtime-deps-block :environments="image.runtime_environments" />
                <ImageAutoRunSteps :image="image" />
                <ImageSkillsList :image="image" />
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <div class="mt-6 rounded-xl bg-white p-6 shadow-md ring-1 ring-gray-100 flex-1">
      <h2 class="text-lg font-semibold text-gray-900">已安装镜像</h2>
      <p class="mt-1 text-sm text-gray-500">当前租户已安装的镜像列表。</p>

      <div class="mt-4">
        <label class="sr-only">筛选已安装镜像</label>
        <input
          v-model="installedSearch"
          type="search"
          placeholder="按名称、描述、供应商筛选…"
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
        >
      </div>

      <div v-if="isLoadingInstalled" class="mt-8 flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
      <div v-else-if="installedError" class="mt-6 rounded-lg border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-700" :data-traceId="installedErrorTraceId || undefined">
        {{ installedError }}
        <button type="button" class="ml-2 font-medium text-primary underline" @click="loadInstalledImages">重试</button>
      </div>
      <div v-else-if="filteredInstalledImages.length === 0" class="mt-8 rounded-lg border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500">
        {{ installedSearch ? '无匹配的已安装镜像' : '尚未安装任何镜像' }}
      </div>
      <ul v-else class="mt-6 grid gap-4 sm:grid-cols-2" data-testid="installed-image-list">
        <li
          v-for="image in filteredInstalledImages"
          :key="image.id"
          class="flex flex-col justify-between rounded-lg border border-gray-100 p-4"
        >
          <div>
            <div class="flex flex-wrap items-center gap-2">
              <ImageGroupIcon :src="installedIconSrc(image)" :alt="image.name" />
              <h3 class="font-medium text-gray-900">{{ image.name }}</h3>
              <span
                v-if="image.install_source === 'development'"
                class="rounded-md bg-orange-100 px-2 py-0.5 text-xs text-orange-800"
              >开发模式安装</span>
            </div>
            <div class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-500">
              <span>版本 {{ image.version || '—' }}</span>
              <span>供应商 {{ image.vendor_name || '—' }}</span>
            </div>
            <runtime-deps-block :environments="image.runtime_environments" />
            <p class="mt-2 text-xs text-gray-400" data-testid="installed-image-updated-at">
              更新时间 {{ formatInstalledImageUpdateTime(image) || '—' }}
            </p>
            <ImageAutoRunSteps :image="image" />
            <ImageSkillsList :image="image" />
          </div>
          <button
            type="button"
            class="mt-3 self-start rounded-lg border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 disabled:opacity-60"
            :disabled="uninstallingId === image.id"
            @click="uninstallImage(image.id)"
          >
            {{ uninstallingId === image.id ? '卸载中…' : '卸载' }}
          </button>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getApiUrl } from '../utils/config'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { buildEmailBindingRedirectUrl } from '../utils/emailBindingDeepLink.js'
import { formatInstalledImageUpdateTime } from '../utils/installedImageLabel.js'
import ImageAutoRunSteps from '../components/ImageAutoRunSteps.vue'
import ImageGroupIcon from '../components/ImageGroupIcon.vue'
import ImageSkillsList from '../components/ImageSkillsList.vue'
import RuntimeDepsBlock from '../components/RuntimeDepsBlock.vue'
import ImageMarketVendorApply from '../components/ImageMarketVendorApply.vue'
import { useImageMarketCatalog } from '../composables/useImageMarketCatalog.js'
import { imageGroupIconSrc } from '../utils/imageGroupIcon.js'
import {
  canOpenVendorApplyForm as canOpenVendorApplyFormFn,
  showBindEmailCta as showBindEmailCtaFn,
  showPendingCta as showPendingCtaFn,
  showVendorApplyOpenCta as showVendorApplyOpenCtaFn,
  showVendorSsoButton as showVendorSsoButtonFn,
  vendorApplyButtonText,
} from '../utils/imageMarketVendorCta.js'

const route = useRoute()
const router = useRouter()
const tenantId = route.params.tenant

const ssoAiProviderVendorHref = computed(() =>
  getApiUrl('/api/accounts/sso/ai-provider/vendor/'),
)
const emailBindingHref = computed(() => buildEmailBindingRedirectUrl())

const vendorStatus = ref('none')
const hasEmail = ref(false)
const vendorApplicationReviewEnabled = ref(true)
const vendorRejectNote = ref('')
const vendorStatusError = ref('')
const vendorStatusErrorTraceId = ref('')
const vendorApplyFormOpen = ref(false)
const vendorFunnel = computed(() => ({
  status: vendorStatus.value,
  hasEmail: hasEmail.value,
  reviewEnabled: vendorApplicationReviewEnabled.value,
}))
const showVendorSsoButton = computed(() => showVendorSsoButtonFn(vendorFunnel.value))
const showBindEmailCta = computed(() => showBindEmailCtaFn(vendorFunnel.value))
const showPendingCta = computed(() => showPendingCtaFn(vendorFunnel.value))
const canOpenVendorApplyForm = computed(() => canOpenVendorApplyFormFn(vendorFunnel.value))
const showVendorApplyOpenCta = computed(() =>
  showVendorApplyOpenCtaFn({ ...vendorFunnel.value, formOpen: vendorApplyFormOpen.value }),
)
const vendorApplyHeading = computed(() => vendorApplyButtonText(vendorStatus.value))

// OPT-20260904-002: 展开态支持 URL/query 深链（?vendor_apply=1），供客服/E2E/审核驳回后
// 直达填表页。表单开/合时用 router.replace 同步 query，仅改 query 不动当前路径。
watch(vendorApplyFormOpen, () => {
  const query = { ...(route.query || {}) }
  if (vendorApplyFormOpen.value) {
    query.vendor_apply = '1'
  } else {
    delete query.vendor_apply
  }
  router.replace({ query })
})

const openVendorApplyFromQueryIfPresent = () => {
  if (route.query?.vendor_apply !== '1') return
  // 无邮箱态不强制展开（此时头部应为绑邮箱引导）；qualified/pending 走 SSO/审核中按钮
  if (canOpenVendorApplyForm.value && hasEmail.value) {
    vendorApplyFormOpen.value = true
  }
}

const {
  catalogImages,
  installedImages,
  isLoadingCatalog,
  isLoadingDevCatalog,
  isLoadingInstalled,
  installingKey,
  uninstallingId,
  catalogError,
  devCatalogError,
  catalogErrorTraceId,
  devCatalogErrorTraceId,
  installedError,
  installedErrorTraceId,
  catalogSearch,
  devSearch,
  installedSearch,
  filteredCatalogImages,
  filteredDevImages,
  filteredInstalledImages,
  devImageGroups,
  catalogImageGroups,
  formatDateTime,
  installedIconSrc,
  pubKey,
  loadCatalog,
  loadDevCatalog,
  loadInstalledImages,
  installPublishedImage,
  installDevelopmentImage,
  uninstallImage,
} = useImageMarketCatalog(tenantId)

const loadVendorStatus = async () => {
  vendorStatusError.value = ''
  vendorStatusErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/ai-provider/vendor-status/', { credentials: 'include' })
    if (response.ok) {
      const { data } = await safeResponseJson(response, { fallback: {} })
      vendorStatus.value = data?.status || 'none'
      hasEmail.value = data?.has_email === true
      vendorApplicationReviewEnabled.value = data?.vendor_application_review_enabled !== false
      vendorRejectNote.value = data?.vendor?.review_note || ''
      return
    }
    vendorStatusError.value = extractErrorMessage(response._errorData, response, '加载厂商门户状态失败')
    vendorStatusErrorTraceId.value = response.traceId || ''
  } catch (error) {
    vendorStatusError.value = error?.message || '加载厂商门户状态失败'
    vendorStatusErrorTraceId.value = error?.traceId || ''
  }
}

const onVendorApplied = async () => {
  vendorApplyFormOpen.value = false
  await loadVendorStatus()
}

onMounted(() => {
  loadVendorStatus().finally(() => openVendorApplyFromQueryIfPresent())
  loadCatalog()
  loadDevCatalog()
  loadInstalledImages()
})
</script>
