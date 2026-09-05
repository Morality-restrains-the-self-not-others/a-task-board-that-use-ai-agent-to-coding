/**
 * VendorPortal cloud-server image registration (OPT-20260817-016).
 */
import { computed, ref } from "vue";
import { api } from "../api";
import { filterCloudServerImagesByKeyword } from "../utils/cloudServerImageFilter.js";
import { normalizeUserdataTemplate } from "../utils/userdataTemplateDisplay.js";


export function initVendorPortalCloudServers(d) {
  const {
    setMsgError, clearMsg, serverImages, refreshServers, loadUserdataTemplateOptions,
    userdataTemplateOptions, regionEnvOpen, regionEnvInitialCsId, regionEnvInitialUdId,
    regionEnvPlatform, regionEnvRegion, clearRegionEnvError,
  } = d

const csOpen = ref(false);
const csEditingId = ref(null);
const csFromRegionEnv = ref(false);
const csForm = ref({
  platform_type: "aliyun",
  region: "",
  selected_image_id: "",
  image_name: "",
  image_id: "",
  os_type: "",
  os_version: "",
  architecture: "",
  image_type: "",
  image_size_gb: "",
  is_active: true,
  default_instance_type_id: "",
  default_instance_type_label: "",
  base_cpu_cores: "",
  base_memory_gib: "",
  userdata_template_id: "",
});
const csRegions = ref([]);
const csImages = ref([]);
const csImageSearchKeyword = ref("");
const csInstanceTypes = ref([]);
const csLoadingRegions = ref(false);
const csLoadingImages = ref(false);
const csLoadingInstanceTypes = ref(false);
const csSaving = ref(false);
const csError = ref("");
const csErrorTraceId = ref("");
function hardwareSummary(s) {
  if (!s) return "—";
  if (s.default_instance_type_label) return s.default_instance_type_label;
  if (s.base_cpu_cores != null || s.base_memory_gib != null) {
    const c = s.base_cpu_cores != null ? `${s.base_cpu_cores}` : "?";
    const m = s.base_memory_gib != null ? `${s.base_memory_gib}` : "?";
    return `${c}vCPU / ${m}GiB`;
  }
  return "—";
}
function optPositiveInt(v) {
  if (v === "" || v === null || v === undefined) return null;
  const n = Number.parseInt(String(v), 10);
  return Number.isFinite(n) && n > 0 ? n : null;
}
const filteredCsImages = computed(() =>
  filterCloudServerImagesByKeyword(csImages.value, csImageSearchKeyword.value),
);
function closeCsModal() {
    csOpen.value = false;
    csEditingId.value = null;
    csFromRegionEnv.value = false;
  }

  async function handleRegionEnvOpenCsCreate() {
    regionEnvOpen.value = false;
    csFromRegionEnv.value = true;
    await openCsCreate();
    csForm.value.platform_type = regionEnvPlatform.value;
    csForm.value.region = regionEnvRegion.value.id;
    await fetchImages();
  }

  async function openCsCreate() {
  csEditingId.value = null;
  csForm.value = {
    platform_type: "aliyun",
    region: "",
    selected_image_id: "",
    image_name: "",
    image_id: "",
    os_type: "",
    os_version: "",
    architecture: "",
    image_type: "",
    image_size_gb: "",
    is_active: true,
    default_instance_type_id: "",
    default_instance_type_label: "",
    base_cpu_cores: "",
    base_memory_gib: "",
    userdata_template_id: "",
  };
  csRegions.value = [];
  csImages.value = [];
  csImageSearchKeyword.value = "";
  csInstanceTypes.value = [];
  csError.value = "";
  csErrorTraceId.value = "";
  await loadUserdataTemplateOptions();
  csOpen.value = true;
  fetchRegions();
}

async function openCsEdit(s) {
  csEditingId.value = s.id;
  csForm.value = {
    platform_type: s.platform_type,
    region: s.region,
    selected_image_id: "__edit__",
    image_name: s.image_name,
    image_id: s.image_id,
    os_type: s.os_type || "",
    os_version: s.os_version || "",
    architecture: s.architecture || "",
    image_type: s.image_type || "",
    image_size_gb: s.image_size_gb != null ? String(s.image_size_gb) : "",
    is_active: s.is_active !== false,
    default_instance_type_id: s.default_instance_type_id || "",
    default_instance_type_label: s.default_instance_type_label || "",
    base_cpu_cores: s.base_cpu_cores != null ? String(s.base_cpu_cores) : "",
    base_memory_gib: s.base_memory_gib != null ? String(s.base_memory_gib) : "",
    userdata_template_id: (() => {
      const ut = normalizeUserdataTemplate(s.userdata_template);
      return ut?.id != null ? String(ut.id) : "";
    })(),
  };
  csRegions.value = [];
  csImages.value = [];
  csImageSearchKeyword.value = "";
  csInstanceTypes.value = [];
  csError.value = "";
  csErrorTraceId.value = "";
  await loadUserdataTemplateOptions();
  csOpen.value = true;
  fetchCsInstanceTypes();
}

async function onPlatformChange() {
  csRegions.value = [];
  csImages.value = [];
  csImageSearchKeyword.value = "";
  csForm.value.region = "";
  csForm.value.selected_image_id = "";
  csError.value = "";
  csErrorTraceId.value = "";
  fetchRegions();
}

async function fetchRegions() {
  csLoadingRegions.value = true;
  csError.value = "";
  csErrorTraceId.value = "";
  try {
    const params = new URLSearchParams({
      platform_type: csForm.value.platform_type,
    });
    const data = await api(`/api/vendor/cloud-server-images/regions/?${params}`);
    if (data.status === 'success') {
      csRegions.value = data.regions || [];
    } else {
      csError.value = data.message || '获取地域列表失败';
      csErrorTraceId.value = data.traceId || "";
    }
  } catch (e) {
    csError.value = e.message;
    csErrorTraceId.value = e.traceId || "";
  } finally {
    csLoadingRegions.value = false;
  }
}

async function onRegionChange() {
  csImages.value = [];
  csImageSearchKeyword.value = "";
  csForm.value.selected_image_id = "";
  csInstanceTypes.value = [];
  csError.value = "";
  csErrorTraceId.value = "";
  if (csForm.value.region) {
    await fetchImages();
  }
}

async function fetchCsInstanceTypes() {
  if (!csForm.value.region) {
    csError.value = "缺少地域，无法加载实例规格";
    csErrorTraceId.value = "";
    return;
  }
  if (!csForm.value.image_id) {
    csError.value = "请先选择镜像后再加载实例规格（需按镜像与可用资源过滤）";
    csErrorTraceId.value = "";
    return;
  }
  csLoadingInstanceTypes.value = true;
  csError.value = "";
  csErrorTraceId.value = "";
  try {
    const params = new URLSearchParams({
      platform_type: csForm.value.platform_type,
      region_id: csForm.value.region,
      image_id: csForm.value.image_id,
    });
    const data = await api(`/api/vendor/cloud-server-images/instance-types/?${params}`);
    if (data.status === "success") {
      csInstanceTypes.value = data.instance_types || [];
      if (!csInstanceTypes.value.length) {
        csError.value = "未返回实例规格（当前可能仅接入部分云平台，或列表为空）";
        csErrorTraceId.value = data.traceId || "";
      }
    } else {
      csInstanceTypes.value = [];
      csError.value = data.message || "获取实例规格失败";
      csErrorTraceId.value = data.traceId || "";
    }
  } catch (e) {
    csInstanceTypes.value = [];
    csError.value = e.message || String(e);
    csErrorTraceId.value = e.traceId || "";
  } finally {
    csLoadingInstanceTypes.value = false;
  }
}

function onCsInstanceTypeChange() {
  const id = csForm.value.default_instance_type_id;
  if (!id) {
    csForm.value.default_instance_type_label = "";
    return;
  }
  const it = csInstanceTypes.value.find((x) => x.instance_type_id === id);
  if (it) {
    const cpu = it.cpu_core_count ?? "";
    const mem = it.memory_size ?? "";
    csForm.value.default_instance_type_label = `${id} (${cpu}vCPU ${mem}GiB)`;
    csForm.value.base_cpu_cores =
      it.cpu_core_count != null ? String(it.cpu_core_count) : csForm.value.base_cpu_cores;
    csForm.value.base_memory_gib =
      it.memory_size != null ? String(Math.round(Number(it.memory_size))) : csForm.value.base_memory_gib;
  }
}

async function fetchImages() {
  if (!csForm.value.region) {
    return;
  }
  csLoadingImages.value = true;
  csError.value = "";
  csErrorTraceId.value = "";
  try {
    const params = new URLSearchParams({
      platform_type: csForm.value.platform_type,
      region_id: csForm.value.region,
    });
    const data = await api(`/api/vendor/cloud-server-images/images/?${params}`);
    if (data.status === 'success') {
      csImages.value = data.images || [];
    } else {
      csError.value = data.message || '获取镜像列表失败';
      csErrorTraceId.value = data.traceId || "";
    }
  } catch (e) {
    csError.value = e.message;
    csErrorTraceId.value = e.traceId || "";
  } finally {
    csLoadingImages.value = false;
  }
}

function onImageChange() {
  const selectedImage = csImages.value.find(img => img.id === csForm.value.selected_image_id);
  if (selectedImage) {
    csForm.value.image_id = selectedImage.id;
    csForm.value.image_name = selectedImage.name;
    csForm.value.os_type = selectedImage.os_type || '';
    csForm.value.os_version = selectedImage.os_version || '';
    csForm.value.architecture = selectedImage.architecture || '';
    csForm.value.image_type = selectedImage.image_type || '';
    csForm.value.image_size_gb = selectedImage.size || '';
  }
}

async function saveCs() {
    if (!csEditingId.value && !csForm.value.selected_image_id) return;
    csSaving.value = true;
    clearMsg();
    try {
      let newServerImageId = null;
      let postCreateServerImage = null;
      if (csEditingId.value) {
        await api(`/api/vendor/cloud-server-images/${csEditingId.value}/`, {
          method: "PATCH",
          body: JSON.stringify({
            image_name: csForm.value.image_name,
            is_active: csForm.value.is_active,
            os_type: csForm.value.os_type,
            os_version: csForm.value.os_version,
            default_instance_type_id: csForm.value.default_instance_type_id || "",
            default_instance_type_label: csForm.value.default_instance_type_label || "",
            base_cpu_cores: optPositiveInt(csForm.value.base_cpu_cores),
            base_memory_gib: optPositiveInt(csForm.value.base_memory_gib),
            userdata_template_id: csForm.value.userdata_template_id || null,
          }),
        });
      } else {
        const body = {
          platform_type: csForm.value.platform_type,
          image_name: csForm.value.image_name,
          image_id: csForm.value.image_id,
          region: csForm.value.region,
          os_type: csForm.value.os_type,
          os_version: csForm.value.os_version,
          architecture: csForm.value.architecture,
          image_type: csForm.value.image_type,
          image_size_gb: csForm.value.image_size_gb ? parseInt(csForm.value.image_size_gb, 10) : null,
          is_active: csForm.value.is_active,
          default_instance_type_id: csForm.value.default_instance_type_id || "",
          default_instance_type_label: csForm.value.default_instance_type_label || "",
          base_cpu_cores: optPositiveInt(csForm.value.base_cpu_cores),
          base_memory_gib: optPositiveInt(csForm.value.base_memory_gib),
          userdata_template_id: csForm.value.userdata_template_id || null,
        };
        const result = await api("/api/vendor/cloud-server-images/", {
          method: "POST",
          body: JSON.stringify(body),
        });
        newServerImageId = result?.id ?? null;
        if (result && typeof result === "object" && result.id != null) {
          postCreateServerImage = result;
        }
      }
      const reopenRegionEnv = csFromRegionEnv.value;
      const selectedTemplateId = csForm.value.userdata_template_id || "";
      closeCsModal();
      await refreshServers();
      if (postCreateServerImage && postCreateServerImage.id != null) {
        const rid = String(postCreateServerImage.id);
        if (!serverImages.value.some((s) => String(s.id) === rid)) {
          serverImages.value = [...serverImages.value, postCreateServerImage];
        }
      }
      if (reopenRegionEnv && newServerImageId) {
        clearRegionEnvError();
        regionEnvInitialCsId.value = String(newServerImageId);
        regionEnvInitialUdId.value = selectedTemplateId;
        regionEnvOpen.value = true;
      }
    } catch (e) {
      setMsgError(e);
    } finally {
      csSaving.value = false;
    }
  }

async function removeServer(s) {
  if (!confirm("删除该条服务器镜像？")) return;
  await api(`/api/vendor/cloud-server-images/${s.id}/`, { method: "DELETE" });
  await refreshServers();
}

  Object.assign(d, {
    hardwareSummary,
    optPositiveInt,
    closeCsModal,
    handleRegionEnvOpenCsCreate,
    openCsCreate,
    openCsEdit,
    onPlatformChange,
    fetchRegions,
    onRegionChange,
    fetchCsInstanceTypes,
    onCsInstanceTypeChange,
    fetchImages,
    onImageChange,
    saveCs,
    removeServer,
    csOpen,
    csEditingId,
    csFromRegionEnv,
    csForm,
    csRegions,
    csImages,
    csImageSearchKeyword,
    csInstanceTypes,
    csLoadingRegions,
    csLoadingImages,
    csLoadingInstanceTypes,
    csSaving,
    csError,
    csErrorTraceId,
    filteredCsImages,
  })
}
