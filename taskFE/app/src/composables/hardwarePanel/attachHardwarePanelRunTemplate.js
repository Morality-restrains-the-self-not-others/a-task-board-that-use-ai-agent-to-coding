import { computed } from 'vue'
import {
  runTemplateToServerConfigShape,
  hardwarePanelStateToRunTemplate,
  resolveRunTemplateInstanceType,
  summarizeRunTemplate,
  projectHasConfiguredRunTemplate,
} from '../../utils/projectRunTemplateUtils.js'

export function attachHardwarePanelRunTemplate(ctx) {
  const {
    props,
    emit,
    selectedImageId,
    route,
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
    filterOptions,
    cloudPlatformDefaultConfigs,
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
    instancePrices,
    priceLoading,
    bandwidthLimitations,
    bandwidthLoading,
    selectedBandwidth,
    selectedBandwidthChargingMode,
    sdkMethods,
    imageArchitectureFromSelection,
    imageArchitectureDisplayLabel,
    imageArchitectureDisplayHint,
    containerImageArchitectureForCloudQuery,
    isServerRuntimeRunning,
    startServerDisabledReason,
    isStartServerDisabled,
    startServerButtonLabel,
    installedImages,
    runTemplateMode,
    isServerRunning,
    isServerStarting,
    serverJumpUrl,
    serverJumpDefaultPort,
    availableInstancesFetchScheduler,
  } = ctx

  ctx.applyServerConfigShape = async (serverConfig, { setPreviousRef = false, skipFinalize = false } = {}) => {
    if (!serverConfig) return
    if (setPreviousRef) {
      previousServerConfig.value = serverConfig
    }
    if (serverConfig.hardware_config) {
      hardwareConfig.value = {
        cpu_cores: serverConfig.hardware_config.cpu_cores || '1',
        memory_gb: serverConfig.hardware_config.memory_gb || '1',
        storage_gb: serverConfig.hardware_config.storage_gb || '40',
      }
      if (serverConfig.hardware_config.cpu_cores) {
        filterOptions.value.cores = String(serverConfig.hardware_config.cpu_cores)
      }
      if (serverConfig.hardware_config.memory_gb) {
        filterOptions.value.memory = String(serverConfig.hardware_config.memory_gb)
      }
    }
    const filterOptionsFromTemplate = serverConfig.filter_options
    if (filterOptionsFromTemplate && typeof filterOptionsFromTemplate === 'object') {
      const { image_architecture: _savedImageArch, ...restFilterOptions } = filterOptionsFromTemplate
      filterOptions.value = {
        ...filterOptions.value,
        ...restFilterOptions,
      }
    }
    ctx.syncImageArchitectureFilterFromSelection()
    if (serverConfig.platform_id) {
      selectedCloudPlatform.value = String(serverConfig.platform_id)
      ctx.ensureDefaultConfigForPlatform(serverConfig.platform_id, serverConfig.authorization_id)
    }
    if (serverConfig.region) {
      selectedRegion.value = String(serverConfig.region)
      if (!regions.value.length) {
        await ctx.loadRegions()
      }
      await ctx.loadVpcs()
      if (serverConfig.vpc_id) {
        selectedVpc.value = String(serverConfig.vpc_id)
        await ctx.loadVswitches()
      }
      if (serverConfig.zone_id) {
        selectedZone.value = serverConfig.vswitch_id
          ? `${String(serverConfig.zone_id)}:${String(serverConfig.vswitch_id)}`
          : String(serverConfig.zone_id)
      }
    }
    if (serverConfig.security_group_id) {
      selectedSecurityGroup.value = String(serverConfig.security_group_id)
    }
    if (!skipFinalize) {
      await ctx.finalizeServerConfigShape(serverConfig)
    }
  }

  ctx.resolveShapePlatformId = (shape, template) => {
    const direct = String(
      shape.platform_id || template?.cloud_platform_id || template?.platform_id || '',
    ).trim()
    if (direct) {
      const byId = cloudPlatforms.value.find((p) => String(p.id) === direct)
      if (byId) {
        shape.platform_id = direct
        const liveAuth = String(byId?.authorization_id ?? byId?.iam_id ?? '').trim()
        if (liveAuth) shape.authorization_id = liveAuth
        return true
      }
    }
    const authId = String(shape.authorization_id || template?.authorization_id || '').trim()
    const platformType = String(shape.platform || template?.platform || '').trim().toLowerCase()
    if (cloudPlatforms.value.length) {
      let matched = null
      if (authId) {
        matched = cloudPlatforms.value.find((p) => {
          const pAuth = String(p?.authorization_id ?? p?.iam_id ?? '').trim()
          return pAuth === authId
        })
      }
      if (!matched && platformType) {
        matched = cloudPlatforms.value.find(
          (p) => String(p?.platform_type || '').trim().toLowerCase() === platformType,
        )
      }
      if (!matched && cloudPlatforms.value.length === 1) {
        matched = cloudPlatforms.value[0]
      }
      if (matched?.id != null) {
        shape.platform_id = String(matched.id)
        const liveAuth = String(matched?.authorization_id ?? matched?.iam_id ?? '').trim()
        if (liveAuth) shape.authorization_id = liveAuth
        return true
      }
    }
    if (direct) {
      shape.platform_id = direct
      return true
    }
    return false
  }

  ctx.projectTemplateBannerText = computed(() => {
    const tpl = props.projectServerRunTemplate
    if (!tpl || typeof tpl !== 'object' || !Object.keys(tpl).length) return ''
    const label = summarizeRunTemplate(tpl)
    if (label === '未设置') return ''
    return `关联项目的运行硬件模版：${label}`
  })

  ctx.restoreProjectTemplateDefaults = async () => {
    projectRunTemplateApplied.value = false
    await ctx.applyProjectRunTemplateIfNeeded()
  }

  ctx.openTemporaryConfig = async () => {
    temporaryConfigExpanded.value = true
    projectRunTemplateApplied.value = false
    await ctx.applyProjectRunTemplateIfNeeded()
    if (!selectedCloudPlatform.value && cloudPlatforms.value.length > 0) {
      selectedCloudPlatform.value = String(cloudPlatforms.value[0].id)
      await ctx.handleCloudPlatformChange()
    }
  }

  ctx.closeTemporaryConfig = () => {
    temporaryConfigExpanded.value = false
    void ctx.fetchProjectTemplateSummaryPrice()
  }

  ctx.applyProjectRunTemplateIfNeeded = async () => {
    const shape = runTemplateToServerConfigShape(props.projectServerRunTemplate)
    if (!shape) return
    if (props.projectServerRunTemplate?.template_id || props.projectServerRunTemplate?.label) {
      runTemplateMeta.value = {
        template_id: String(props.projectServerRunTemplate.template_id || '').trim(),
        label: String(props.projectServerRunTemplate.label || '').trim(),
      }
    }
    ctx.resolveShapePlatformId(shape, props.projectServerRunTemplate)
    if (shape.platform_id) {
      selectedCloudPlatform.value = String(shape.platform_id)
      ctx.ensureDefaultConfigForPlatform(shape.platform_id, shape.authorization_id)
    }
    projectRunTemplateApplied.value = true
    await ctx.applyServerConfigShape(shape)
  }

  ctx.normalizeDefaultConfigsList = (raw) => {
    if (Array.isArray(raw)) return raw
    if (raw && typeof raw === 'object') {
      return Object.entries(raw).map(([platform_type, value]) => ({
        platform_type,
        ...(value && typeof value === 'object' ? value : {}),
      }))
    }
    return []
  }

  ctx.resolveInstanceTypeFromServerConfig = function(serverConfig) {
    return String(
      serverConfig?.hardware_config?.instance_type
      || serverConfig?.selected_instance
      || props.projectServerRunTemplate?.selected_instance
      || resolveRunTemplateInstanceType(props.projectServerRunTemplate)
      || '',
    ).trim()
  }

  ctx.findCachedInstanceByType = function(instanceType) {
    const target = String(instanceType || '').trim()
    if (!target) return null
    return pageDataCache.value.find((item) => item.instance_type === target)
      || availableInstances.value.find((item) => item.instance_type === target)
      || null
  }

  ctx.ensureInstanceVisibleOnCurrentPage = function(instanceType) {
    const target = String(instanceType || '').trim()
    if (!target || !pageDataCache.value.length) return false
    const idx = pageDataCache.value.findIndex((item) => item.instance_type === target)
    if (idx < 0) return false
    const targetPage = Math.floor(idx / pageSize.value) + 1
    if (currentPage.value !== targetPage) {
      currentPage.value = targetPage
      ctx.updateCurrentPageData()
    }
    return true
  }

  ctx.queueRestoreSelectedInstanceFromTemplate = (instanceType) => {
    const target = String(instanceType || '').trim()
    pendingRestoreInstanceType.value = target
  }

  ctx.tryRestorePendingSelectedInstance = async () => {
    const target = String(pendingRestoreInstanceType.value || '').trim()
    if (!target) return false

    if (selectedInstance.value?.instance_type === target) {
      pendingRestoreInstanceType.value = ''
      return true
    }

    const cached = ctx.findCachedInstanceByType(target)
    if (!cached) return false

    ctx.ensureInstanceVisibleOnCurrentPage(target)
    const instanceToSelect = ctx.findCachedInstanceByType(target)
    if (!instanceToSelect) return false

    await ctx.selectInstance(instanceToSelect)
    if (selectedInstance.value?.instance_type === target) {
      pendingRestoreInstanceType.value = ''
      return true
    }
    return false
  }

  ctx.restoreSelectedInstanceFromTemplate = async (instanceType) => {
    ctx.queueRestoreSelectedInstanceFromTemplate(instanceType)
    return ctx.tryRestorePendingSelectedInstance()
  }

  ctx.finalizeServerConfigShape = async (serverConfig) => {
    if (!selectedCloudPlatform.value) return
    ctx.ensureDefaultConfigForPlatform(selectedCloudPlatform.value)
    if (!regions.value.length) {
      await ctx.loadRegions()
    }
    if (!selectedZone.value && selectedRegion.value && selectedVpc.value) {
      await ctx.loadVswitches()
    }
    if (selectedRegion.value && selectedZone.value) {
      ctx.scheduleFetchAvailableInstances({ immediate: true })
    }
    const instanceType = ctx.resolveInstanceTypeFromServerConfig(serverConfig)
    if (instanceType) {
      ctx.queueRestoreSelectedInstanceFromTemplate(instanceType)
      await ctx.tryRestorePendingSelectedInstance()
    }
  }

  ctx.hasConfiguredProjectRunTemplate = () =>
    projectHasConfiguredRunTemplate({ server_run_template: props.projectServerRunTemplate })
  ctx.buildRunTemplatePayload = () => {
    if (!selectedCloudPlatform.value || !selectedRegion.value) {
      return {}
    }
    return hardwarePanelStateToRunTemplate({
      selectedCloudPlatform: selectedCloudPlatform.value,
      cloudPlatforms: cloudPlatforms.value,
      cloudPlatformDefaultConfigs: cloudPlatformDefaultConfigs.value,
      selectedRegion: selectedRegion.value,
      selectedZone: selectedZone.value,
      selectedVpc: selectedVpc.value,
      selectedSecurityGroup: selectedSecurityGroup.value,
      hardwareConfig: hardwareConfig.value,
      filterOptions: filterOptions.value,
      selectedInstance: selectedInstance.value,
      selectedBandwidth: selectedBandwidth.value,
      selectedBandwidthChargingMode: selectedBandwidthChargingMode.value,
      templateMeta: runTemplateMeta.value,
    })
  }

  ctx.applyRunTemplate = async (template) => {
    ctx.lastApplyRunTemplateError = ''
    if (!template || typeof template !== 'object') {
      ctx.lastApplyRunTemplateError = 'invalid_template'
      return false
    }
    const platformsReady = await ctx.bootstrapCloudPlatformsIfNeeded()
    if (!platformsReady) {
      ctx.lastApplyRunTemplateError = 'no_platforms'
      return false
    }
    const shape = runTemplateToServerConfigShape(template)
    if (!shape) {
      ctx.lastApplyRunTemplateError = 'invalid_shape'
      return false
    }
    runTemplateMeta.value = {
      template_id: String(template?.template_id || '').trim(),
      label: String(template?.label || '').trim(),
    }
    if (!ctx.resolveShapePlatformId(shape, template)) {
      ctx.lastApplyRunTemplateError = 'no_match'
      return false
    }
    projectRunTemplateApplied.value = true
    await ctx.applyServerConfigShape(shape)
    return true
  }

  ctx.getLastApplyRunTemplateError = () => ctx.lastApplyRunTemplateError
}
