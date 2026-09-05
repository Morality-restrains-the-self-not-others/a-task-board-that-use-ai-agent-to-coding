<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-[60]">
    <div class="modal-select-overflow-visible bg-white rounded-lg shadow-xl w-full max-w-2xl flex flex-col max-h-[90vh]">
      <div class="flex justify-between items-center px-6 pt-6 pb-4 flex-shrink-0">
        <h3 class="text-lg font-semibold text-gray-900">设置默认配置</h3>
        <button
          type="button"
          class="text-gray-500 hover:text-gray-700 transition-colors"
          @click="handleClose"
        >
          &times;
        </button>
      </div>

      <form class="flex flex-col flex-1 min-h-0" @submit.prevent="handleSubmit">
        <SetDefaultConfigFormFields
          :form-data="formData"
          :regions="regions"
          :vpcs="vpcs"
          :vswitches="vswitches"
          :security-groups="securityGroups"
          :instances="instances"
          :loading-regions="loadingRegions"
          :region-error="regionError"
          :loading-vpcs="loadingVpcs"
          :loading-vswitches="loadingVswitches"
          :loading-security-groups="loadingSecurityGroups"
          :loading-instances="loadingInstances"
          :instance-error="instanceError"
          @region-change="handleRegionChange"
          @vpc-change="handleVpcChange"
          @instance-filter-change="handleInstanceFilterChange"
          @create-vpc="openCreateVpcModal"
          @edit-vpc="openEditVpcModal"
          @create-vswitch="openCreateVswitchModal"
          @edit-vswitch="openEditVswitchModal"
          @create-security-group="openCreateSecurityGroupModal"
          @edit-security-group="openEditSecurityGroupModal"
        />

        <div class="flex justify-end space-x-3 px-6 py-4 flex-shrink-0 border-t border-gray-100">
          <button
            type="button"
            class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300"
            @click="handleClose"
          >
            取消
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300"
          >
            {{ loading ? '保存中...' : '保存默认配置' }}
          </button>
        </div>
      </form>

      <SetDefaultConfigResourceModals
        :form-data="formData"
        :vpcs="vpcs"
        :show-create-vpc-modal="showCreateVpcModal"
        :show-edit-vpc-modal="showEditVpcModal"
        :show-create-vswitch-modal="showCreateVswitchModal"
        :show-edit-vswitch-modal="showEditVswitchModal"
        :show-create-security-group-modal="showCreateSecurityGroupModal"
        :show-edit-security-group-modal="showEditSecurityGroupModal"
        :editing-vpc-id="editingVpcId"
        :editing-vpc-name="editingVpcName"
        :editing-vpc-cidr="editingVpcCidr"
        :editing-vswitch-id="editingVswitchId"
        :editing-vswitch-name="editingVswitchName"
        :editing-vswitch-zone="editingVswitchZone"
        :editing-vswitch-cidr="editingVswitchCidr"
        :editing-security-group-id="editingSecurityGroupId"
        :editing-security-group-name="editingSecurityGroupName"
        :editing-security-group-description="editingSecurityGroupDescription"
        :editing-security-group-inbound-rules="editingSecurityGroupInboundRules"
        :editing-security-group-outbound-rules="editingSecurityGroupOutboundRules"
        @close-vpc="closeVpcModal"
        @close-vswitch="closeVswitchModal"
        @close-security-group="closeSecurityGroupModal"
        @vpc-created="handleVpcCreated"
        @vswitch-created="handleVswitchCreated"
        @security-group-created="handleSecurityGroupCreated"
      />
    </div>
  </div>
</template>

<script setup>
import { useSetDefaultConfigForm } from '../../composables/useSetDefaultConfigForm.js'
import SetDefaultConfigFormFields from './SetDefaultConfigFormFields.vue'
import SetDefaultConfigResourceModals from './SetDefaultConfigResourceModals.vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  initialData: {
    type: Object,
    default: () => ({
      authorization_id: '',
      platform_type: ''
    })
  }
})

const emit = defineEmits(['close', 'saved', 'config-updated'])

const {
  formData,
  loading,
  loadingRegions,
  regionError,
  loadingVpcs,
  loadingVswitches,
  loadingSecurityGroups,
  regions,
  vpcs,
  vswitches,
  securityGroups,
  instances,
  loadingInstances,
  instanceError,
  showCreateVpcModal,
  showCreateVswitchModal,
  showCreateSecurityGroupModal,
  showEditVpcModal,
  showEditVswitchModal,
  showEditSecurityGroupModal,
  editingVpcId,
  editingVpcName,
  editingVpcCidr,
  editingVswitchId,
  editingVswitchName,
  editingVswitchZone,
  editingVswitchCidr,
  editingSecurityGroupId,
  editingSecurityGroupName,
  editingSecurityGroupDescription,
  editingSecurityGroupInboundRules,
  editingSecurityGroupOutboundRules,
  handleRegionChange,
  handleVpcChange,
  handleInstanceFilterChange,
  loadInstances,
  openCreateVpcModal,
  openCreateVswitchModal,
  openCreateSecurityGroupModal,
  openEditVpcModal,
  openEditVswitchModal,
  openEditSecurityGroupModal,
  handleVpcCreated,
  handleVswitchCreated,
  handleSecurityGroupCreated,
  closeVpcModal,
  closeVswitchModal,
  closeSecurityGroupModal,
  handleClose,
  handleSubmit
} = useSetDefaultConfigForm(props, emit)
</script>
