/**
 * VendorPortal session, SSO, and shared message/credential state (OPT-20260817-016).
 */
import { computed, ref } from "vue";
import { api } from "../api";
import { mainSaasLogoutUrlWithNext, ssoVendorEntryUrl } from "../lib/mainSaasUrls.js";


export function initVendorPortalSession(d) {
const token = ref(localStorage.getItem("vendor_token") || "");
const me = ref(null);
const msg = ref("");
const msgTraceId = ref("");
const tab = ref("groups");

// OPT-20260806-064: 微信扫码用户的合成邮箱（sso-<id>@sso.invalid，RFC 2606 保留域
// 永不可投递）不直接展示 —— 邮箱与由合成前缀派生的公司名统一展示占位。
const isSyntheticEmail = computed(() =>
  /@sso\.invalid$/i.test(String(me.value?.email ?? "").trim()),
);
const displayEmail = computed(() =>
  isSyntheticEmail.value ? "微信登录用户（未绑定邮箱）" : me.value?.email || "—",
);
const displayCompanyName = computed(() => {
  const name = String(me.value?.company_name ?? "").trim();
  if (isSyntheticEmail.value && /^sso-\d+$/i.test(name)) return "未设置";
  return name || "—";
});

function clearMsg() {
  msg.value = "";
  msgTraceId.value = "";
}

function setMsgError(e) {
  msg.value = e?.message || String(e ?? "");
  msgTraceId.value = e?.traceId || "";
}

function setMsgText(text) {
  msg.value = text;
  msgTraceId.value = "";
}
const ssoVendorEntryHref = computed(() => ssoVendorEntryUrl());
const oidcVendorEntryHref = computed(() => "/api/auth/oidc/authorize/?role=vendor");
const credentialHint = ref("");
const credentialsTabRef = ref(null);
const credentials = ref([]);
const activeCredentialCount = computed(() => credentials.value.filter((c) => c.is_active).length);

async function loadMe() {
  me.value = await api("/api/vendor/auth/me/");
}
async function loadCredentials() {
  try {
    credentials.value = await api("/api/vendor/cloud-platform-credentials/");
  } catch {
    credentials.value = [];
  }
}
function clearLocalVendorSession() {
  localStorage.removeItem("vendor_token");
  localStorage.removeItem("staff_token");
  token.value = "";
  me.value = null;
}

function logout() {
  clearLocalVendorSession();
  const nextAbs = `${window.location.origin}${window.location.pathname || "/"}${window.location.search || ""}`;
  const url = mainSaasLogoutUrlWithNext(nextAbs);
  if (url) {
    window.location.assign(url);
  }
}
async function tryExchangeSsoFromHash() {
  const hash = window.location.hash || "";
  if (!hash.startsWith("#sso_bridge=")) {
    return false;
  }
  const encoded = hash.slice("#sso_bridge=".length);
  let bridge = encoded;
  try {
    bridge = decodeURIComponent(encoded);
  } catch {
    bridge = encoded;
  }
  history.replaceState(null, "", window.location.pathname + window.location.search);
  const data = await api("/api/auth/sso/exchange/", {
    method: "POST",
    body: JSON.stringify({ bridge }),
  });
  if (!data.access || data.role !== "vendor") {
    throw new Error(data.detail || "SSO 换票失败或角色不是厂商");
  }
  localStorage.removeItem("staff_token");
  localStorage.setItem("vendor_token", data.access);
  token.value = data.access;
  return true;
}

function tryExchangeOidcFromHash() {
  const hash = window.location.hash || "";
  if (!hash.startsWith("#token=")) {
    return false;
  }
  const raw = hash.slice("#token=".length);
  history.replaceState(null, "", window.location.pathname + window.location.search);
  localStorage.removeItem("staff_token");
  localStorage.setItem("vendor_token", raw);
  token.value = raw;
  return true;
}

  Object.assign(d, {
    clearMsg,
    setMsgError,
    setMsgText,
    loadMe,
    loadCredentials,
    clearLocalVendorSession,
    logout,
    tryExchangeSsoFromHash,
    tryExchangeOidcFromHash,
    token,
    me,
    msg,
    msgTraceId,
    tab,
    isSyntheticEmail,
    displayEmail,
    displayCompanyName,
    ssoVendorEntryHref,
    oidcVendorEntryHref,
    credentialHint,
    credentialsTabRef,
    credentials,
    activeCredentialCount,
  })
}
