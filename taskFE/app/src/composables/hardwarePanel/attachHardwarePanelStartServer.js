import { resolveEnvParamsSourceRequiredHint } from '../../utils/envParamsSourceSelection.js'
import { buildStartVmErrorStatusUpdate } from '../../utils/startVmHttpResult.js'
import {
  buildManualHardwareStartVmBody,
  buildTemplateStartVmRequestWithClientIp,
  postHardwarePanelStartVm,
} from '../../utils/hardwarePanelStartVm.js'

export function attachHardwarePanelStartServer(ctx) {
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

  ctx.startServer = async () => {
    if (props.requireEnvParamsSource) {
      const envHint = resolveEnvParamsSourceRequiredHint({
        featureParamsSource: props.featureParamsSource,
        selectedPersonalConfigId: props.selectedPersonalConfigId,
      })
      if (envHint) {
        console.warn(envHint)
        return
      }
    }
    if (!selectedImageId.value) {
      alert('请先选择镜像')
      return
    }
    if (autoReleaseEnabled.value) {
      const m = Math.floor(Number(autoReleaseMinutes.value))
      if (!Number.isFinite(m) || m < 30 || m > 1440) {
        alert('自动释放时间请填写 30～1440 之间的整数分钟（阿里云最短为当前时间 30 分钟后）')
        return
      }
    }

    const tenantId = route.params.tenant
    const workspaceId = props.workspaceId || route.params.workspaceId
    const onAccepted = (result) => emit('start-request-accepted', result)
    const failNetwork = () => {
      if (props.updateServerStatus) {
        props.updateServerStatus(
          buildStartVmErrorStatusUpdate(null, '启动服务器失败，请检查网络连接或联系管理员'),
        )
      }
    }

    if (useProjectTemplateForStart.value) {
      try {
        const built = await buildTemplateStartVmRequestWithClientIp({
          taskId: ctx.resolveTaskId(),
          containerImageId: selectedImageId.value,
          runTemplate: props.projectServerRunTemplate,
          autoReleaseEnabled: autoReleaseEnabled.value,
          autoReleaseMinutes: autoReleaseMinutes.value,
        })
        if (!built) {
          alert('项目运行硬件模版配置不完整，请检查或使用临时配置')
          return
        }
        await postHardwarePanelStartVm({
          tenantId,
          workspaceId,
          apiPath: built.apiPath,
          body: built.body,
          updateServerStatus: props.updateServerStatus,
          onAccepted,
        })
      } catch (error) {
        console.error('启动服务器出错:', error)
        failNetwork()
      }
      return
    }

    if (!selectedCloudPlatform.value) { alert('请先选择云平台'); return }
    if (!selectedRegion.value) { alert('请选择地域'); return }
    if (!selectedVpc.value) { alert('请选择VPC'); return }
    if (!selectedZone.value) { alert('请选择可用区'); return }

    try {
      let zoneId = selectedZone.value
      let vswitchId = null
      let autoCreateVswitch = false
      if (zoneId.includes(':')) {
        const parts = zoneId.split(':')
        zoneId = parts[0]
        vswitchId = parts[1]
      } else {
        const previousZoneId = previousServerConfig.value?.zone_id
        const previousVswitchId = previousServerConfig.value?.vswitch_id
        if (previousZoneId && previousVswitchId && previousZoneId === zoneId) {
          vswitchId = previousVswitchId
        } else {
          autoCreateVswitch = true
        }
      }
      let securityGroupId = selectedSecurityGroup.value
      let autoCreateSecurityGroup = false
      if (securityGroupId === 'auto_create_security_group') {
        autoCreateSecurityGroup = true
        securityGroupId = null
      }
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
      const autoCreateVpc = selectedVpc.value === 'auto_create_vpc'
      const apiPath = (autoCreateVpc || autoCreateVswitch || autoCreateSecurityGroup)
        ? 'start-vm-auto'
        : 'start-vm'
      const body = await buildManualHardwareStartVmBody({
        taskId: ctx.resolveTaskId(),
        containerImageId: selectedImageId.value,
        hardwareConfig: hardwareConfig.value,
        filterOptions: filterOptions.value,
        selectedInstance: selectedInstance.value,
        selectedBandwidth: selectedBandwidth.value,
        defaultBandwidth: ctx.DEFAULT_BANDWIDTH,
        regionId: selectedRegion.value,
        zoneId,
        vpcId: autoCreateVpc ? null : selectedVpc.value,
        vswitchId,
        autoCreateVswitch,
        cloudPlatformId: selectedCloudPlatform.value,
        authorizationId: defaultConfig ? defaultConfig.authorization_id : null,
        securityGroupId,
        autoCreateSecurityGroup,
        bandwidth: selectedInstance.value
          ? (selectedBandwidth.value[selectedInstance.instance_type] || ctx.DEFAULT_BANDWIDTH)
          : ctx.DEFAULT_BANDWIDTH,
        bandwidthChargingMode: selectedBandwidthChargingMode.value,
        autoReleaseEnabled: autoReleaseEnabled.value,
        autoReleaseMinutes: autoReleaseMinutes.value,
      })
      await postHardwarePanelStartVm({
        tenantId,
        workspaceId,
        apiPath,
        body,
        updateServerStatus: props.updateServerStatus,
        onAccepted,
      })
    } catch (error) {
      console.error('启动服务器出错:', error)
      failNetwork()
    }
  }
  // 深度比较两个对象是否相等
  ctx.deepEqual = function(obj1, obj2) {
    if (obj1 === obj2) return true;
    if (obj1 == null || obj2 == null) return false;
    if (typeof obj1 !== typeof obj2) return false;
    if (typeof obj1 === 'object') {
      const keys1 = Object.keys(obj1);
      const keys2 = Object.keys(obj2);
      if (keys1.length !== keys2.length) return false;
      for (const key of keys1) {
        if (!keys2.includes(key) || !ctx.deepEqual(obj1[key], obj2[key])) return false;
      }
      return true;
    }
    return false;
  }
}
