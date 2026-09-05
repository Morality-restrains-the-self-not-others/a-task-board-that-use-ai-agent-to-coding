<template>
  <div data-alias="view-system-admin-container-images" id="container-images" class="p-6">
    <div class="mb-6">
      <div class="flex justify-between items-center">
        <h2 class="text-xl font-semibold text-gray-900">容器镜像列表</h2>
        <button
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
          @click="uploadModalVisible = true"
        >
          上传镜像
        </button>
      </div>
    </div>

    <section
      data-testid="marketplace-admin-panel"
      class="mb-6 rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
    >
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="min-w-0">
          <h3 class="text-base font-semibold text-gray-900">镜像市场管理</h3>
          <p class="mt-1 text-sm text-gray-500">
            通过 SSO 进入厂商镜像运营后台；可配置租户成为厂商是否需要申请审核。
          </p>
        </div>
        <a
          :href="ssoAiProviderAdminHref"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex shrink-0 items-center justify-center rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50"
        >
          镜像市场管理（SSO）
        </a>
      </div>
      <div class="mt-4 flex flex-col gap-2 border-t border-gray-100 pt-4">
        <label class="inline-flex items-center gap-3 text-sm font-medium text-gray-800">
          <input
            v-model="vendorApplicationReviewEnabled"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
            :disabled="settingsLoading || settingsSaving || saveMarketplaceSettingsGuard.isBusy()"
            @change="saveMarketplaceSettings"
          >
          开启厂商申请审核
        </label>
        <p class="text-xs text-gray-500">
          关闭后，已绑定邮箱的租户可直接打开厂商门户（SSO），无需提交申请等待审核。
        </p>
        <p
          v-if="settingsError"
          class="text-xs text-red-600"
          :data-traceId="settingsErrorTraceId || undefined"
        >
          {{ settingsError }}
        </p>
      </div>
    </section>

    <SystemAdminContainerImageUploadModal
      v-model:visible="uploadModalVisible"
      :form="uploadForm"
      :submitting="uploading"
      @submit="handleUploadImage"
    />

    <SystemAdminContainerImageEditModal
      v-model:visible="editModalVisible"
      :form="editForm"
      :loading="loadingEdit"
      :submitting="editing"
      @submit="handleEditImage"
    />

    <SystemAdminContainerImageDeleteModal
      v-model:visible="deleteModalVisible"
      :submitting="deleting"
      @submit="handleDeleteImage"
    />

    <SystemAdminContainerImageEnvironmentModal
      v-model:visible="environmentModalVisible"
      :form="environmentForm"
      :cloud-platforms="cloudPlatforms"
      :server-images="serverImages"
      :loading="loadingEnvironment"
      :submitting="settingEnvironment"
      @submit="handleSetEnvironment"
    />

    <div>
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">名称</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">描述</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">镜像地址</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">大小</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="image in containerImages" :key="image.id">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ image.id }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ image.name }}</td>
              <td class="px-6 py-4 text-sm text-gray-500">{{ image.description || '' }}</td>
              <td class="px-6 py-4 text-sm text-gray-500">{{ image.image_url }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ image.size ? formatFileSize(image.size) : '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(image.created_at) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <button
                  v-if="image.is_cloud_imported"
                  class="text-gray-400 cursor-not-allowed mr-3"
                  disabled
                  title="云平台导入的镜像不可编辑"
                >
                  编辑
                </button>
                <button
                  v-else
                  class="text-primary hover:text-primary/90 mr-3"
                  @click="openEditImageModal(image.id)"
                >
                  编辑
                </button>
                <button
                  class="text-blue-600 hover:text-blue-800 mr-3"
                  @click="openSetEnvironmentModal(image.id)"
                >
                  设置运行环境
                </button>
                <button
                  class="text-red-600 hover:text-red-800"
                  @click="openDeleteImageModal(image.id)"
                >
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:view-system-admin-container-images */
import { computed, onMounted, ref } from 'vue'
import SystemAdminContainerImageUploadModal from '../components/SystemAdminContainerImageUploadModal.vue'
import SystemAdminContainerImageEditModal from '../components/SystemAdminContainerImageEditModal.vue'
import SystemAdminContainerImageDeleteModal from '../components/SystemAdminContainerImageDeleteModal.vue'
import SystemAdminContainerImageEnvironmentModal from '../components/SystemAdminContainerImageEnvironmentModal.vue'
import { useSystemAdminContainerImages } from '../composables/useSystemAdminContainerImages.js'
import { getApiUrl } from '../utils/config'
import { apiFetch } from '../utils/apiUtils'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import toastService from '../utils/toastService'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const {
  uploadModalVisible,
  editModalVisible,
  deleteModalVisible,
  environmentModalVisible,
  loadingEdit,
  loadingEnvironment,
  uploading,
  editing,
  deleting,
  settingEnvironment,
  uploadForm,
  editForm,
  environmentForm,
  containerImages,
  serverImages,
  cloudPlatforms,
  formatFileSize,
  formatDate,
  handleUploadImage,
  openEditImageModal,
  handleEditImage,
  openDeleteImageModal,
  handleDeleteImage,
  openSetEnvironmentModal,
  handleSetEnvironment
} = useSystemAdminContainerImages()

const ssoAiProviderAdminHref = computed(() =>
  getApiUrl('/api/accounts/sso/ai-provider/admin/')
)

const vendorApplicationReviewEnabled = ref(true)
const settingsLoading = ref(false)
const settingsSaving = ref(false)
const settingsError = ref('')
const settingsErrorTraceId = ref('')

// OPT-20260819-038: 镜像市场设置保存是写操作，防连点/超时重试双发 PATCH
const saveMarketplaceSettingsGuard = createClickGuard()

const loadMarketplaceSettings = async () => {
  settingsLoading.value = true
  settingsError.value = ''
  settingsErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/ai-provider/admin-marketplace-settings/', {
      credentials: 'include',
    })
    const { data, traceId } = await safeResponseJson(response, { fallback: {} })
    if (!response.ok) {
      settingsError.value = data?.detail || data?.error?.message || '加载镜像市场设置失败'
      settingsErrorTraceId.value = traceId || ''
      return
    }
    vendorApplicationReviewEnabled.value = data?.vendor_application_review_enabled !== false
  } catch (error) {
    settingsError.value = error?.message || '加载镜像市场设置失败'
  } finally {
    settingsLoading.value = false
  }
}

const saveMarketplaceSettings = async () => {
  // OPT-20260819-038: 镜像市场设置保存是写操作，防连点/超时重试双发 PATCH
  await saveMarketplaceSettingsGuard.run(async ({ idempotencyKey }) => {
    settingsSaving.value = true
    settingsError.value = ''
    settingsErrorTraceId.value = ''
    try {
      const response = await apiFetch('/api/ai-provider/admin-marketplace-settings/', {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          vendor_application_review_enabled: vendorApplicationReviewEnabled.value,
        }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        settingsError.value = data?.detail || data?.error?.message || '保存失败'
        settingsErrorTraceId.value = traceId || ''
        toastService.error(settingsError.value)
        await loadMarketplaceSettings()
        return
      }
      vendorApplicationReviewEnabled.value = data?.vendor_application_review_enabled !== false
      toastService.success(
        vendorApplicationReviewEnabled.value ? '已开启厂商申请审核' : '已关闭厂商申请审核，租户可直达厂商门户'
      )
    } catch (error) {
      settingsError.value = error?.message || '保存失败'
      toastService.error(settingsError.value)
      await loadMarketplaceSettings()
    } finally {
      settingsSaving.value = false
    }
  })
}

onMounted(() => {
  loadMarketplaceSettings()
})
</script>

<style scoped>
/* 组件内样式可以在这里添加 */
</style>
