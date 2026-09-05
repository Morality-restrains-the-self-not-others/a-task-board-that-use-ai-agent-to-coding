<template>
  <div data-alias="view-system-admin-cloud-authorizations" id="cloud-authorizations" class="p-6">
    <div class="mb-6">
      <div class="flex justify-between items-center">
        <h2 class="text-xl font-semibold text-gray-900">云平台服务器镜像设置</h2>
        <button
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
          @click="addImageModalVisible = true"
        >
          添加云平台服务器镜像
        </button>
      </div>
    </div>

    <SystemAdminCloudAuthAddModal
      v-model:visible="addImageModalVisible"
      v-model:selected-image="selectedImage"
      :form="addForm"
      :cloud-platforms="cloudPlatforms"
      :regions="regions"
      :images="images"
      :submitting="addingImage"
      @submit="handleAddImage"
      @platform-type-change="handlePlatformTypeChange"
      @region-change="handleRegionChange"
      @cloud-image-change="handleCloudImageChange"
    />

    <SystemAdminCloudAuthEditModal
      v-model:visible="editImageModalVisible"
      v-model:selected-image="editSelectedImage"
      :form="editForm"
      :cloud-platforms="cloudPlatforms"
      :regions="editRegions"
      :images="editImages"
      :loading="loadingEditData"
      :submitting="editingImage"
      @submit="handleEditImage"
      @platform-type-change="handleEditPlatformTypeChange"
      @region-change="handleEditRegionChange"
      @cloud-image-change="handleEditCloudImageChange"
    />

    <SystemAdminCloudAuthImagesTable
      :loading="loadingImages"
      :images="cloudServerImages"
      :get-platform-label="getPlatformLabel"
      :format-date="formatDate"
      @add="addImageModalVisible = true"
      @edit="openEditImageModal"
      @delete="(image) => handleDeleteImage(image.id, image.image_name)"
    />
  </div>
</template>

<script setup>
/* @alias:view-system-admin-cloud-authorizations */
import SystemAdminCloudAuthAddModal from '../components/SystemAdminCloudAuthAddModal.vue'
import SystemAdminCloudAuthEditModal from '../components/SystemAdminCloudAuthEditModal.vue'
import SystemAdminCloudAuthImagesTable from '../components/SystemAdminCloudAuthImagesTable.vue'
import { useSystemAdminCloudAuthorizations } from '../composables/useSystemAdminCloudAuthorizations.js'

const {
  addImageModalVisible,
  editImageModalVisible,
  loadingImages,
  addingImage,
  editingImage,
  cloudServerImages,
  cloudPlatforms,
  regions,
  images,
  selectedImage,
  editRegions,
  editImages,
  editSelectedImage,
  loadingEditData,
  addForm,
  editForm,
  formatDate,
  getPlatformLabel,
  handlePlatformTypeChange,
  handleRegionChange,
  handleCloudImageChange,
  handleAddImage,
  openEditImageModal,
  handleEditPlatformTypeChange,
  handleEditRegionChange,
  handleEditCloudImageChange,
  handleEditImage,
  handleDeleteImage
} = useSystemAdminCloudAuthorizations()
</script>

<style scoped>
/* 组件内样式可以在这里添加 */
</style>
