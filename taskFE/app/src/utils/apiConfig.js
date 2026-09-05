// API 配置文件 — convention: /api/${serviceName}/${funcName}/key/value/...
export const API_BASE_URL = '';

export const API_ENDPOINTS = {
  // 认证相关
  LOGIN: `${API_BASE_URL}/auth/login/`,
  REGISTER: `${API_BASE_URL}/auth/register/`,
  RESET_PASSWORD_REQUEST: `${API_BASE_URL}/auth/reset-password-request/`,
  RESET_PASSWORD: `${API_BASE_URL}/auth/reset-password/`,
  ACTIVATE: `${API_BASE_URL}/auth/activate/`,

  // 用户相关
  SEND_VERIFICATION_CODE: `${API_BASE_URL}/api/accounts/users/send_verification_code/`,
  RESEND_ACTIVATION_EMAIL: `${API_BASE_URL}/api/accounts/users/resend_activation_email/`,

  // 项目相关 — /api/projects/{funcName}/tenant_id/{tid}
  PROJECTS: (tenantId) => `${API_BASE_URL}/api/projects/tenant_id/${tenantId}`,
  PROJECT_DETAIL: (tenantId, projectId) => `${API_BASE_URL}/api/projects/tenant_id/${tenantId}/${projectId}`,

  // 工作区相关 — /api/projects/workspaces/tenant_id/{tid}
  WORKSPACES: (tenantId) => `${API_BASE_URL}/api/projects/workspaces/tenant_id/${tenantId}`,
  WORKSPACE_DETAIL: (tenantId, workspaceId) => `${API_BASE_URL}/api/projects/workspaces/tenant_id/${tenantId}/${workspaceId}`,

  // 账单相关
  BILLING: `${API_BASE_URL}/api/billing/`,

  // 云平台相关
  CLOUD_PLATFORMS: `${API_BASE_URL}/api/cloud/`,

  // 系统管理员相关
  SYSTEM_ADMIN: `${API_BASE_URL}/api/system-admin/`,
  SYSTEM_ADMIN_USERS: `${API_BASE_URL}/api/system-admin/users/`,
};

export default {
  API_BASE_URL,
  API_ENDPOINTS
};
