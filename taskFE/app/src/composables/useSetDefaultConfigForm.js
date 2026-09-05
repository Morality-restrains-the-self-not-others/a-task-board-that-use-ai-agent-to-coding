import { ref, onMounted, watch, nextTick } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import modalService from '../utils/modalService.js'
import { coalesceDataDiskCategory } from '../utils/availableInstancesQueryParams.js'

const getRegionDisplayName = (regionId, fallbackName) => fallbackName || regionId

const getTenantId = () => window.location.pathname.match(/\/tenant\/(\d+)\//)?.[1]

const createEmptyFormData = (initialData = {}) => ({
  authorization_id: initialData.authorization_id || '',
  platform_type: initialData.platform_type || '',
  region: '',
  zone_id: '',
  vpc_id: '',
  vswitch_id: '',
  security_group_id: '',
  payment_type: '',
  bandwidth_charging_mode: '',
  bandwidth: 1,
  // 硬件配置
  cpu_cores: '',
  memory_gb: '',
  instance_type: '',
  system_disk_category: 'cloud_essd',
  data_disk_category: 'cloud_essd',
  // 实例筛选（OPT-20260812-020）：IoOptimized 默认 optimized（与实例区勾选一致），竞价策略默认不选
  io_optimized: 'optimized',
  spot_strategy: '',
})

export function useSetDefaultConfigForm(props, emit) {
  const formData = ref(createEmptyFormData())
  const loading = ref(false)
  const loadingRegions = ref(false)
  const loadingVpcs = ref(false)
  const loadingVswitches = ref(false)
  const loadingSecurityGroups = ref(false)

  const regions = ref([])
  const vpcs = ref([])
  const vswitches = ref([])
  const securityGroups = ref([])
  const regionError = ref('')

  // 硬件配置 — 可用实例列表
  const instances = ref([])
  const loadingInstances = ref(false)
  const instanceError = ref('')

  const showCreateVpcModal = ref(false)
  const showCreateVswitchModal = ref(false)
  const showCreateSecurityGroupModal = ref(false)
  const showEditVpcModal = ref(false)
  const showEditVswitchModal = ref(false)
  const showEditSecurityGroupModal = ref(false)

  const editingVpcId = ref('')
  const editingVpcName = ref('')
  const editingVpcCidr = ref('')
  const editingVswitchId = ref('')
  const editingVswitchName = ref('')
  const editingVswitchZone = ref('')
  const editingVswitchCidr = ref('')
  const editingSecurityGroupId = ref('')
  const editingSecurityGroupName = ref('')
  const editingSecurityGroupDescription = ref('')
  const editingSecurityGroupInboundRules = ref([])
  const editingSecurityGroupOutboundRules = ref([])

  const resetFormData = () => {
    formData.value = createEmptyFormData(props.initialData)
  }

  const loadSavedConfig = async () => {
    const authorizationId = props.initialData.authorization_id || formData.value.authorization_id
    if (!authorizationId) {
      console.log('授权ID为空，跳过加载保存的配置')
      return
    }

    try {
      const tenantId = getTenantId()
      const response = await apiFetch(
        `/api/cloud/server-config-default/tenant_id/${tenantId}/${encodeURIComponent(String(authorizationId))}/`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      if (response.ok) {
        const data = await response.json()
        if (data.status === 'success' && data.data) {
          formData.value = {
            ...formData.value,
            region: data.data.region || '',
            zone_id: data.data.zone_id || '',
            vpc_id: data.data.vpc_id || '',
            vswitch_id: data.data.vswitch_id || '',
            security_group_id: data.data.security_group_id || '',
            payment_type: data.data.payment_type || '',
            bandwidth_charging_mode: data.data.bandwidth_charging_mode || '',
            bandwidth: data.data.bandwidth || 1,
            // 硬件配置
            cpu_cores: data.data.cpu_cores || '',
            memory_gb: data.data.memory_gb || '',
            instance_type: data.data.instance_type || '',
            system_disk_category: data.data.system_disk_category || 'cloud_essd',
            // 空字符串表示「不要数据盘」，须用 coalesce 保留，避免被 || 回落成默认盘型
            data_disk_category: coalesceDataDiskCategory(data.data.data_disk_category),
            // IoOptimized/竞价策略：已有保存值才覆盖，保留默认 optimized/不选
            io_optimized: data.data.io_optimized || 'optimized',
            spot_strategy: data.data.spot_strategy || '',
          }
          console.log('加载保存的配置成功:', formData.value)

          if (formData.value.region) {
            await loadVpcs()
          }
        }
      }
    } catch (error) {
      console.error('加载保存的配置失败:', error)
    }
  }

  const fetchImageRegions = async (imageId, platformType) => {
    try {
      const tenantId = getTenantId()
      const url = platformType
        ? `/api/cloud/installed-images/${imageId}/regions/tenant_id/${tenantId}/?platform_type=${platformType}`
        : `/api/cloud/installed-images/${imageId}/regions/tenant_id/${tenantId}/`
      console.log('请求镜像地域列表:', url)

      const response = await apiFetch(url, {
        credentials: 'include',
        headers: { Accept: 'application/json' }
      })

      if (response.ok) {
        const data = await response.json()
        console.log('镜像地域列表响应:', data)
        return Array.isArray(data)
          ? data.map((region) => ({
              region_id: region.region_id || region.id,
              region_name: region.region_name || region.name
            }))
          : []
      }

      console.error('获取镜像支持地域列表失败')
      return []
    } catch (error) {
      console.error('获取镜像支持地域列表出错:', error)
      return []
    }
  }

  const clearNetworkSelections = () => {
    vpcs.value = []
    vswitches.value = []
    securityGroups.value = []
    formData.value.vpc_id = ''
    formData.value.vswitch_id = ''
    formData.value.security_group_id = ''
  }

  const loadRegions = async () => {
    if (loadingRegions.value) {
      console.log('正在加载地域列表，跳过重复调用')
      return
    }

    console.log('开始加载地域列表...')
    console.log('formData:', formData.value)
    console.log('initialData:', props.initialData)

    const authorizationId = props.initialData.authorization_id || formData.value.authorization_id
    const platformType = props.initialData.platform_type || formData.value.platform_type
    const imageId = props.initialData.image_id || formData.value.image_id

    console.log('使用的授权ID:', authorizationId)
    console.log('使用的平台类型:', platformType)
    console.log('使用的镜像ID:', imageId)

    if (!platformType) {
      console.log('平台类型为空，跳过加载地域列表')
      regionError.value = '平台类型为空，请先选择云平台授权'
      regions.value = []
      clearNetworkSelections()
      formData.value.region = ''
      return
    }

    loadingRegions.value = true
    regionError.value = ''
    try {
      const tenantId = getTenantId()
      console.log('租户ID:', tenantId)

      const apiUrl = `/api/cloud/cloud-platform/${authorizationId}/regions/tenant_id/${tenantId}/?platform_type=${platformType}`
      console.log('API请求URL:', apiUrl)

      const response = await apiFetch(apiUrl, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      console.log('API响应状态:', response.status)

      if (response.ok) {
        const data = await response.json()
        console.log('API响应数据:', data)

        const mapRegionItem = (region) => ({
          id: region.region_id || region.id || region.value || region.code,
          name: region.region_name || region.name || region.label || region.title
        })

        if (!Array.isArray(data) && data.status === 'error') {
          console.log('API返回错误:', data.message)
          regionError.value = data.message || '获取地域列表失败'
          regions.value = []
          clearNetworkSelections()
          formData.value.region = ''
          return
        }

        let platformRegions = []
        if (Array.isArray(data)) {
          platformRegions = data.map(mapRegionItem).filter((region) => region.id && region.name)
        } else if (data.status === 'success') {
          if (data.regions && Array.isArray(data.regions)) {
            platformRegions = data.regions.map(mapRegionItem).filter((region) => region.id && region.name)
          } else if (data.data && Array.isArray(data.data)) {
            platformRegions = data.data.map(mapRegionItem).filter((region) => region.id && region.name)
          } else {
            console.log('API返回的数据格式不正确:', data)
            platformRegions = []
          }
        } else if (data.data && Array.isArray(data.data)) {
          platformRegions = data.data.map(mapRegionItem).filter((region) => region.id && region.name)
        } else {
          console.log('API返回的数据格式不正确:', data)
          platformRegions = []
        }

        let filteredRegions = platformRegions
        if (imageId) {
          console.log('获取镜像支持的地域列表...')
          const imageRegions = await fetchImageRegions(imageId, platformType)
          console.log('镜像支持的地域列表:', imageRegions)

          if (imageRegions.length > 0) {
            const imageRegionMap = new Map(imageRegions.map((r) => [r.region_id, r.region_name]))
            filteredRegions = platformRegions
              .filter((region) => imageRegionMap.has(region.id))
              .map((region) => ({
                id: region.id,
                name: imageRegionMap.get(region.id) || region.name
              }))
            console.log('过滤后的地域列表:', filteredRegions)
          }
        }

        regions.value = filteredRegions.map((region) => ({
          id: region.id,
          name: getRegionDisplayName(region.id, region.name)
        }))

        console.log('最终地域列表:', regions.value)
      } else {
        console.log('API请求失败，状态码:', response.status)
        regionError.value = '获取地域列表失败 (HTTP ' + response.status + ')'
        regions.value = []
      }
    } catch (error) {
      console.error('加载地域列表失败:', error)
      regions.value = []
    } finally {
      loadingRegions.value = false
      console.log('地域列表加载完成')
    }
  }

  const loadVpcs = async () => {
    if (!formData.value.platform_type || !formData.value.region || !formData.value.authorization_id) {
      clearNetworkSelections()
      return
    }

    loadingVpcs.value = true
    try {
      const tenantId = getTenantId()
      const vpcsResponse = await apiFetch(
        `/api/cloud/server-images/vpcs/tenant_id/${tenantId}/?region_id=${formData.value.region}&authorization_id=${formData.value.authorization_id}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      if (vpcsResponse.ok) {
        const vpcsData = await vpcsResponse.json()
        // Guard: accept bare array or {items:[]} wrapper; defend against misrouted object responses
        const vpcItems = Array.isArray(vpcsData) ? vpcsData : (vpcsData && Array.isArray(vpcsData.items) ? vpcsData.items : null)
        if (vpcItems) {
          vpcs.value = vpcItems
            .map((vpc) => ({
              id: vpc.vpc_id || vpc.id || vpc.value,
              name: vpc.vpc_name || vpc.name || vpc.label || vpc.id,
              cidr_block: vpc.cidr_block || vpc.cidr || ''
            }))
            .filter((vpc) => vpc.id)
          if (vpcs.value.length > 0) {
            const preferredVpcId = formData.value.vpc_id
            const keepVpc =
              preferredVpcId && vpcs.value.some((v) => String(v.id) === String(preferredVpcId))
            if (!keepVpc) {
              formData.value.vpc_id = vpcs.value[0].id
            }
            await loadVswitches()
            await loadSecurityGroups()
            await loadInstances()
          }
        } else {
          vpcs.value = []
        }
      } else {
        vpcs.value = []
      }
    } catch (error) {
      console.error('加载VPC列表失败:', error)
      vpcs.value = []
    } finally {
      loadingVpcs.value = false
    }
  }

  const loadVswitches = async () => {
    if (
      !formData.value.platform_type ||
      !formData.value.region ||
      !formData.value.vpc_id ||
      !formData.value.authorization_id
    ) {
      vswitches.value = []
      formData.value.vswitch_id = ''
      return
    }

    loadingVswitches.value = true
    try {
      const tenantId = getTenantId()
      const vswitchesResponse = await apiFetch(
        `/api/cloud/server-images/vswitches/tenant_id/${tenantId}/?region_id=${formData.value.region}&vpc_id=${formData.value.vpc_id}&authorization_id=${formData.value.authorization_id}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      if (vswitchesResponse.ok) {
        const vswitchesData = await vswitchesResponse.json()
        const vswItems = Array.isArray(vswitchesData) ? vswitchesData : (vswitchesData && Array.isArray(vswitchesData.items) ? vswitchesData.items : null)
        if (vswItems) {
          vswitches.value = vswItems
          if (vswitches.value.length > 0) {
            const preferredVswitchId = formData.value.vswitch_id
            const keepVswitch =
              preferredVswitchId &&
              vswitches.value.some((v) => String(v.id) === String(preferredVswitchId))
            if (!keepVswitch) {
              formData.value.vswitch_id = vswitches.value[0].id
            }
          }
        } else {
          vswitches.value = []
        }
      } else {
        vswitches.value = []
      }
    } catch (error) {
      console.error('加载交换机列表失败:', error)
      vswitches.value = []
    } finally {
      loadingVswitches.value = false
    }
  }

  const loadSecurityGroups = async () => {
    if (
      !formData.value.platform_type ||
      !formData.value.region ||
      !formData.value.vpc_id ||
      !formData.value.authorization_id
    ) {
      securityGroups.value = []
      formData.value.security_group_id = ''
      return
    }

    loadingSecurityGroups.value = true
    try {
      const tenantId = getTenantId()
      const securityGroupsResponse = await apiFetch(
        `/api/cloud/server-images/security-groups/tenant_id/${tenantId}/?region_id=${formData.value.region}&vpc_id=${formData.value.vpc_id}&authorization_id=${formData.value.authorization_id}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      if (securityGroupsResponse.ok) {
        const securityGroupsData = await securityGroupsResponse.json()
        const sgItems = Array.isArray(securityGroupsData) ? securityGroupsData : (securityGroupsData && Array.isArray(securityGroupsData.items) ? securityGroupsData.items : null)
        if (sgItems) {
          securityGroups.value = sgItems
          if (securityGroups.value.length > 0) {
            const preferredSgId = formData.value.security_group_id
            const keepSg =
              preferredSgId &&
              securityGroups.value.some((sg) => String(sg.id) === String(preferredSgId))
            if (!keepSg) {
              formData.value.security_group_id = securityGroups.value[0].id
            }
          }
        } else {
          securityGroups.value = []
        }
      } else {
        securityGroups.value = []
      }
    } catch (error) {
      console.error('加载安全组列表失败:', error)
      securityGroups.value = []
    } finally {
      loadingSecurityGroups.value = false
    }
  }

  const loadInstances = async () => {
    if (
      !formData.value.platform_type ||
      !formData.value.region ||
      !formData.value.authorization_id
    ) {
      instances.value = []
      return
    }

    loadingInstances.value = true
    instanceError.value = ''
    try {
      const tenantId = getTenantId()
      let zoneId = ''
      if (formData.value.vswitch_id) {
        // 尝试从 vswitches 列表中获取 zone_id
        const matchedVsw = vswitches.value.find(
          (v) => String(v.id) === String(formData.value.vswitch_id)
        )
        if (matchedVsw && matchedVsw.zone_id) {
          zoneId = matchedVsw.zone_id
        }
      }

      let url = `/api/cloud/cloud-platform/${encodeURIComponent(String(formData.value.authorization_id))}/available-instances/tenant_id/${tenantId}/?platform_type=${formData.value.platform_type}&region_id=${formData.value.region}`
      if (zoneId) {
        url += `&zone_id=${zoneId}`
      }
      // 添加过滤参数
      if (formData.value.cpu_cores) {
        url += `&Cores=${formData.value.cpu_cores}`
      }
      if (formData.value.memory_gb) {
        url += `&Memory=${formData.value.memory_gb}`
      }
      if (formData.value.system_disk_category) {
        url += `&SystemDiskCategory=${formData.value.system_disk_category}`
      }
      if (formData.value.data_disk_category) {
        url += `&DataDiskCategory=${formData.value.data_disk_category}`
      }
      url += '&IoOptimized=optimized&DestinationResource=InstanceType&ResourceType=instance'

      const response = await apiFetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        const data = await response.json()

        const mapInstance = (inst) => ({
          instance_type: inst.instance_type || inst.InstanceTypeId || inst.instance_type_id || '',
          cpu_cores: inst.cpu_cores ?? inst.CpuCoreCount ?? 0,
          memory_gb: inst.memory_gb ?? inst.MemorySize ?? 0,
          instance_type_family: inst.instance_type_family || inst.InstanceTypeFamily || '',
          status: inst.status || inst.Status || 'available',
          gpu_cores: inst.gpu_cores ?? inst.GPUAmount ?? inst.GpuCoreCount ?? 0,
          gpu_type: inst.gpu_type || inst.GpuSpec || '',
          gpu_memory: inst.gpu_memory ?? inst.GpuMemory ?? 0,
          architecture: inst.architecture || inst.Architecture || '',
          cpu_type: inst.cpu_type || inst.CpuType || '',
          max_bandwidth_out: inst.max_bandwidth_out ?? inst.MaxBandwidthOut ?? 0,
          network_performance: inst.network_performance || inst.NetworkPerformance || '',
          local_disk_size: inst.local_disk_size ?? inst.LocalDiskSize ?? 0,
          local_disk_type: inst.local_disk_type || inst.LocalDiskCategory || '',
          storage_type: inst.storage_type || inst.StorageType || '',
          max_internet_bandwidth_out: inst.max_internet_bandwidth_out ?? inst.MaxInternetBandwidthOut ?? 0,
        })

        if (Array.isArray(data)) {
          instances.value = data.map(mapInstance).filter((i) => i.instance_type)
        } else if (data.InstanceTypes && data.InstanceTypes.InstanceType) {
          instances.value = data.InstanceTypes.InstanceType.map(mapInstance).filter((i) => i.instance_type)
        } else if (data.instance_types) {
          instances.value = data.instance_types.map((t) =>
            typeof t === 'string' ? { instance_type: t, cpu_cores: 0, memory_gb: 0 } : mapInstance(t)
          ).filter((i) => i.instance_type)
        } else if (Array.isArray(data.data)) {
          instances.value = data.data.map(mapInstance).filter((i) => i.instance_type)
        } else {
          instances.value = []
        }

        // 如果先前有 instance_type 但当前列表中不存在，则清空选中
        if (formData.value.instance_type && instances.value.length > 0) {
          const stillExists = instances.value.some(
            (i) => String(i.instance_type) === String(formData.value.instance_type)
          )
          if (!stillExists) {
            formData.value.instance_type = ''
          }
        }
      } else {
        instanceError.value = `获取实例列表失败 (HTTP ${response.status})`
        instances.value = []
      }
    } catch (error) {
      console.error('加载可用实例列表失败:', error)
      instanceError.value = '获取实例列表失败'
      instances.value = []
    } finally {
      loadingInstances.value = false
    }
  }

  const handleRegionChange = async () => {
    if (!formData.value.platform_type || !formData.value.region) {
      clearNetworkSelections()
      return
    }
    await loadVpcs()
  }

  const handleVpcChange = async () => {
    if (!formData.value.platform_type || !formData.value.region || !formData.value.vpc_id) {
      vswitches.value = []
      securityGroups.value = []
      instances.value = []
      formData.value.vswitch_id = ''
      formData.value.security_group_id = ''
      return
    }
    await loadVswitches()
    await loadSecurityGroups()
    await loadInstances()
  }

  const openCreateVpcModal = () => {
    if (!formData.value.region) {
      modalService.alert('请先选择地域')
      return
    }
    editingVpcId.value = ''
    editingVpcName.value = ''
    editingVpcCidr.value = ''
    showCreateVpcModal.value = true
  }

  const openCreateVswitchModal = () => {
    if (!formData.value.vpc_id) {
      modalService.alert('请先选择VPC')
      return
    }
    editingVswitchId.value = ''
    editingVswitchName.value = ''
    editingVswitchZone.value = ''
    editingVswitchCidr.value = ''
    showCreateVswitchModal.value = true
  }

  const openCreateSecurityGroupModal = () => {
    if (!formData.value.vpc_id) {
      modalService.alert('请先选择VPC')
      return
    }
    editingSecurityGroupId.value = ''
    editingSecurityGroupName.value = ''
    editingSecurityGroupDescription.value = ''
    showCreateSecurityGroupModal.value = true
  }

  const openEditVpcModal = () => {
    if (!formData.value.vpc_id) {
      modalService.alert('请先选择VPC')
      return
    }

    const selectedVpc = vpcs.value.find((vpc) => vpc.id === formData.value.vpc_id)
    if (selectedVpc) {
      editingVpcId.value = selectedVpc.id
      editingVpcName.value = selectedVpc.name
      editingVpcCidr.value = selectedVpc.cidr_block || ''
      showEditVpcModal.value = true
    }
  }

  const openEditVswitchModal = () => {
    if (!formData.value.vswitch_id) {
      modalService.alert('请先选择交换机')
      return
    }

    const selectedVswitch = vswitches.value.find((vswitch) => vswitch.id === formData.value.vswitch_id)
    if (selectedVswitch) {
      editingVswitchId.value = selectedVswitch.id
      editingVswitchName.value = selectedVswitch.name
      editingVswitchZone.value = selectedVswitch.zone_id || selectedVswitch.zone
      editingVswitchCidr.value = selectedVswitch.cidr_block || selectedVswitch.cidr
      showEditVswitchModal.value = true
    }
  }

  const openEditSecurityGroupModal = () => {
    if (!formData.value.security_group_id) {
      modalService.alert('请先选择安全组')
      return
    }

    const selectedSecurityGroup = securityGroups.value.find(
      (sg) => sg.id === formData.value.security_group_id
    )
    if (selectedSecurityGroup) {
      editingSecurityGroupId.value = selectedSecurityGroup.id
      editingSecurityGroupName.value = selectedSecurityGroup.name
      editingSecurityGroupDescription.value = selectedSecurityGroup.description || ''
      editingSecurityGroupInboundRules.value = selectedSecurityGroup.inbound_rules || []
      editingSecurityGroupOutboundRules.value = selectedSecurityGroup.outbound_rules || []
      showEditSecurityGroupModal.value = true
    }
  }

  const handleVpcCreated = async () => {
    await loadVpcs()
  }

  const handleVswitchCreated = async () => {
    await loadVswitches()
  }

  const handleSecurityGroupCreated = async () => {
    await loadSecurityGroups()
  }

  const closeVpcModal = () => {
    showCreateVpcModal.value = false
    showEditVpcModal.value = false
  }

  const closeVswitchModal = () => {
    showCreateVswitchModal.value = false
    showEditVswitchModal.value = false
  }

  const closeSecurityGroupModal = () => {
    showCreateSecurityGroupModal.value = false
    showEditSecurityGroupModal.value = false
  }

  const handleClose = () => {
    emit('close')
  }

  // 硬件配置筛选变更时重新加载实例列表
  const handleInstanceFilterChange = async () => {
    if (!formData.value.region || !formData.value.authorization_id) {
      return
    }
    await loadInstances()
  }

  const resolveZoneIdForSubmit = () => {
    const existing = String(formData.value.zone_id || '').trim()
    if (existing) return existing
    const vswId = String(formData.value.vswitch_id || '').trim()
    if (!vswId) return ''
    const matched = vswitches.value.find((v) => String(v.id || v.vswitch_id || '') === vswId)
    return String(matched?.zone_id || matched?.zone || '').trim()
  }

  const handleSubmit = async () => {
    loading.value = true
    try {
      const tenantId = getTenantId()
      const zoneId = resolveZoneIdForSubmit()
      const response = await apiFetch(`/api/cloud/server-config-default/tenant_id/${tenantId}/`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify({
          ...formData.value,
          zone_id: zoneId,
          authorization_id: String(formData.value.authorization_id)
        })
      })

      if (response.ok) {
        const data = await response.json()
        if (data.status === 'success') {
          formData.value.zone_id = zoneId
          modalService.alert('默认配置设置成功，包含硬件配置')
          emit('config-updated')
          emit('close')
        } else {
          throw new Error(data.message || '默认配置设置失败')
        }
      } else {
        throw new Error('默认配置设置失败')
      }
    } catch (error) {
      console.error('设置默认配置失败:', error)
      modalService.alert('设置默认配置失败: ' + error.message, '错误', { traceId: error.traceId })
    } finally {
      loading.value = false
    }
  }

  const initializeForm = async () => {
    resetFormData()
    await nextTick()
    await loadRegions()
    await loadSavedConfig()
  }

  onMounted(async () => {
    if (props.visible) {
      await initializeForm()
    }
  })

  watch(
    () => props.visible,
    async (newValue) => {
      if (newValue) {
        await initializeForm()
      }
    }
  )

  watch(
    () => props.initialData,
    async (newValue) => {
      if (props.visible && newValue) {
        formData.value = {
          ...formData.value,
          authorization_id: newValue.authorization_id || '',
          platform_type: newValue.platform_type || ''
        }
        await nextTick()
        await loadRegions()
        await loadSavedConfig()
      }
    },
    { deep: true }
  )

  return {
    formData,
    loading,
    loadingRegions,
    loadingVpcs,
    loadingVswitches,
    loadingSecurityGroups,
    regions,
    regionError,
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
  }
}
