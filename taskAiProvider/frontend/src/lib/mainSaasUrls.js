/** 主站（Saas_project）入口，来自 Vite `VITE_MAIN_SAAS_ORIGIN` 或构建时写入的 port_config。 */
export function mainSaasBaseUrl() {
  const raw = import.meta.env.VITE_MAIN_SAAS_ORIGIN || "";
  return String(raw).replace(/\/$/, "");
}

function mainSaasAdminSsoBaseUrl() {
  const raw = import.meta.env.VITE_MAIN_SAAS_ADMIN_SSO_ORIGIN || "";
  return String(raw).replace(/\/$/, "") || mainSaasBaseUrl();
}

function mainSaasVendorSsoBaseUrl() {
  const raw = import.meta.env.VITE_MAIN_SAAS_VENDOR_SSO_ORIGIN || "";
  return String(raw).replace(/\/$/, "") || mainSaasBaseUrl();
}

/** 主站浏览器会话所在源（通常为 Vue dev `port_config.vue`，与 django API 端口可能不同）。 */
export function mainSaasSessionBaseUrl() {
  const raw = import.meta.env.VITE_MAIN_SAAS_SESSION_ORIGIN || "";
  return String(raw).replace(/\/$/, "");
}

/**
 * 主站 Django `GET /logout/`（经 Vue 代理时须使用 session 源），登出后 `next` 回到镜像市场等地址。
 */
export function mainSaasLogoutUrlWithNext(nextAbsoluteUrl) {
  const base = mainSaasSessionBaseUrl();
  if (!base) {
    return "";
  }
  if (!nextAbsoluteUrl) {
    return `${base}/logout/`;
  }
  return `${base}/logout/?next=${encodeURIComponent(nextAbsoluteUrl)}`;
}

export function ssoAdminEntryUrl() {
  const base = mainSaasAdminSsoBaseUrl();
  const p = "/api/accounts/sso/ai-provider/admin/";
  return base ? `${base}${p}` : p;
}

export function ssoVendorEntryUrl() {
  const base = mainSaasVendorSsoBaseUrl();
  const p = "/api/accounts/sso/ai-provider/vendor/";
  return base ? `${base}${p}` : p;
}
