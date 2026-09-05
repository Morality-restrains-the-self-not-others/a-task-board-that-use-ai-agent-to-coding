import { createAvailableInstancesFetchScheduler } from '../../utils/availableInstancesFetchCoordinator.js'
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { projectHasConfiguredRunTemplate } from '../../utils/projectRunTemplateUtils.js'
import { resolveServerStartDisabledReason } from '../../utils/serverStartDisabledReason.js'
import {
  resolvePrimaryImageArchitecture,
} from '../../utils/containerImageArchitecture.js'
import { resolveServerConfigTaskId } from '../../utils/serverConfigRouteHelpers.js'

export function createHardwarePanelCtx(props, emit, selectedImageId) {
  const route = useRoute()
  const ctx = {
    props,
    emit,
    selectedImageId,
    route,
    loadRegionsPromise: null,
    loadRegionsPromiseImageKey: '',
    loadRegionsGeneration: 0,
    isLoadingVpcsFlag: false,
    isLoadingZonesFlag: false,
    isLoadingPlatformsFlag: false,
    loadCloudPlatformsPromise: null,
    isLoadingDefaultConfigsFlag: false,
    loadDefaultConfigsPromise: null,
    isFetchingInstances: false,
    availableInstancesFetchGeneration: 0,
    availableInstancesAbortController: null,
    availableInstancesDetailsAbortController: null,
    lastApplyRunTemplateError: '',
  }

  const resolveTenantId = () => String(props.tenantId || route.params.tenant || '').trim()
  const resolveWorkspaceId = () => String(props.workspaceId || route.params.workspaceId || '').trim()
  const resolveTaskId = () => resolveServerConfigTaskId({
    route,
    task: props.task,
    taskId: props.taskId,
    tenantId: props.tenantId,
    workspaceId: props.workspaceId,
  })
  const hasCloudContext = () => Boolean(resolveTenantId() && resolveWorkspaceId())

  const getRegionDisplayName = (regionId, fallbackName) => {
    return fallbackName || regionId
  }

  const installedImages = computed(() => (Array.isArray(props.installedImages) ? props.installedImages : []))

  const cloudPlatforms = ref([])
  const regions = ref([])
  const zones = ref([])
  const vpcs = ref([])
  const vswitches = ref([])
  const securityGroups = ref([])
  const selectedRegion = ref('')
  const selectedZone = ref('')
  const selectedVpc = ref('')
  const selectedSecurityGroup = ref('')
  const selectedCloudPlatform = ref('')
  const hardwareBootstrapDone = ref(false)

  const isLoadingPlatforms = ref(false)
  const isLoadingRegions = ref(false)
  const isLoadingVpcs = ref(false)
  const isLoadingZones = ref(false)

  // 防止重复请求的标志
  ctx.loadRegionsPromise = null
  // OPT-20260821-005：快速切换镜像时旧 loadRegions 请求不得被新镜像复用；
  // 记录请求发起时的镜像 id + 代际，过期结果整体丢弃。
  ctx.loadRegionsPromiseImageKey = ''
  ctx.loadRegionsGeneration = 0
  ctx.isLoadingVpcsFlag = false
  ctx.isLoadingZonesFlag = false

  const previousServerConfig = ref(null)
  const projectRunTemplateApplied = ref(false)

  // 面板展开 / 临时配置 / 项目模版摘要（拆分 composable 时须保留，否则 setup 抛 ReferenceError 导致整块硬件面板不挂载）
  const panelExpanded = ref(Boolean(props.runTemplateMode))
  /** 任务详情：用户点开「临时配置」后才展示可编辑硬件表单 */
  const temporaryConfigExpanded = ref(false)

  const showProjectTemplateSummaryOnly = computed(
    () =>
      !props.runTemplateMode
      && projectHasConfiguredRunTemplate({ server_run_template: props.projectServerRunTemplate })
      && !temporaryConfigExpanded.value,
  )

  const showHardwareConfigForm = computed(() => {
    if (props.runTemplateMode) return panelExpanded.value
    if (projectHasConfiguredRunTemplate({ server_run_template: props.projectServerRunTemplate })) {
      return temporaryConfigExpanded.value
    }
    return true
  })

  const useProjectTemplateForStart = computed(
    () => showProjectTemplateSummaryOnly.value,
  )

  const hardwareConfigSource = computed(() =>
    (useProjectTemplateForStart.value ? 'project_template' : 'temporary'),
  )

  const activeHardwareConfigSourceLabel = computed(() => {
    if (props.runTemplateMode) return ''
    if (hardwareConfigSource.value === 'project_template') {
      return '将使用关联项目的运行硬件模版启动'
    }
    if (temporaryConfigExpanded.value) {
      return '将使用本次临时硬件配置启动（不会保存到项目）'
    }
    return ''
  })

  const runTemplateMeta = ref({ template_id: '', label: '' })

  // 硬件配置
  const hardwareConfig = ref({
    cpu_cores: "1",
    memory_gb: "1",
    storage_gb: "40",
    generation: ""
  })

  // 按量实例自动释放（分钟，与后端 RunInstances AutoReleaseTime 对应）
  const autoReleaseEnabled = ref(true)
  const autoReleaseMinutes = ref(30)

  // 过滤选项
  const DEFAULT_SYSTEM_DISK_CATEGORY = 'cloud_essd'
  const DEFAULT_STORAGE_TYPES = Object.freeze([DEFAULT_SYSTEM_DISK_CATEGORY])

  const resolveInstanceStorageTypes = (instance, preferredCategory = DEFAULT_SYSTEM_DISK_CATEGORY) => {
    const types = instance?.diskSupport?.storage_types
      || instance?.storage_types
      || instance?.StorageTypes
      || instance?.SupportedStorageCategories
    if (Array.isArray(types) && types.length > 0) {
      return types
    }
    const single = instance?.storage_type || instance?.StorageType || preferredCategory
    return single ? [single] : [...DEFAULT_STORAGE_TYPES]
  }

  const resolveInstanceStorageType = (instance, preferredCategory = filterOptions.value.system_disk_category || DEFAULT_SYSTEM_DISK_CATEGORY) => {
    const fromInstance = instance?.storage_type || instance?.StorageType
    if (fromInstance) {
      return fromInstance
    }
    const types = resolveInstanceStorageTypes(instance, preferredCategory)
    return types[0] || preferredCategory || DEFAULT_SYSTEM_DISK_CATEGORY
  }

  const filterOptions = ref({
    // 基础过滤选项
    cores: '2',
    memory: '4',
    io_optimized: true,
    system_disk_category: 'cloud_essd',
    data_disk_category: 'cloud_essd',
    spot_strategy: 'SpotAsPriceGo',
    /** 与镜像 target_architectures 对齐，传给 available-instances 的 image_architecture */
    image_architecture: '',
    /** 与阿里云 RunInstances.SpotDuration 一致；抢占策略下多为 0 */
    spot_duration: 0,
    /** PostPaid 等，与 RunInstances.InstanceChargeType 对齐 */
    instance_charge_type: 'PostPaid',
    network_category: 'vpc'
  })

  // 云平台默认配置
  const cloudPlatformDefaultConfigs = ref({})

  const ensureDefaultConfigForPlatform = (platformId, fallbackAuthId = '') => {
    const pid = String(platformId || selectedCloudPlatform.value || '').trim()
    if (!pid) return null
    const existing = cloudPlatformDefaultConfigs.value[pid]
    if (existing?.authorization_id) return existing
    const platform = cloudPlatforms.value.find((p) => String(p.id) === pid)
    const authId = String(
      platform?.authorization_id ?? platform?.iam_id ?? fallbackAuthId ?? '',
    ).trim()
    if (!platform && !authId) return existing || null
    cloudPlatformDefaultConfigs.value[pid] = {
      platform_type: platform?.platform_type || existing?.platform_type || 'aliyun',
      authorization_id: authId,
      remark: platform?.remark || existing?.remark || '',
      config: existing?.config ?? null,
    }
    return cloudPlatformDefaultConfigs.value[pid]
  }

  const resolveCloudPlatformDefaultConfig = () => {
    const pid = String(selectedCloudPlatform.value || '').trim()
    if (!pid) return null
    const cfg = cloudPlatformDefaultConfigs.value[pid]
    if (cfg?.authorization_id) return cfg
    return ensureDefaultConfigForPlatform(pid)
  }
  // 防止重复请求的标志
  ctx.isLoadingPlatformsFlag = false
  ctx.loadCloudPlatformsPromise = null
  ctx.isLoadingDefaultConfigsFlag = false
  ctx.loadDefaultConfigsPromise = null

  // 可用实例列表
  const availableInstances = ref([])
  const isLoadingInstances = ref(false)
  // 选中的实例
  const selectedInstance = ref(null)
  /** 模版加载后待恢复的实例类型（可用实例列表异步拉取完成后应用） */
  const pendingRestoreInstanceType = ref('')
  // 分页信息
  const nextToken = ref(null)
  const isLoadingNextPage = ref(false)
  // 分页控制
  const currentPage = ref(1)
  const pageSize = ref(10)
  const totalPages = ref(1)
  // 页面数据缓存
  const pageDataCache = ref([])
  // 防止重复请求的标志
  ctx.isFetchingInstances = false
  // 可用实例列表请求：代际 + 取消，避免 debounce/模版加载期间过期响应覆盖最新结果
  ctx.availableInstancesFetchGeneration = 0
  ctx.availableInstancesAbortController = null
  ctx.availableInstancesDetailsAbortController = null

  function abortPendingAvailableInstancesFetch() {
    if (ctx.availableInstancesAbortController) {
      ctx.availableInstancesAbortController.abort()
      ctx.availableInstancesAbortController = null
    }
    abortPendingInstanceDetailsFetch()
  }

  function abortPendingInstanceDetailsFetch() {
    if (ctx.availableInstancesDetailsAbortController) {
      ctx.availableInstancesDetailsAbortController.abort()
      ctx.availableInstancesDetailsAbortController = null
    }
  }

  function releaseAvailableInstancesFetchState(fetchGeneration) {
    if (fetchGeneration !== ctx.availableInstancesFetchGeneration) {
      return
    }
    isLoadingInstances.value = false
    isLoadingNextPage.value = false
    ctx.isFetchingInstances = false
    ctx.availableInstancesAbortController = null
  }

  const availableInstancesFetchScheduler = createAvailableInstancesFetchScheduler({
    onRun: (intentGeneration) => {
      nextToken.value = null
      void ctx.fetchAvailableInstances(intentGeneration)
    },
  })

  function scheduleFetchAvailableInstances({ immediate = false } = {}) {
    availableInstancesFetchScheduler.schedule({
      immediate,
      onAbort: abortPendingAvailableInstancesFetch,
    })
  }
  // 价格信息缓存
  const instancePrices = ref({})
  // 价格加载状态
  const priceLoading = ref({})
  // 带宽限制信息
  const bandwidthLimitations = ref({})
  // 带宽加载状态
  const bandwidthLoading = ref({})
  // 选中的带宽值
  const selectedBandwidth = ref({})
  // 选中的带宽计费模式
  const selectedBandwidthChargingMode = ref('PayByTraffic')
  // 默认带宽值
  const DEFAULT_BANDWIDTH = 5
  // SSE连接由父组件TaskDetail管理
  // SDK方法调用记录
  const sdkMethods = ref([])
  // 计算属性
  /** 已选镜像推导出的 CPU 架构，用于实例列表过滤 */
  const imageArchitectureFromSelection = computed(() =>
    resolvePrimaryImageArchitecture({
      task: props.task,
      selectedImageId: selectedImageId.value,
      installedImages: installedImages.value,
    }),
  )

  const imageArchitectureDisplayLabel = computed(() => imageArchitectureFromSelection.value || '—')

  const imageArchitectureDisplayHint = computed(() => {
    if (imageArchitectureFromSelection.value) {
      return `与已选镜像 CPU 架构一致：${imageArchitectureFromSelection.value}`
    }
    return '请先选择已安装镜像以解析 CPU 架构'
  })

  function syncImageArchitectureFilterFromSelection() {
    filterOptions.value.image_architecture = imageArchitectureFromSelection.value || ''
  }

  /** 传给可用实例等云 API 的 CPU 架构，始终与镜像一致 */
  const containerImageArchitectureForCloudQuery = computed(() => imageArchitectureFromSelection.value)

  const isServerRuntimeRunning = computed(() => props.serverRuntimeStatus === 'Running')

  const startServerDisabledReason = computed(() =>
    resolveServerStartDisabledReason({
      hardwareConfigSource: hardwareConfigSource.value,
      isServerStarting: props.isServerStarting,
      isServerRunning: props.isServerRunning,
      isServerRuntimeRunning: isServerRuntimeRunning.value,
      serverRuntimeStatus: props.serverRuntimeStatus,
      hasCloudContext: hasCloudContext(),
      selectedImageId: selectedImageId.value,
      taskId: resolveTaskId(),
      runTemplate: props.projectServerRunTemplate,
      autoReleaseEnabled: autoReleaseEnabled.value,
      autoReleaseMinutes: autoReleaseMinutes.value,
      isLoadingPlatforms: isLoadingPlatforms.value,
      selectedCloudPlatform: selectedCloudPlatform.value,
      isLoadingRegions: isLoadingRegions.value,
      selectedRegion: selectedRegion.value,
      selectedVpc: selectedVpc.value,
      selectedZone: selectedZone.value,
      isLoadingInstances: isLoadingInstances.value,
      selectedInstance: selectedInstance.value,
      requireEnvParamsSource: props.requireEnvParamsSource,
      featureParamsSource: props.featureParamsSource,
      selectedPersonalConfigId: props.selectedPersonalConfigId,
    }),
  )

  const isStartServerDisabled = computed(() => Boolean(startServerDisabledReason.value))

  const startServerButtonLabel = computed(() => {
    if (props.isServerStarting) return '启动中...'
    if (props.isServerRunning || isServerRuntimeRunning.value) return '服务器运行中'
    const s = props.serverRuntimeStatus
    if (s === 'Starting' || s === 'Pending' || s === 'Initializing') return '实例创建中…'
    if (s === 'Stopping' || s === 'Rebooting') return '实例处理中…'
    return '启动服务器'
  })
  // 获取存储类型的最小容量
  const getStorageMin = (instance, storageType) => {
    if (instance.diskSupport && instance.diskSupport.storage_configs) {
      const config = instance.diskSupport.storage_configs.find(config => config.type === storageType);
      if (config) {
        return config.min;
      }
    }
    return 20; // 默认值
  }

  // 获取存储类型的最大容量
  const getStorageMax = (instance, storageType) => {
    if (instance.diskSupport && instance.diskSupport.storage_configs) {
      const config = instance.diskSupport.storage_configs.find(config => config.type === storageType);
      if (config) {
        return config.max;
      }
    }
    return 2048; // 默认值
  }

  const runTemplateMode = computed(() => props.runTemplateMode)
  const isServerRunning = computed(() => props.isServerRunning)
  const isServerStarting = computed(() => props.isServerStarting)
  const serverJumpUrl = computed(() => props.serverJumpUrl)
  const serverJumpDefaultPort = computed(() => props.serverJumpDefaultPort)


  Object.assign(ctx, {
    resolveTenantId,
    resolveWorkspaceId,
    resolveTaskId,
    hasCloudContext,
    getRegionDisplayName,
    installedImages,
    cloudPlatforms,
    regions,
    zones,
    vpcs,
    vswitches,
    securityGroups,
    selectedRegion,
    selectedZone,
    selectedVpc,
    selectedSecurityGroup,
    selectedCloudPlatform,
    hardwareBootstrapDone,
    isLoadingPlatforms,
    isLoadingRegions,
    isLoadingVpcs,
    isLoadingZones,
    previousServerConfig,
    projectRunTemplateApplied,
    panelExpanded,
    temporaryConfigExpanded,
    showProjectTemplateSummaryOnly,
    showHardwareConfigForm,
    useProjectTemplateForStart,
    hardwareConfigSource,
    activeHardwareConfigSourceLabel,
    runTemplateMeta,
    hardwareConfig,
    autoReleaseEnabled,
    autoReleaseMinutes,
    DEFAULT_SYSTEM_DISK_CATEGORY,
    DEFAULT_STORAGE_TYPES,
    resolveInstanceStorageTypes,
    resolveInstanceStorageType,
    filterOptions,
    cloudPlatformDefaultConfigs,
    ensureDefaultConfigForPlatform,
    resolveCloudPlatformDefaultConfig,
    availableInstances,
    isLoadingInstances,
    selectedInstance,
    pendingRestoreInstanceType,
    nextToken,
    isLoadingNextPage,
    currentPage,
    pageSize,
    totalPages,
    pageDataCache,
    abortPendingAvailableInstancesFetch,
    abortPendingInstanceDetailsFetch,
    releaseAvailableInstancesFetchState,
    availableInstancesFetchScheduler,
    scheduleFetchAvailableInstances,
    instancePrices,
    priceLoading,
    bandwidthLimitations,
    bandwidthLoading,
    selectedBandwidth,
    selectedBandwidthChargingMode,
    DEFAULT_BANDWIDTH,
    sdkMethods,
    imageArchitectureFromSelection,
    imageArchitectureDisplayLabel,
    imageArchitectureDisplayHint,
    syncImageArchitectureFilterFromSelection,
    containerImageArchitectureForCloudQuery,
    isServerRuntimeRunning,
    startServerDisabledReason,
    isStartServerDisabled,
    startServerButtonLabel,
    getStorageMin,
    getStorageMax,
    runTemplateMode,
    isServerRunning,
    isServerStarting,
    serverJumpUrl,
    serverJumpDefaultPort,
  })

  return ctx
}
