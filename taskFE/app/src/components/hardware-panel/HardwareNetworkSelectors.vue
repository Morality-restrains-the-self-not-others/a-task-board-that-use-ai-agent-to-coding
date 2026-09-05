<template>
      <!-- 网络配置选择：云平台、地域、VPC、安全组、可用区 -->
      <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-2">
        <!-- 云平台选择 -->
        <div class="flex-shrink-0 min-w-[140px]">
          <label for="cloud-platform-select" class="block text-xs font-medium text-gray-500 mb-1">
            云平台
            <span v-if="panel.isLoadingPlatforms" class="inline-block animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full ml-1"></span>
          </label>
          <select
            id="cloud-platform-select"
            v-model="panel.selectedCloudPlatform"
            class="w-full px-2 py-1 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
            :disabled="panel.isLoadingPlatforms || panel.cloudPlatforms.length === 0"
            @change="panel.handleCloudPlatformChange"
          >
            <option v-if="panel.isLoadingPlatforms" disabled value="">加载中...</option>
            <option v-else-if="!panel.hasCloudContext()" disabled value="">等待工作空间上下文...</option>
            <option v-else-if="!panel.cloudPlatforms.length" disabled value="">暂无云平台授权</option>
            <option v-else disabled value="">-- 请选择 --</option>
            <option
              v-for="platform in panel.cloudPlatforms"
              :key="platform.id"
              :value="String(platform.id)"
            >
              {{ platform.remark ? `${platform.remark}-${getPlatformLabel(platform.platform_type)}` : getPlatformLabel(platform.platform_type) }}
            </option>
          </select>
        </div>
        <!-- 地域选择 -->
        <div class="flex-shrink-0 min-w-[140px]">
          <label for="region-select" class="block text-xs font-medium text-gray-500 mb-1">
            地域
            <span v-if="panel.isLoadingRegions" class="inline-block animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full ml-1"></span>
          </label>
          <select
            id="region-select"
            v-model="panel.selectedRegion"
            class="w-full px-2 py-1 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
            :disabled="!panel.selectedCloudPlatform || panel.isLoadingRegions || panel.regions.length === 0"
            @change="panel.loadVpcs"
          >
            <option v-if="panel.isLoadingRegions" disabled value="">加载中...</option>
            <option v-else-if="!panel.regions.length" disabled value="">请先选择云平台</option>
            <option v-else disabled value="">-- 请选择 --</option>
            <option
              v-for="region in panel.regions"
              :key="region.region_id"
              :value="String(region.region_id)"
            >
              {{ region.region_name }}
            </option>
          </select>
        </div>
        <!-- VPC选择 -->
        <div class="flex-shrink-0 min-w-[140px]">
          <label for="vpc-select" class="block text-xs font-medium text-gray-500 mb-1">
            VPC
            <span v-if="panel.isLoadingVpcs" class="inline-block animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full ml-1"></span>
          </label>
          <select
            id="vpc-select"
            v-model="panel.selectedVpc"
            class="w-full px-2 py-1 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
            :disabled="!panel.selectedRegion || panel.isLoadingVpcs || panel.vpcs.length === 0"
            @change="panel.loadVswitches"
          >
            <option v-if="panel.isLoadingVpcs" disabled value="">加载中...</option>
            <option v-else-if="!panel.vpcs.length" disabled value="">请先选择地域</option>
            <option v-else disabled value="">-- 请选择 --</option>
            <option v-for="vpc in panel.vpcs" :key="vpc.id" :value="String(vpc.id)">{{ vpc.name }}</option>
          </select>
        </div>
        <!-- 安全组选择 -->
        <div class="flex-shrink-0 min-w-[140px]">
          <label for="security-group-select" class="block text-xs font-medium text-gray-500 mb-1">
            安全组
          </label>
          <select id="security-group-select" v-model="panel.selectedSecurityGroup" class="w-full px-2 py-1 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary" :disabled="!panel.selectedVpc">
            <option value="">-- 请选择 --</option>
            <option v-for="sg in panel.securityGroups" :key="sg.id" :value="String(sg.id)">{{ sg.name }}</option>
          </select>
        </div>
        <!-- 可用区选择 -->
        <div class="flex-shrink-0 min-w-[140px]">
          <label for="zone-select" class="block text-xs font-medium text-gray-500 mb-1">
            可用区
            <span v-if="panel.isLoadingZones" class="inline-block animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full ml-1"></span>
          </label>
          <select
            id="zone-select"
            v-model="panel.selectedZone"
            class="w-full px-2 py-1 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
            :disabled="!panel.selectedVpc || panel.isLoadingZones || panel.zones.length === 0"
          >
            <option v-if="panel.isLoadingZones" disabled value="">加载中...</option>
            <option v-else-if="!panel.zones.length" disabled value="">请先选择 VPC</option>
            <option v-else disabled value="">-- 请选择 --</option>
            <option
              v-for="zone in panel.zones"
              :key="zone.zone_id"
              :value="String(zone.zone_id)"
            >
              {{ zone.display_name }}
            </option>
          </select>
        </div>
      </div>
      
      <div
        v-if="panel.selectedSecurityGroup === 'auto_create_security_group'"
        class="mt-2 text-xs text-amber-800 bg-amber-50 border border-amber-100 rounded-md px-2 py-1.5"
        role="alert"
      >
        <span class="font-medium">安全提示：</span>选择「自动创建安全组」时，入网仅允许「您当前的公网 IP」与「服务器公网 IP」，其余地址禁止访问。请确认本机公网 IP 未变更；如需开放更多来源，请改用已有安全组或在云控制台调整规则。
      </div>
      
</template>

<script setup>
const PLATFORM_LABELS = {
  aliyun: '阿里云',
  tencentcloud: '腾讯云',
  huaweicloud: '华为云',
  ctyun: '天翼云',
  cmcc: '移动云',
  cucloud: '联通云',
  baiducloud: '百度智能云',
  aws: 'AWS',
  jdcloud: '京东云',
}

const getPlatformLabel = (type) => PLATFORM_LABELS[type] || type

defineProps({
  panel: { type: Object, required: true },
})
</script>
