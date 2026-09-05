<template>
  <CreateVpcModal
    :visible="showCreateVpcModal || showEditVpcModal"
    :initial-data="{
      authorization_id: formData.authorization_id,
      platform_type: formData.platform_type,
      region: formData.region,
      vpc_id: editingVpcId,
      vpc_name: editingVpcName,
      cidr_block: editingVpcCidr
    }"
    @close="$emit('close-vpc')"
    @created="$emit('vpc-created')"
  />

  <CreateVswitchModal
    :visible="showCreateVswitchModal || showEditVswitchModal"
    :initial-data="{
      authorization_id: formData.authorization_id,
      platform_type: formData.platform_type,
      vpc_id: formData.vpc_id,
      vpc_name: selectedVpcName,
      region: formData.region,
      vpc_cidr_block: selectedVpcCidr,
      vswitch_id: editingVswitchId,
      vswitch_name: editingVswitchName,
      zone_id: editingVswitchZone,
      cidr_block: editingVswitchCidr
    }"
    @close="$emit('close-vswitch')"
    @created="$emit('vswitch-created')"
  />

  <CreateSecurityGroupModal
    :visible="showCreateSecurityGroupModal || showEditSecurityGroupModal"
    :initial-data="{
      authorization_id: formData.authorization_id,
      platform_type: formData.platform_type,
      vpc_id: formData.vpc_id,
      vpc_name: selectedVpcName,
      region: formData.region,
      security_group_id: editingSecurityGroupId,
      security_group_name: editingSecurityGroupName,
      description: editingSecurityGroupDescription,
      inbound_rules: editingSecurityGroupInboundRules,
      outbound_rules: editingSecurityGroupOutboundRules
    }"
    @close="$emit('close-security-group')"
    @created="$emit('security-group-created')"
  />
</template>

<script setup>
import { computed } from 'vue'
import CreateVpcModal from './CreateVpcModal.vue'
import CreateVswitchModal from './CreateVswitchModal.vue'
import CreateSecurityGroupModal from './CreateSecurityGroupModal.vue'

const props = defineProps({
  formData: {
    type: Object,
    required: true
  },
  vpcs: {
    type: Array,
    default: () => []
  },
  showCreateVpcModal: {
    type: Boolean,
    default: false
  },
  showEditVpcModal: {
    type: Boolean,
    default: false
  },
  showCreateVswitchModal: {
    type: Boolean,
    default: false
  },
  showEditVswitchModal: {
    type: Boolean,
    default: false
  },
  showCreateSecurityGroupModal: {
    type: Boolean,
    default: false
  },
  showEditSecurityGroupModal: {
    type: Boolean,
    default: false
  },
  editingVpcId: {
    type: String,
    default: ''
  },
  editingVpcName: {
    type: String,
    default: ''
  },
  editingVpcCidr: {
    type: String,
    default: ''
  },
  editingVswitchId: {
    type: String,
    default: ''
  },
  editingVswitchName: {
    type: String,
    default: ''
  },
  editingVswitchZone: {
    type: String,
    default: ''
  },
  editingVswitchCidr: {
    type: String,
    default: ''
  },
  editingSecurityGroupId: {
    type: String,
    default: ''
  },
  editingSecurityGroupName: {
    type: String,
    default: ''
  },
  editingSecurityGroupDescription: {
    type: String,
    default: ''
  },
  editingSecurityGroupInboundRules: {
    type: Array,
    default: () => []
  },
  editingSecurityGroupOutboundRules: {
    type: Array,
    default: () => []
  }
})

defineEmits([
  'close-vpc',
  'close-vswitch',
  'close-security-group',
  'vpc-created',
  'vswitch-created',
  'security-group-created'
])

const selectedVpc = computed(() =>
  props.vpcs.find((vpc) => vpc.id === props.formData.vpc_id)
)

const selectedVpcName = computed(() => selectedVpc.value?.name || '')
const selectedVpcCidr = computed(() => selectedVpc.value?.cidr_block || '')
</script>
