/**
 * VendorPortal image-group / version / runtime-env logic (OPT-20260817-016).
 */
import { computed, ref } from "vue";
import { createClickGuard } from "../utils/clickGuard.js";
import { createVendorImageGroupFormApi } from "../utils/vendorImageGroupForm.js";
import { api } from "../api";
import { summarizeImageResolveInfo } from "../lib/imageResolveInfo.js";
import {
  enrichImageGroups,
  formatUnavailableBadge,
  getGroupVersions as filterGroupVersions,
  groupActiveVersion,
  isActiveVersion,
  isMarketplaceUnavailable,
  statusDisplay,
} from "../utils/containerImageGroup.js";
import {
  versionHasAction,
  withdrawButtonLabel,
} from "../lib/vendorVersionRowActions.js";
import { buildRegionEnvAssociationPayload } from "../utils/regionEnvAssociationPayload.js";
import { createVendorImageActionConfirm } from "./vendorImageActionConfirm.js";
import {
  normalizeUserdataTemplate,
  userdataTemplateDisplayLabel,
} from "../utils/userdataTemplateDisplay.js";
import { CLOUD_PLATFORMS as cloudPlatforms } from "../utils/platformLabel.js";


export function initVendorPortalImageGroups(d) {
  const {
    setMsgError, setMsgText, clearMsg, credentialHint, msg,
  } = d
  const userdataTemplateOptions = ref([]);

const imageGroups = ref([]);
const containerImages = ref([]);
const serverImages = ref([]);
const expandedGroups = ref({});
const expandedRejectNotes = ref({});

const groupOpen = ref(false);
const groupEditingId = ref(null);
const groupForm = ref({ name: "", description: "", icon_file_key: "", icon_url: "" });
const groupIconFile = ref(null);
const groupIconPreview = ref("");
const groupFormError = ref("");
const groupFormErrorTraceId = ref("");
const groupSaving = ref(false);
const groupSaveGuard = createClickGuard();

const imageVersionModals = ref(null);

const envTargetArchs = ref([]);

const regionEnvOpen = ref(false);
const regionEnvImageId = ref(null);
const regionEnvPlatform = ref(null);
const regionEnvRegion = ref(null);
const regionEnvInitialCsId = ref("");
const regionEnvInitialUdId = ref("");
const savingRegionEnv = ref(false);
const regionEnvError = ref("");
const regionEnvErrorTraceId = ref("");

function clearRegionEnvError() {
  regionEnvError.value = "";
  regionEnvErrorTraceId.value = "";
}

function setRegionEnvError(e) {
  regionEnvError.value = e?.message || String(e ?? "");
  regionEnvErrorTraceId.value = e?.traceId || "";
}

const expandedRuntimeEnvs = ref({});
const runtimeEnvData = ref({});
const platformRegions = ref({});
const loadingRegions = ref({});
function badgeClass(st) {
  return {
    badge: true,
    "badge-draft": st === "draft",
    "badge-pending": st === "pending_review",
    "badge-ok": st === "approved",
    "badge-reject": st === "rejected",
  };
}

async function toggleGroup(id) {
  expandedGroups.value[id] = !expandedGroups.value[id];
  if (expandedGroups.value[id]) {
    // Proactively load runtime env data for all versions so version-item rows
    // can immediately show UserData-template-missing warnings.
    const versions = getGroupVersions(id);
    await Promise.all([
      loadAllPlatformRegions(),
      loadUserdataTemplateOptions(),
      ...versions.map((im) => loadRuntimeEnvData(im.id)),
    ]);
  }
}

function toggleRejectNote(id) {
  expandedRejectNotes.value[id] = !expandedRejectNotes.value[id];
}

async function toggleRuntimeEnv(containerImageId) {
  expandedRuntimeEnvs.value[containerImageId] = !expandedRuntimeEnvs.value[containerImageId];
  if (expandedRuntimeEnvs.value[containerImageId]) {
    await Promise.all([
      loadRuntimeEnvData(containerImageId),
      loadAllPlatformRegions(),
      loadUserdataTemplateOptions(),
    ]);
  }
}

async function loadRuntimeEnvData(containerImageId) {
  if (runtimeEnvData.value[containerImageId] !== undefined) return; // already cached
  try {
    const data = await api(`/api/vendor/container-images/${containerImageId}/cloud-server-image-associations/`);
    runtimeEnvData.value[containerImageId] = data;
  } catch {
    runtimeEnvData.value[containerImageId] = [];
  }
}

function getRuntimeEnvByPlatform(containerImageId, platformType) {
  const envs = runtimeEnvData.value[containerImageId] || [];
  return envs.filter(e => e.platform_type === platformType);
}

function getRuntimeEnvForRegion(containerImageId, platformType, regionId) {
  const envs = getRuntimeEnvByPlatform(containerImageId, platformType);
  return envs.find(e => e.region === regionId);
}

async function loadAllPlatformRegions() {
  for (const platform of cloudPlatforms) {
    if (!platformRegions.value[platform.value]) {
      await loadPlatformRegions(platform.value);
    }
  }
}

async function loadPlatformRegions(platformType) {
  if (loadingRegions.value[platformType]) return;
  loadingRegions.value[platformType] = true;
  try {
    const params = new URLSearchParams({ platform_type: platformType });
    const data = await api(`/api/vendor/cloud-server-images/regions/?${params}`);
    if (data.status === 'success') {
      platformRegions.value[platformType] = data.regions || [];
    } else {
      platformRegions.value[platformType] = [];
      if (data.code === 'vendor_cloud_credential_missing') {
        credentialHint.value = data.message || '请先在「云平台测试密钥」配置 AccessKey';
      }
    }
  } catch (e) {
    platformRegions.value[platformType] = [];
    if (e.data?.code === 'vendor_cloud_credential_missing') {
      credentialHint.value = e.data?.message || e.message || '请先在「云平台测试密钥」配置 AccessKey';
    }
  } finally {
    loadingRegions.value[platformType] = false;
  }
}

function getPlatformRegions(platformType) {
  return platformRegions.value[platformType] || [];
}

function getGroupVersions(groupId) {
  // 列表 API 字段为 image_group（非 image_group_id）；见 containerImageGroup.js
  return filterGroupVersions(containerImages.value, groupId);
}

// OPT-20260824-063：镜像行「技能/运行说明」摘要，一次计算所有版本复用
const versionResolveSummaries = computed(() => {
  const map = {};
  for (const im of containerImages.value) {
    map[im.id] = summarizeImageResolveInfo(im);
  }
  return map;
});

function userdataTemplateLabel(t) {
  return userdataTemplateDisplayLabel(t, userdataTemplateOptions.value);
}

function runtimeEnvHasUserdataTemplate(env) {
  return normalizeUserdataTemplate(env?.cloud_server_image?.userdata_template) != null;
}

function runtimeEnvUserdataTemplateLabel(env) {
  if (!runtimeEnvHasUserdataTemplate(env)) return "未选择";
  return userdataTemplateDisplayLabel(
    env.cloud_server_image.userdata_template,
    userdataTemplateOptions.value,
  );
}

function getRuntimeEnvStatus(containerImageId, platformType, regionId) {
  const env = getRuntimeEnvForRegion(containerImageId, platformType, regionId);
  if (!env) return "unset";
  return runtimeEnvHasUserdataTemplate(env) ? "set" : "unavailable";
}

/** Returns true when the version has at least one region env that is bound to a
 *  server image but missing a UserData template — i.e. the region is "unavailable". */
function versionHasUnavailableRegion(containerImageId) {
  const envs = runtimeEnvData.value[containerImageId] || [];
  if (envs.length === 0) return false;
  return envs.some((env) => !runtimeEnvHasUserdataTemplate(env));
}

function getRuntimeEnvRowClass(containerImageId, platformType, regionId) {
  const status = getRuntimeEnvStatus(containerImageId, platformType, regionId);
  if (status === "unset") return "unset";
  if (status === "unavailable") return "unavailable";
  return "";
}

async function loadUserdataTemplateOptions() {
  try {
    userdataTemplateOptions.value = await api("/api/vendor/userdata-templates/");
  } catch {
    userdataTemplateOptions.value = [];
  }
}
function formatDate(dateStr) {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
async function refreshGroups() {
  const groups = await api("/api/vendor/image-groups/");
  imageGroups.value = enrichImageGroups(groups, containerImages.value);
}

async function refreshImages() {
  containerImages.value = await api("/api/vendor/container-images/");
  imageGroups.value = enrichImageGroups(imageGroups.value, containerImages.value);
  // Preload runtime env data for all versions so that
  // versionHasUnavailableRegion / isMarketplaceUnavailable badges render immediately.
  await Promise.all([
    loadAllPlatformRegions(),
    loadUserdataTemplateOptions(),
    ...containerImages.value.map((im) => loadRuntimeEnvData(im.id)),
  ]);
}

async function refreshServers() {
  serverImages.value = await api("/api/vendor/cloud-server-images/");
}

const { openGroupCreate, openGroupEdit, onGroupIconFile, saveGroup } = createVendorImageGroupFormApi({
  api,
  groupEditingId,
  groupForm,
  groupIconFile,
  groupIconPreview,
  groupFormError,
  groupFormErrorTraceId,
  groupOpen,
  groupSaving,
  groupSaveGuard,
  clearMsg,
  setMsgError,
  msg,
  refreshGroups,
});

const {
  actionConfirm,
  versionActionBusy,
  openVersionDelete,
  openVersionWithdraw,
  openVersionActivate,
  openGroupDelete,
  actionConfirmTitle,
  actionConfirmMessage,
  actionConfirmLabel,
  actionConfirmDanger,
  confirmPortalAction,
  submitVersionReview,
} = createVendorImageActionConfirm({
  setMsgError,
  refreshGroups,
  refreshImages,
});

async function removeGroup(g) {
  if (getGroupVersions(g.id).length > 0) {
    setMsgText("该镜像组下有版本，请先删除所有版本");
    return;
  }
  openGroupDelete(g);
}

function openAddVersion(g) {
  imageVersionModals.value?.openCreate(g);
}

function openEdit(im) {
  imageVersionModals.value?.openEdit(im);
}

async function onImageVersionSaved() {
  clearMsg();
  await refreshImages();
  await refreshGroups();
}

// submitReview 委托给 vendorImageActionConfirm 的 submitVersionReview，
// 共享 versionWriteGuard：防双击重复提交并携带 Idempotency-Key。
async function submitReview(im) {
  await submitVersionReview(im);
}
/** 关联接口里的服务器镜像若尚未出现在列表缓存中，合并进来以便下拉与筛选一致 */
function mergeAssociatedCloudServerImageIntoList(existingEnv) {
  const csi = existingEnv?.cloud_server_image;
  if (!csi || csi.id == null || csi.id === "") return;
  const sid = String(csi.id);
  if (serverImages.value.some((s) => String(s.id) === sid)) return;
  serverImages.value = [...serverImages.value, csi];
}

async function openRegionEnv(im, platform, region) {
  clearRegionEnvError();
  regionEnvImageId.value = im.id;
  regionEnvPlatform.value = platform;
  regionEnvRegion.value = region;
  envTargetArchs.value = im.target_architectures || [];
  await loadUserdataTemplateOptions();
  await Promise.all([loadRuntimeEnvData(im.id), refreshServers()]);
  const existingEnv = getRuntimeEnvForRegion(im.id, platform, region.id);
  mergeAssociatedCloudServerImageIntoList(existingEnv);
  const ut = normalizeUserdataTemplate(existingEnv?.cloud_server_image?.userdata_template);
  regionEnvInitialCsId.value = existingEnv ? String(existingEnv.cloud_server_image.id) : "";
  regionEnvInitialUdId.value = ut?.id != null ? String(ut.id) : "";
  regionEnvOpen.value = true;
}

async function handleRegionEnvSave({ containerImageId, platformType, regionId, cloudServerImageId, userdataTemplateId }) {
  savingRegionEnv.value = true;
  clearRegionEnvError();
  try {
    const payload = buildRegionEnvAssociationPayload({
      platformType,
      regionId,
      cloudServerImageId,
      userdataTemplateId,
    });
    await api(`/api/vendor/container-images/${containerImageId}/cloud-server-image-association/`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    regionEnvOpen.value = false;
    await loadRuntimeEnvData(containerImageId);
    await refreshServers();
  } catch (e) {
    // Modal covers page-level p.msg; keep failure visible inside the dialog.
    setRegionEnvError(e);
  } finally {
    savingRegionEnv.value = false;
  }
}

  Object.assign(d, {
    clearRegionEnvError,
    setRegionEnvError,
    badgeClass,
    toggleGroup,
    toggleRejectNote,
    toggleRuntimeEnv,
    loadRuntimeEnvData,
    getRuntimeEnvByPlatform,
    getRuntimeEnvForRegion,
    loadAllPlatformRegions,
    loadPlatformRegions,
    getPlatformRegions,
    getGroupVersions,
    userdataTemplateLabel,
    runtimeEnvHasUserdataTemplate,
    runtimeEnvUserdataTemplateLabel,
    getRuntimeEnvStatus,
    versionHasUnavailableRegion,
    getRuntimeEnvRowClass,
    loadUserdataTemplateOptions,
    formatDate,
    refreshGroups,
    refreshImages,
    refreshServers,
    openGroupCreate,
    openGroupEdit,
    onGroupIconFile,
    saveGroup,
    removeGroup,
    openAddVersion,
    openEdit,
    onImageVersionSaved,
    submitReview,
    mergeAssociatedCloudServerImageIntoList,
    openRegionEnv,
    handleRegionEnvSave,
    openVersionDelete,
    openVersionWithdraw,
    openVersionActivate,
    confirmPortalAction,
    actionConfirm,
    actionConfirmTitle,
    actionConfirmMessage,
    actionConfirmLabel,
    actionConfirmDanger,
    versionActionBusy,
    versionHasAction,
    withdrawButtonLabel,
    imageGroups,
    containerImages,
    serverImages,
    expandedGroups,
    expandedRejectNotes,
    groupOpen,
    groupEditingId,
    groupForm,
    groupIconPreview,
    groupFormError,
    groupFormErrorTraceId,
    groupSaving,
    imageVersionModals,
    envTargetArchs,
    regionEnvOpen,
    regionEnvImageId,
    regionEnvPlatform,
    regionEnvRegion,
    regionEnvInitialCsId,
    regionEnvInitialUdId,
    savingRegionEnv,
    regionEnvError,
    regionEnvErrorTraceId,
    expandedRuntimeEnvs,
    runtimeEnvData,
    platformRegions,
    loadingRegions,
    versionResolveSummaries,
    formatUnavailableBadge,
    groupActiveVersion,
    isActiveVersion,
    isMarketplaceUnavailable,
    statusDisplay,
    cloudPlatforms,
    userdataTemplateOptions,
  })
}
