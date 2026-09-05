import { apiFetch } from '../../utils/apiUtils.js'

export function attachHardwarePanelCloudLoad(ctx) {
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

  ctx.fetchImageRegions = async (imageId) => {
    if (!imageId) {
      return []
    }
  
    try {
      const tenantId = route.params.tenant
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
      const platformType = defaultConfig?.platform_type || ''
    
      const response = await apiFetch(`/api/cloud/installed-images/${imageId}/regions/tenant_id/${tenantId}/?platform_type=${platformType}`, {
        credentials: 'include',
        headers: {
          'Accept': 'application/json'
        }
      })
    
      if (response.ok) {
        const data = await response.json()
        return Array.isArray(data) ? data.map(region => {
          const regionId = region.region_id || region.id
          const regionName = region.region_name || region.name
          return {
            region_id: regionId,
            region_name: ctx.getRegionDisplayName(regionId, regionName)
          }
        }) : []
      } else {
        console.error('获取镜像支持地域列表失败')
        return []
      }
    } catch (error) {
      console.error('获取镜像支持地域列表出错:', error)
      return []
    }
  }

  ctx.loadRegions = async () => {
    const imageKey = String(selectedImageId.value || '').trim()
    // 同镜像并发仍合并；快速切换镜像时旧请求不得复用，立即为当前镜像发起新请求
    if (ctx.loadRegionsPromise && ctx.loadRegionsPromiseImageKey === imageKey) {
      return ctx.loadRegionsPromise
    }
    const generation = ++ctx.loadRegionsGeneration
    ctx.loadRegionsPromiseImageKey = imageKey
    ctx.loadRegionsPromise = (async () => {
      try {
        isLoadingRegions.value = true
        const tenantId = ctx.resolveTenantId()
        ctx.ensureDefaultConfigForPlatform(selectedCloudPlatform.value)
        const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()

        if (!defaultConfig || !defaultConfig.authorization_id) {
          return
        }

        let fetchedRegions = []

        // 如果已选择镜像，优先从镜像支持的地域列表中获取
        if (imageKey) {
          fetchedRegions = await ctx.fetchImageRegions(imageKey)
          console.log('从镜像获取地域列表:', fetchedRegions.length, '个地域')
        }

        // 如果镜像地域列表为空或未选择镜像，回退到云平台地域列表
        if (fetchedRegions.length === 0) {
          const response = await apiFetch(`/api/cloud/cloud-platform/${defaultConfig.authorization_id}/regions/tenant_id/${tenantId}/?platform_type=${defaultConfig.platform_type}`, {
            credentials: 'include',
            headers: {
              'Accept': 'application/json',
            },
          })

          if (response.ok) {
            const data = await response.json()
            if (data.status !== 'error') {
              fetchedRegions = Array.isArray(data) ? data.map(region => ({
                region_id: region.id,
                region_name: ctx.getRegionDisplayName(region.id, region.name),
              })) : []
            }
          } else {
            console.error('获取云平台地域列表失败')
          }
        }

        // 请求期间镜像已再次切换：丢弃过期结果，避免旧地域/旧实例列表套到新镜像
        if (generation !== ctx.loadRegionsGeneration) {
          return
        }

        regions.value = fetchedRegions

        // 运行模版/历史配置已指定地域时保留，否则用工作区默认或列表首项
        if (regions.value.length > 0) {
          const platDefault = ctx.resolveCloudPlatformDefaultConfig()
          const savedRegion = platDefault?.config?.region
          const regionIds = regions.value.map((r) => r.region_id)
          const preferredRegion = String(selectedRegion.value || '').trim()
          if (!preferredRegion || !regionIds.includes(preferredRegion)) {
            selectedRegion.value = String(
              savedRegion && regionIds.includes(savedRegion)
                ? savedRegion
                : regions.value[0].region_id,
            )
          }
          if (!selectedZone.value) {
            vswitches.value = []
            zones.value = []
            selectedZone.value = ''
          }
          await ctx.loadVpcs()
          if (!props.runTemplateMode || !projectRunTemplateApplied.value) {
            ctx.scheduleFetchAvailableInstances({ immediate: true })
          }
        }
      } catch (error) {
        console.error('获取地域列表出错:', error)
      } finally {
        if (generation === ctx.loadRegionsGeneration) {
          isLoadingRegions.value = false
          ctx.loadRegionsPromise = null
          ctx.loadRegionsPromiseImageKey = ''
        }
      }
    })()

    return ctx.loadRegionsPromise
  }

  ctx.loadVpcs = async () => {
    // 防止重复请求
    if (ctx.isLoadingVpcsFlag) {
      console.log('正在获取VPC列表，跳过重复请求');
      return
    }
  
    // 清空VPC列表的数据
    vpcs.value = []
    selectedVpc.value = ''
  
    if (!selectedRegion.value) {
      return
    }
  
    try {
      ctx.isLoadingVpcsFlag = true
      isLoadingVpcs.value = true
      const tenantId = route.params.tenant
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
    
      if (defaultConfig && defaultConfig.authorization_id) {
        const response = await apiFetch(`/api/cloud/server-images/vpcs/tenant_id/${tenantId}/?region_id=${selectedRegion.value}&authorization_id=${defaultConfig.authorization_id}`, {
          credentials: 'include',
          headers: {
            'Accept': 'application/json'
          }
        })
      
        if (response.ok) {
          const data = await response.json()
          if (data.status !== 'error') {
            const rawVpcs = Array.isArray(data) ? data : (data && Array.isArray(data.items) ? data.items : [])
            vpcs.value = rawVpcs.map(vpc => ({
              id: vpc.vpc_id || vpc.id || vpc.VpcId,
              name: vpc.vpc_name || vpc.name || vpc.VpcName || '未命名VPC'
            }))

            // 无论VPC列表是否为空，都添加"自动创建VPC"选项
            vpcs.value.push({
              id: 'auto_create_vpc',
              name: '自动创建VPC'
            })
          
            // 优先使用工作区已保存的默认 VPC，否则选列表第一项（排除末尾「自动创建」）
            if (vpcs.value.length > 0) {
              const platDefault = ctx.resolveCloudPlatformDefaultConfig()
              const savedVpc = platDefault?.config?.vpc_id
              const vpcIds = vpcs.value.map((v) => v.id)
              if (savedVpc && vpcIds.includes(savedVpc)) {
                selectedVpc.value = String(savedVpc)
              } else {
                const firstReal = vpcs.value.find((v) => v.id !== 'auto_create_vpc')
                selectedVpc.value = String((firstReal || vpcs.value[0]).id)
              }
            }
          
            // 加载交换机列表
            await ctx.loadVswitches()
          }
        } else {
          console.error('获取VPC列表失败')
        }
      }
    } catch (error) {
      console.error('获取VPC列表出错:', error)
    } finally {
      isLoadingVpcs.value = false
      ctx.isLoadingVpcsFlag = false
    }
  }

  ctx.loadVswitches = async () => {
    if (!selectedVpc.value) {
      vswitches.value = []
      zones.value = []
      securityGroups.value = []
      selectedZone.value = ''
      selectedSecurityGroup.value = ''
      return
    }
  
    try {
      const tenantId = route.params.tenant
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
    
      if (defaultConfig && defaultConfig.authorization_id) {
        // 同时加载交换机和安全组
        const [vswitchesResponse, securityGroupsResponse] = await Promise.all([
          // 加载交换机列表
          selectedVpc.value === 'auto_create_vpc' ? Promise.resolve({ ok: true, json: () => Promise.resolve([]) }) :
            apiFetch(`/api/cloud/server-images/vswitches/tenant_id/${tenantId}/?region_id=${selectedRegion.value}&vpc_id=${selectedVpc.value}&authorization_id=${defaultConfig.authorization_id}`, {
              credentials: 'include',
              headers: {
                'Accept': 'application/json'
              }
            }),
          // 加载安全组列表
          apiFetch(`/api/cloud/server-images/security-groups/tenant_id/${tenantId}/?region_id=${selectedRegion.value}&vpc_id=${selectedVpc.value}&authorization_id=${defaultConfig.authorization_id}`, {
            credentials: 'include',
            headers: {
              'Accept': 'application/json'
            }
          })
        ])
      
        // 处理交换机响应
        if (vswitchesResponse.ok) {
          const data = await vswitchesResponse.json()
          if (data.status !== 'error') {
            const rawVswitches = Array.isArray(data) ? data : (data && Array.isArray(data.items) ? data.items : [])
            vswitches.value = rawVswitches.map(vswitch => ({
              vswitch_id: vswitch.vswitch_id || vswitch.id || vswitch.VSwitchId || `vswitch_${Date.now()}_${Math.floor(Math.random() * 1000)}`,
              zone_id: vswitch.zone_id || vswitch.zoneId || vswitch.ZoneId || vswitch.zone_id || `zone_${Date.now()}_${Math.floor(Math.random() * 1000)}`,
              vswitch_name: vswitch.vswitch_name || vswitch.name || vswitch.VSwitchName || '未命名交换机'
            }))
          }
        } else {
          console.error('获取交换机列表失败')
        }
      
        // 处理安全组响应
        if (securityGroupsResponse.ok) {
          const data = await securityGroupsResponse.json()
          if (data.status !== 'error') {
            const rawSgs = Array.isArray(data) ? data : (data && Array.isArray(data.items) ? data.items : [])
            securityGroups.value = rawSgs.map(sg => ({
              id: sg.security_group_id || sg.id || sg.SecurityGroupId,
              name: sg.security_group_name || sg.name || sg.SecurityGroupName || '未命名安全组'
            }))
          
            // 添加"自动创建安全组"选项
            securityGroups.value.push({
              id: 'auto_create_security_group',
              name: '自动创建安全组'
            })
          
            // 优先使用工作区已保存的默认安全组
            if (securityGroups.value.length > 0) {
              const platDefault = ctx.resolveCloudPlatformDefaultConfig()
              const savedSg = platDefault?.config?.security_group_id
              const sgIds = securityGroups.value.map((s) => s.id)
              if (savedSg && sgIds.includes(savedSg)) {
                selectedSecurityGroup.value = String(savedSg)
              } else {
                const firstReal = securityGroups.value.find((s) => s.id !== 'auto_create_security_group')
                selectedSecurityGroup.value = String((firstReal || securityGroups.value[0]).id)
              }
            }
          }
        } else {
          console.error('获取安全组列表失败')
        }
      
        // 加载可用区列表
        await ctx.loadZones()
      }
    } catch (error) {
      console.error('获取网络资源出错:', error)
    }
  }

  ctx.loadZones = async () => {
    // 防止重复请求
    if (ctx.isLoadingZonesFlag) {
      console.log('正在获取可用区列表，跳过重复请求');
      return
    }
  
    if (!selectedRegion.value) {
      zones.value = []
      return
    }
  
    try {
      ctx.isLoadingZonesFlag = true
      isLoadingZones.value = true
      const tenantId = route.params.tenant
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
    
      if (defaultConfig && defaultConfig.authorization_id) {
        const response = await apiFetch(`/api/cloud/cloud-platform/${defaultConfig.authorization_id}/zones/tenant_id/${tenantId}/?region_id=${selectedRegion.value}`, {
          credentials: 'include',
          headers: {
            'Accept': 'application/json'
          }
        })
      
        if (response.ok) {
          const data = await response.json()
          if (data.status !== 'error') {
            // 转换API响应格式并添加交换机信息
            zones.value = Array.isArray(data) ? data.map(zone => {
              // 查找该可用区的交换机
              const zoneVswitches = vswitches.value.filter(vswitch => vswitch.zone_id === (zone.id || zone.zone_id))
            
              // 获取可用区名称，支持不同的属性名
              const zoneName = zone.name || zone.zone_name || zone.display_name || '未知可用区'
            
              // 获取可用区ID，支持不同的属性名
              const zoneId = zone.id || zone.zone_id || zone.ZoneId || `zone_${Date.now()}_${Math.floor(Math.random() * 1000)}`
            
              // 为每个交换机创建一个选项
              const options = zoneVswitches.map(vswitch => {
                const vswitchId = vswitch.vswitch_id || vswitch.id || vswitch.VSwitchId || `vswitch_${Date.now()}_${Math.floor(Math.random() * 1000)}`
                return {
                  zone_id: `${zoneId}:${vswitchId}`,
                  zone_name: zoneName,
                  display_name: `${zoneName}-${vswitch.vswitch_name || vswitch.name || vswitch.VSwitchName || '未命名交换机'}`
                }
              })
            
              // 为每个可用区添加"自动创建交换机"选项
              options.push({
                zone_id: zoneId,
                zone_name: zoneName,
                display_name: `${zoneName}-自动创建交换机`
              })
            
              return options
            }).flat() : []
          
            // 优先使用工作区已保存的默认可用区+交换机，否则选第一项
            if (zones.value.length > 0) {
              const platDefault = ctx.resolveCloudPlatformDefaultConfig()
              const cfg = platDefault?.config
              const savedZoneId = cfg?.zone_id
              const savedVsw = cfg?.vswitch_id
              let picked = ''
              if (savedZoneId && savedVsw) {
                const composite = `${savedZoneId}:${savedVsw}`
                if (zones.value.some((z) => z.zone_id === composite)) {
                  picked = composite
                }
              }
              if (!picked && savedZoneId) {
                const plain = zones.value.find((z) => z.zone_id === savedZoneId)
                if (plain) {
                  picked = plain.zone_id
                }
              }
              selectedZone.value = String(picked || zones.value[0].zone_id)
            }
          }
        } else {
          console.error('获取可用区列表失败')
        }
      }
    } catch (error) {
      console.error('获取可用区列表出错:', error)
    } finally {
      isLoadingZones.value = false
      ctx.isLoadingZonesFlag = false
    }
  }
}
