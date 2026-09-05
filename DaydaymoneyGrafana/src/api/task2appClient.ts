export type Task2appCompany = {
  id: string;
  name: string;
};

export type Task2appLoginUser = {
  id?: string;
  companies?: Task2appCompany[];
};

export type Task2appLoginResult = {
  token?: string;
  user?: Task2appLoginUser;
  user_id?: string;
  userId?: string;
  error?: string;
};

export type Task2appProject = {
  id: string;
  name: string;
  description?: string;
  tags?: string[];
};

export type Task2appWorkspace = {
  id: string;
  name: string;
  company_id?: string;
  company_name?: string;
};

export type Task2appMember = {
  id: string;
  user_id?: string;
  username?: string;
  display_name?: string;
  member_name?: string;
  email?: string;
};

export function memberDisplayName(member: Pick<Task2appMember, 'display_name' | 'member_name' | 'email' | 'username' | 'id'>): string {
  return (
    member.display_name?.trim() ||
    member.member_name?.trim() ||
    member.email?.trim() ||
    member.username?.trim() ||
    member.id
  );
}

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.trim().replace(/\/+$/, '');
}

function authHeader(token: string): string {
  const value = token.trim();
  if (value.startsWith('at_')) {
    return `Bearer ${value}`;
  }
  return `Token ${value}`;
}

function newRequestTraceId(): string {
  try {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return crypto.randomUUID();
    }
  } catch {
    /* ignore */
  }
  return `grafana-${Date.now()}-${Math.random().toString(36).slice(2, 12)}`;
}

function resolveTraceId(res: Response, requestTraceId: string, bodyText: string): string {
  const fromHeader = res.headers.get('X-Trace-Id') || res.headers.get('x-trace-id');
  if (fromHeader?.trim()) {
    return fromHeader.trim();
  }
  try {
    const j = bodyText ? (JSON.parse(bodyText) as Record<string, unknown>) : null;
    if (j && typeof j.trace_id === 'string' && j.trace_id.trim()) {
      return j.trace_id.trim();
    }
    if (j && typeof j.traceId === 'string' && j.traceId.trim()) {
      return j.traceId.trim();
    }
  } catch {
    /* ignore */
  }
  return requestTraceId;
}

async function request<T>(
  baseUrl: string,
  token: string,
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const url = `${normalizeBaseUrl(baseUrl)}${path}`;
  const requestTraceId = newRequestTraceId();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Trace-Id': requestTraceId,
  };
  if (token.trim()) {
    headers.Authorization = authHeader(token);
  }

  const res = await fetch(url, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    mode: 'cors',
    credentials: 'omit',
  });

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = new Error(`${method} ${path} → ${res.status}: ${text.slice(0, 500)}`) as Error & {
      traceId?: string;
    };
    err.traceId = resolveTraceId(res, requestTraceId, text);
    throw err;
  }

  return res.json() as Promise<T>;
}

export function isAccessTokenFormat(token: string): boolean {
  return typeof token === 'string' && token.startsWith('at_') && token.length >= 12;
}

export async function loginWithAccessToken(
  baseUrl: string,
  username: string,
  accessToken: string
): Promise<Task2appLoginResult> {
  if (!username.trim()) {
    throw new Error('请填写账号');
  }
  if (!isAccessTokenFormat(accessToken)) {
    throw new Error('访问令牌格式无效，应以 at_ 开头');
  }
  return request<Task2appLoginResult>(baseUrl, '', 'POST', '/api/accounts/users/login-with-access-token/', {
    username: username.trim(),
    access_token: accessToken.trim(),
  });
}

export async function fetchCurrentUser(
  baseUrl: string,
  token: string,
  userId: string
): Promise<{ id: string; companies: Task2appCompany[] }> {
  const uid = userId.trim();
  if (!uid) {
    throw new Error('缺少 userId，请重新登录');
  }
  const data = await request<Record<string, unknown>>(
    baseUrl,
    token,
    'GET',
    `/api/user/${encodeURIComponent(uid)}/accounts/users/me/`
  );
  const companiesRaw = Array.isArray(data.companies) ? data.companies : [];
  return {
    id: String(data.id ?? uid),
    companies: companiesRaw.map((row) => {
      const item = (row && typeof row === 'object' ? row : {}) as Record<string, unknown>;
      return {
        id: String(item.id ?? ''),
        name: String(item.name ?? ''),
      };
    }).filter((company) => company.id),
  };
}

export async function listWorkspaces(baseUrl: string, token: string, tenantId: string): Promise<Task2appWorkspace[]> {
  const data = await request<unknown>(baseUrl, token, 'GET', `/api/tenant/${tenantId}/workspaces/`);
  return normalizeWorkspaceList(data).map((workspace) => ({
    ...workspace,
    company_id: workspace.company_id || tenantId,
  }));
}

/** Resolve display name: /me company.name wins; ignore workspace company_name when it is just the id. */
export function resolveWorkspaceCompanyName(
  company: Pick<Task2appCompany, 'id' | 'name'>,
  workspace: Pick<Task2appWorkspace, 'company_id' | 'company_name'>
): string {
  const companyId = (workspace.company_id || company.id).trim();
  const fromUser = company.name?.trim() || '';
  if (fromUser) {
    return fromUser;
  }
  const fromWorkspace = workspace.company_name?.trim() || '';
  if (fromWorkspace && fromWorkspace !== companyId) {
    return fromWorkspace;
  }
  return companyId;
}

/** 聚合当前用户在所有租户下可访问的工作空间（经 tenant 前缀路由）。 */
export async function listAllWorkspaces(
  baseUrl: string,
  token: string,
  userId: string
): Promise<Task2appWorkspace[]> {
  const user = await fetchCurrentUser(baseUrl, token, userId);
  const merged: Task2appWorkspace[] = [];

  for (const company of user.companies) {
    const rows = await listWorkspaces(baseUrl, token, company.id);
    for (const workspace of rows) {
      const companyId = workspace.company_id || company.id;
      merged.push({
        ...workspace,
        company_id: companyId,
        company_name: resolveWorkspaceCompanyName(company, workspace),
      });
    }
  }

  return merged.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'));
}

export async function listProjects(
  baseUrl: string,
  token: string,
  tenantId: string,
  workspaceId: string
): Promise<Task2appProject[]> {
  const path = `/api/tenant/${tenantId}/projects/?workspace_id=${encodeURIComponent(workspaceId)}`;
  const data = await request<unknown>(baseUrl, token, 'GET', path);
  return normalizeProjectList(data);
}

function normalizeProjectList(data: unknown): Task2appProject[] {
  return extractWorkspaceRows(data).map((row) => {
    const item = (row && typeof row === 'object' ? row : {}) as Record<string, unknown>;
    const rawTags = item.tags;
    const tags = Array.isArray(rawTags)
      ? rawTags.map((tag) => String(tag).trim()).filter(Boolean)
      : undefined;
    return {
      id: String(item.id ?? ''),
      name: String(item.name ?? item.title ?? ''),
      description: item.description != null ? String(item.description) : undefined,
      tags,
    };
  });
}

export async function listMembers(baseUrl: string, token: string, tenantId: string): Promise<Task2appMember[]> {
  const data = await request<{ members?: unknown[] }>(
    baseUrl,
    token,
    'GET',
    `/api/tenant/${tenantId}/accounts/members/company_members/`
  );
  return normalizeMemberList(data.members);
}

export async function listWorkspaceCollaborators(
  baseUrl: string,
  token: string,
  tenantId: string,
  workspaceId: string
): Promise<Task2appMember[]> {
  const path = `/api/tenant/${tenantId}/projects/workspace-access/workspace-collaborators/?workspace_id=${encodeURIComponent(workspaceId)}`;
  const data = await request<unknown>(baseUrl, token, 'GET', path);
  return normalizeMemberList(extractWorkspaceRows(data));
}

export function workspaceMembersCacheKey(tenantId: string, workspaceId: string): string {
  return `${tenantId}:${workspaceId}`;
}

export type CreateTaskPayload = {
  title: string;
  description: string;
  owner: string;
  workspace_id: string;
  assignees: string[];
  projects: Array<{
    project_id: string;
    base_branch: string;
    target_branch: string;
    repo_index?: number;
  }>;
  branch_strategy?: {
    work_branch_name: string;
    merge_target_branch_name: string;
    target_branch_name?: string;
  };
  priority?: string;
};

export async function createTask(
  baseUrl: string,
  token: string,
  tenantId: string,
  workspaceId: string,
  payload: CreateTaskPayload
): Promise<{ id?: string; task_id?: string }> {
  return request(baseUrl, token, 'POST', `/api/tenant/${tenantId}/workspace/${workspaceId}/todos/`, payload);
}

function normalizeMemberList(rows: unknown[] | undefined): Task2appMember[] {
  if (!Array.isArray(rows)) {
    return [];
  }

  return rows.map((row) => {
    const item = (row && typeof row === 'object' ? row : {}) as Record<string, unknown>;
    const id = String(item.id ?? item.company_member_id ?? '');
    const memberName = String(item.member_name ?? item.display_name ?? item.username ?? '').trim();
    const email = String(item.email ?? '').trim();
    const username = String(item.username ?? '').trim();
    const normalized: Task2appMember = {
      id,
      user_id: item.user_id != null ? String(item.user_id) : item.user != null ? String(item.user) : undefined,
      username: username || undefined,
      member_name: memberName || undefined,
      email: email || undefined,
    };
    normalized.display_name = memberDisplayName(normalized);
    return normalized;
  });
}

function extractWorkspaceRows(data: unknown): unknown[] {
  if (Array.isArray(data)) {
    return data;
  }
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    for (const key of ['results', 'items', 'data']) {
      if (Array.isArray(obj[key])) {
        return obj[key] as unknown[];
      }
    }
  }
  return [];
}

function normalizeWorkspaceList(data: unknown): Task2appWorkspace[] {
  return extractWorkspaceRows(data).map((row) => {
    const item = (row && typeof row === 'object' ? row : {}) as Record<string, unknown>;
    return {
      id: String(item.id ?? ''),
      name: String(item.name ?? item.title ?? ''),
      company_id: item.company_id != null ? String(item.company_id) : undefined,
      company_name: item.company_name != null ? String(item.company_name) : undefined,
    };
  });
}
