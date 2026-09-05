<template>
  <div class="modal-select-scroll-body space-y-4 px-6 overflow-y-auto flex-1 min-h-0">
    <input type="hidden" v-model="formData.authorization_id">
    <input type="hidden" v-model="formData.platform_type">

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">
        区域
        <span v-if="loadingRegions" class="inline-block ml-2 animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-gray-400"></span>
      </label>
      <select
        v-model="formData.region"
        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
        required
        :disabled="loadingRegions"
        @change="$emit('region-change')"
      >
        <option value="">请选择区域</option>
        <option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</option>
      </select>
      <p v-if="regionError && !loadingRegions" class="mt-1 text-sm text-red-600">{{ regionError }}</p>
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">
        VPC ID（专有网络ID）
        <span v-if="loadingVpcs" class="inline-block ml-2 animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-gray-400"></span>
      </label>
      <div class="flex space-x-2">
        <select
          v-model="formData.vpc_id"
          class="flex-1 px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
          required
          :disabled="loadingVpcs"
          @change="$emit('vpc-change')"
        >
          <option value="">请选择VPC</option>
          <option v-for="vpc in vpcs" :key="vpc.id" :value="vpc.id">{{ vpc.name }}</option>
        </select>
        <button
          type="button"
          :class="[
            'px-4 py-3 rounded-lg transition-colors whitespace-nowrap',
            (loadingVpcs || !formData.vpc_id)
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-blue-600 text-white hover:bg-blue-700'
          ]"
          :disabled="loadingVpcs || !formData.vpc_id"
          @click="$emit('edit-vpc')"
        >
          编辑
        </button>
        <button
          type="button"
          :class="[
            'px-4 py-3 rounded-lg transition-colors whitespace-nowrap',
            (loadingVpcs || !formData.region)
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-green-600 text-white hover:bg-green-700'
          ]"
          :disabled="loadingVpcs || !formData.region"
          @click="$emit('create-vpc')"
        >
          创建
        </button>
      </div>
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">
        交换机ID
        <span v-if="loadingVswitches" class="inline-block ml-2 animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-gray-400"></span>
      </label>
      <div class="flex space-x-2">
        <select
          v-model="formData.vswitch_id"
          class="flex-1 px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
          required
          :disabled="loadingVswitches"
        >
          <option value="">请选择交换机</option>
          <option v-for="vswitch in vswitches" :key="vswitch.id" :value="vswitch.id">{{ vswitch.name }}</option>
        </select>
        <button
          type="button"
          :class="[
            'px-4 py-3 rounded-lg transition-colors whitespace-nowrap',
            (loadingVswitches || !formData.vswitch_id)
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-blue-600 text-white hover:bg-blue-700'
          ]"
          :disabled="loadingVswitches || !formData.vswitch_id"
          @click="$emit('edit-vswitch')"
        >
          编辑
        </button>
        <button
          type="button"
          :class="[
            'px-4 py-3 rounded-lg transition-colors whitespace-nowrap',
            (loadingVswitches || !formData.vpc_id)
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-green-600 text-white hover:bg-green-700'
          ]"
          :disabled="loadingVswitches || !formData.vpc_id"
          @click="$emit('create-vswitch')"
        >
          创建
        </button>
      </div>
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">
        安全组ID
        <span v-if="loadingSecurityGroups" class="inline-block ml-2 animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-gray-400"></span>
      </label>
      <div class="flex space-x-2">
        <select
          v-model="formData.security_group_id"
          class="flex-1 px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
          required
          :disabled="loadingSecurityGroups"
        >
          <option value="">请选择安全组</option>
          <option v-for="sg in securityGroups" :key="sg.id" :value="sg.id">{{ sg.name }}</option>
        </select>
        <button
          type="button"
          :class="[
            'px-4 py-3 rounded-lg transition-colors whitespace-nowrap',
            (loadingSecurityGroups || !formData.security_group_id)
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-blue-600 text-white hover:bg-blue-700'
          ]"
          :disabled="loadingSecurityGroups || !formData.security_group_id"
          @click="$emit('edit-security-group')"
        >
          编辑
        </button>
        <button
          type="button"
          :class="[
            'px-4 py-3 rounded-lg transition-colors whitespace-nowrap',
            (loadingSecurityGroups || !formData.vpc_id)
              ? 'bg-gray-400 text-gray-200 cursor-not-allowed'
              : 'bg-green-600 text-white hover:bg-green-700'
          ]"
          :disabled="loadingSecurityGroups || !formData.vpc_id"
          @click="$emit('create-security-group')"
        >
          创建
        </button>
      </div>
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">付费类型</label>
      <select
        v-model="formData.payment_type"
        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
        required
      >
        <option value="">请选择付费类型</option>
        <option value="PrePaid">预付费</option>
        <option value="PostPaid">后付费</option>
      </select>
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">带宽计费模式</label>
      <select
        v-model="formData.bandwidth_charging_mode"
        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
        required
      >
        <option value="">请选择带宽计费模式</option>
        <option value="PayByBandwidth">按带宽计费</option>
        <option value="PayByTraffic">按流量计费</option>
      </select>
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">带宽值 (Mbps)</label>
      <input
        v-model="formData.bandwidth"
        type="number"
        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
        placeholder="带宽值"
        required
        min="1"
      >
    </div>

    <!-- 默认硬件配置 -->
    <div class="border-t border-gray-200 pt-4">
      <h4 class="text-sm font-semibold text-gray-700 mb-3">默认硬件配置</h4>
      <p class="text-xs text-gray-500 mb-3">
        设置默认的实例规格筛选条件；项目「快速应用默认模版」将读取这些配置。
      </p>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">CPU 核心数</label>
          <input
            v-model="formData.cpu_cores"
            type="number"
            class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            placeholder="如 2"
            min="1"
            @change="$emit('instance-filter-change')"
          >
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">内存 (GB)</label>
          <input
            v-model="formData.memory_gb"
            type="number"
            class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            placeholder="如 4"
            min="1"
            @change="$emit('instance-filter-change')"
          >
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 mt-3">
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">IoOptimized</label>
          <div class="flex items-center h-[30px]">
            <input
              id="io-optimized"
              type="checkbox"
              class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              :checked="formData.io_optimized === 'optimized'"
              @change="formData.io_optimized = $event.target.checked ? 'optimized' : 'none'; $emit('instance-filter-change')"
            >
            <label for="io-optimized" class="ml-2 text-sm text-gray-700">优化实例</label>
          </div>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">竞价策略</label>
          <select
            v-model="formData.spot_strategy"
            class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @change="$emit('instance-filter-change')"
          >
            <option value="">-- 请选择 --</option>
            <option value="NoSpot">正常按量付费</option>
            <option value="SpotWithPriceLimit">设置上限价格</option>
            <option value="SpotAsPriceGo">系统自动出价</option>
          </select>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 mt-3">
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">系统盘类型</label>
          <select
            v-model="formData.system_disk_category"
            class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @change="$emit('instance-filter-change')"
          >
            <option value="cloud_efficiency">高效云盘</option>
            <option value="cloud_ssd">SSD 云盘</option>
            <option value="cloud_essd">ESSD 云盘</option>
            <option value="cloud_auto">ESSD AutoPL 云盘</option>
            <option value="cloud_essd_entry">ESSD Entry 云盘</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">数据盘类型</label>
          <select
            v-model="formData.data_disk_category"
            data-testid="data-disk-category-select"
            class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @change="$emit('instance-filter-change')"
          >
            <option value="">不要数据盘</option>
            <option value="cloud_efficiency">高效云盘</option>
            <option value="cloud_ssd">SSD 云盘</option>
            <option value="cloud_essd">ESSD 云盘</option>
            <option value="cloud_auto">ESSD AutoPL 云盘</option>
            <option value="cloud_essd_entry">ESSD Entry 云盘</option>
          </select>
        </div>
      </div>

      <div class="mt-3">
        <label class="block text-sm font-medium text-gray-700 mb-2">
          默认实例类型
          <span v-if="loadingInstances" class="inline-block ml-2 animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-gray-400"></span>
        </label>
        <select
          v-model="formData.instance_type"
          class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
          :disabled="loadingInstances || instances.length === 0"
        >
          <option value="">请选择默认实例类型</option>
          <option
            v-for="inst in instances"
            :key="inst.instance_type"
            :value="inst.instance_type"
          >
            {{ inst.instance_type }}
            <template v-if="inst.cpu_cores || inst.memory_gb">
              ({{ inst.cpu_cores }}核 / {{ inst.memory_gb }}GB
              <template v-if="inst.gpu_cores"> / {{ inst.gpu_cores }}GPU</template>)
            </template>
          </option>
        </select>
        <p v-if="instanceError && !loadingInstances" class="mt-1 text-sm text-red-600">{{ instanceError }}</p>
        <p v-else-if="!loadingInstances && instances.length === 0 && formData.region && formData.vpc_id" class="mt-1 text-sm text-amber-600">
          当前区域和筛选条件下暂无可用实例，请调整 CPU/内存筛选条件
        </p>
        <p v-else-if="!formData.region" class="mt-1 text-sm text-gray-400">
          请先选择区域和 VPC 以加载可用实例
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  formData: {
    type: Object,
    required: true
  },
  regions: {
    type: Array,
    default: () => []
  },
  vpcs: {
    type: Array,
    default: () => []
  },
  vswitches: {
    type: Array,
    default: () => []
  },
  securityGroups: {
    type: Array,
    default: () => []
  },
  instances: {
    type: Array,
    default: () => []
  },
  loadingRegions: {
    type: Boolean,
    default: false
  },
  regionError: {
    type: String,
    default: ''
  },
  loadingVpcs: {
    type: Boolean,
    default: false
  },
  loadingVswitches: {
    type: Boolean,
    default: false
  },
  loadingSecurityGroups: {
    type: Boolean,
    default: false
  },
  loadingInstances: {
    type: Boolean,
    default: false
  },
  instanceError: {
    type: String,
    default: ''
  }
})

defineEmits([
  'region-change',
  'vpc-change',
  'instance-filter-change',
  'create-vpc',
  'edit-vpc',
  'create-vswitch',
  'edit-vswitch',
  'create-security-group',
  'edit-security-group'
])
</script>
