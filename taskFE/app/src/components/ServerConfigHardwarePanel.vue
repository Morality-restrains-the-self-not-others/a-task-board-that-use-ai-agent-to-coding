<template>
  <div
    :id="sectionId"
    data-testid="server-hardware-config-panel"
    :class="rootClass"
  >
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-sm font-medium text-gray-500">服务器硬件配置</h3>
      <HardwarePanelHeader :panel="panel" />
    </div>

    <HardwareProjectTemplateBanner
      v-if="panel.showProjectTemplateSummaryOnly"
      :panel="panel"
    />

    <div v-show="panel.showHardwareConfigForm">
      <HardwareTemporaryConfigToolbar
        v-if="!props.runTemplateMode && panel.temporaryConfigExpanded && panel.hasConfiguredProjectRunTemplate()"
        :panel="panel"
      />
      <HardwareNetworkSelectors :panel="panel" />
      <HardwareInstanceFilters :panel="panel" />
      <HardwareInstanceListPanel :panel="panel" />
      <HardwareSelectedInstancePrice :panel="panel" />
    </div>

    <HardwareAutoReleaseSettings
      v-if="!props.runTemplateMode"
      :panel="panel"
    />
  </div>
</template>

<script setup>
/**
 * 任务详情「服务器硬件配置」组合壳：子组件 + useServerConfigHardwarePanel。
 */
import { reactive, toRef } from 'vue'
import { useServerConfigHardwarePanel } from '../composables/hardwarePanel/useServerConfigHardwarePanel.js'
import HardwarePanelHeader from './hardware-panel/HardwarePanelHeader.vue'
import HardwareProjectTemplateBanner from './hardware-panel/HardwareProjectTemplateBanner.vue'
import HardwareTemporaryConfigToolbar from './hardware-panel/HardwareTemporaryConfigToolbar.vue'
import HardwareNetworkSelectors from './hardware-panel/HardwareNetworkSelectors.vue'
import HardwareInstanceFilters from './hardware-panel/HardwareInstanceFilters.vue'
import HardwareInstanceListPanel from './hardware-panel/HardwareInstanceListPanel.vue'
import HardwareSelectedInstancePrice from './hardware-panel/HardwareSelectedInstancePrice.vue'
import HardwareAutoReleaseSettings from './hardware-panel/HardwareAutoReleaseSettings.vue'

const props = defineProps({
  task: { type: Object, default: null },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: null },
  installedImages: { type: Array, default: () => [] },
  updateServerStatus: { type: Function, default: null },
  projectServerRunTemplate: { type: Object, default: null },
  requireEnvParamsSource: { type: Boolean, default: false },
  featureParamsSource: { type: String, default: '' },
  selectedPersonalConfigId: { type: String, default: '' },
  runTemplateMode: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
  sectionId: { type: String, default: 'hardware-config-section' },
  /** 嵌于镜像卡内：去掉外层卡片边框，用顶部分隔线区分 */
  embedded: { type: Boolean, default: false },
})

const emit = defineEmits(['spec-summary-change'])
const selectedImageId = defineModel('selectedImageId', { type: String, default: '' })

const rootClass = props.embedded
  ? 'mt-4 pt-4 border-t border-gray-200'
  : 'p-4 bg-white border border-gray-200 rounded-lg'

const panel = reactive(useServerConfigHardwarePanel(props, emit, selectedImageId))

defineExpose({
  initHardwareFromParent: (...a) => panel.initHardwareFromParent(...a),
  buildRunTemplatePayload: (...a) => panel.buildRunTemplatePayload(...a),
  applyRunTemplate: (...a) => panel.applyRunTemplate(...a),
  bootstrapManualConfig: (...a) => panel.bootstrapManualConfig(...a),
  getLastApplyRunTemplateError: (...a) => panel.getLastApplyRunTemplateError(...a),
  openTemporaryConfig: (...a) => panel.openTemporaryConfig(...a),
  closeTemporaryConfig: (...a) => panel.closeTemporaryConfig(...a),
  restoreProjectTemplateDefaults: (...a) => panel.restoreProjectTemplateDefaults(...a),
  getHardwareConfigSource: () => panel.hardwareConfigSource,
  hardwareConfigSource: toRef(panel, 'hardwareConfigSource'),
  scheduleFetchAvailableInstances: (...a) => panel.scheduleFetchAvailableInstances(...a),
  syncImageArchitectureFilterFromSelection: (...a) => panel.syncImageArchitectureFilterFromSelection(...a),
})
</script>
