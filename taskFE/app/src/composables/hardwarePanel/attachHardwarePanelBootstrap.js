import { watch } from 'vue'

export function attachHardwarePanelBootstrap(ctx) {
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

  watch(selectedImageId, async () => {
    ctx.syncImageArchitectureFilterFromSelection()
    if (hardwareBootstrapDone.value && selectedCloudPlatform.value) {
      await ctx.loadRegions()
      // OPT-20260821-005：换镜像强制刷新可用实例（ctx.loadRegions 已按新镜像重新拉地域，
      // 这里确保实例列表以新镜像架构立即重取，避免残留旧镜像的实例选项）
      if (regions.value.length > 0 && (!props.runTemplateMode || !projectRunTemplateApplied.value)) {
        ctx.scheduleFetchAvailableInstances({ immediate: true })
      }
    }
  })

  watch(
    () => [installedImages.value, imageArchitectureFromSelection.value],
    () => {
      ctx.syncImageArchitectureFilterFromSelection()
    },
    { deep: true },
  )

  ctx.bootstrapCloudPlatformsIfNeeded = async function() {
    if (!ctx.hasCloudContext()) return false
    if (cloudPlatforms.value.length > 0) return true
    if (isLoadingPlatforms.value || ctx.loadCloudPlatformsPromise) {
      await ctx.loadCloudPlatforms({ skipHandleChange: props.runTemplateMode })
      return cloudPlatforms.value.length > 0
    }
    await ctx.loadCloudPlatforms({ skipHandleChange: props.runTemplateMode })
    return cloudPlatforms.value.length > 0
  }

  ctx.initHardwareFromParent = async function() {
    hardwareBootstrapDone.value = false
    if (props.runTemplateMode) {
      projectRunTemplateApplied.value = false
    }
    const hasTemplate = props.runTemplateMode && ctx.hasConfiguredProjectRunTemplate()
    await ctx.loadCloudPlatforms({ skipHandleChange: hasTemplate })
    if (props.runTemplateMode) {
      await ctx.applyProjectRunTemplateIfNeeded()
      if (!regions.value.length && selectedCloudPlatform.value) {
        await ctx.loadRegions()
      }
      if (!projectRunTemplateApplied.value && selectedCloudPlatform.value) {
        await ctx.handleCloudPlatformChange()
      }
    } else if (props.task) {
      await ctx.fetchPreviousServerConfig()
    }
    ctx.syncImageArchitectureFilterFromSelection()
    hardwareBootstrapDone.value = true
    if (showProjectTemplateSummaryOnly.value) {
      void ctx.fetchProjectTemplateSummaryPrice()
    }
  }

  ctx.bootstrapManualConfig = async () => {
    projectRunTemplateApplied.value = false
    runTemplateMeta.value = { template_id: '', label: '' }
    if (selectedCloudPlatform.value) {
      await ctx.handleCloudPlatformChange()
    } else if (cloudPlatforms.value.length > 0) {
      selectedCloudPlatform.value = String(cloudPlatforms.value[0].id)
      await ctx.handleCloudPlatformChange()
    }
  }

  watch(
    () => [ctx.resolveTenantId(), ctx.resolveWorkspaceId()],
    ([tenantId, workspaceId], previous) => {
      const [prevTenantId, prevWorkspaceId] = previous ?? []
      if (!tenantId || !workspaceId) return
      if (tenantId === prevTenantId && workspaceId === prevWorkspaceId && cloudPlatforms.value.length > 0) {
        return
      }
      void ctx.bootstrapCloudPlatformsIfNeeded()
    },
    { immediate: true },
  )

  watch(
    () => props.projectServerRunTemplate,
    () => {
      if (!hardwareBootstrapDone.value) return
      if (!temporaryConfigExpanded.value && ctx.hasConfiguredProjectRunTemplate()) {
        void ctx.fetchProjectTemplateSummaryPrice()
        return
      }
      projectRunTemplateApplied.value = false
      void ctx.applyProjectRunTemplateIfNeeded()
    },
    { deep: true },
  )

  watch(
    showProjectTemplateSummaryOnly,
    (visible) => {
      if (visible) {
        void ctx.fetchProjectTemplateSummaryPrice()
      }
    },
  )

  watch(
    () => ctx.resolveTaskId(),
    () => {
      temporaryConfigExpanded.value = false
      projectRunTemplateApplied.value = false
      if (props.task && hardwareBootstrapDone.value) {
        void ctx.fetchPreviousServerConfig()
      }
    },
  )
}
