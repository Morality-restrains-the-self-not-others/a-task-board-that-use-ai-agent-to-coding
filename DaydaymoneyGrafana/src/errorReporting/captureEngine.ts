import {
  createTask,
  listProjects,
  type CreateTaskPayload,
  type Task2appProject,
} from '../api/task2appClient';
import type { AppPluginSettings, CaptureRule } from '../types/settings';
import {
  buildTaskDescription,
  buildTaskTitle,
  extractBranchFromLog,
  matchesRuleLine,
  resolveProjectsFromLog,
  type MatchedLog,
} from './ruleMatcher';

const projectCache = new Map<string, { projects: Task2appProject[]; fetchedAt: number }>();
const PROJECT_CACHE_TTL_MS = 5 * 60_000;
const seenFingerprints = new Map<string, number>();
const SEEN_TTL_MS = 60_000;

function cacheKey(tenantId: string, workspaceId: string): string {
  return `${tenantId}:${workspaceId}`;
}

async function getProjectsForRule(
  settings: AppPluginSettings,
  rule: CaptureRule
): Promise<Task2appProject[]> {
  const baseUrl = settings.task2appApiBaseUrl?.trim();
  const token = settings.task2appSessionToken?.trim();
  if (!baseUrl || !token || !rule.tenantId || !rule.workspaceId) {
    return [];
  }

  const key = cacheKey(rule.tenantId, rule.workspaceId);
  const now = Date.now();
  const cached = projectCache.get(key);
  if (cached && now - cached.fetchedAt < PROJECT_CACHE_TTL_MS) {
    return cached.projects;
  }

  const projects = await listProjects(baseUrl, token, rule.tenantId, rule.workspaceId);
  projectCache.set(key, { projects, fetchedAt: now });
  return projects;
}

function pruneSeen(now: number): void {
  for (const [key, ts] of seenFingerprints.entries()) {
    if (now - ts > SEEN_TTL_MS) {
      seenFingerprints.delete(key);
    }
  }
}

function fingerprint(ruleId: string, message: string, service: string): string {
  return `${ruleId}|${service}|${message}`;
}

function canUseTaskApi(settings: AppPluginSettings): boolean {
  return Boolean(
    settings.task2appApiBaseUrl?.trim() &&
      settings.task2appSessionToken?.trim() &&
      (settings.captureRules?.length ?? 0) > 0
  );
}

function enabledRules(settings: AppPluginSettings): CaptureRule[] {
  return (settings.captureRules ?? []).filter(
    (rule) =>
      rule.enabled &&
      rule.tenantId.trim() &&
      rule.workspaceId.trim() &&
      rule.ownerMemberId.trim()
  );
}

async function createTaskForRule(settings: AppPluginSettings, rule: CaptureRule, log: MatchedLog): Promise<void> {
  const baseUrl = settings.task2appApiBaseUrl!.trim();
  const token = settings.task2appSessionToken!.trim();
  const workBranch = extractBranchFromLog(rule, log.rawLine);
  const mergeTarget = rule.mergeTargetBranch.trim() || 'main';

  let projects: Task2appProject[] = [];
  try {
    const allProjects = await getProjectsForRule(settings, rule);
    projects = resolveProjectsFromLog(rule, log.rawLine, allProjects);
  } catch (err) {
    console.warn('[DaydaymoneyGrafana] 拉取项目列表失败', err);
  }

  const payload: CreateTaskPayload = {
    title: buildTaskTitle(rule, log),
    description: buildTaskDescription(rule, log),
    owner: rule.ownerMemberId.trim(),
    workspace_id: rule.workspaceId.trim(),
    assignees: [rule.ownerMemberId.trim()],
    projects: projects.map((project) => ({
      project_id: project.id,
      base_branch: workBranch || mergeTarget,
      target_branch: mergeTarget,
      repo_index: 0,
    })),
    priority: '1',
  };

  if (workBranch || mergeTarget) {
    payload.branch_strategy = {
      work_branch_name: workBranch,
      merge_target_branch_name: mergeTarget,
      target_branch_name: mergeTarget,
    };
  }

  await createTask(baseUrl, token, rule.tenantId.trim(), rule.workspaceId.trim(), payload);
}

export async function dispatchCapturedLog(settings: AppPluginSettings, log: MatchedLog): Promise<void> {
  const now = Date.now();
  pruneSeen(now);

  const rules = enabledRules(settings);
  if (rules.length === 0) {
    return;
  }

  if (!canUseTaskApi(settings)) {
    console.warn('[DaydaymoneyGrafana] 未登录 task2app，跳过建任务', { service: log.service });
    return;
  }

  for (const rule of rules) {
    if (!matchesRuleLine(rule, log.rawLine)) {
      continue;
    }

    const fp = fingerprint(rule.id, log.message, log.service);
    if (seenFingerprints.has(fp)) {
      continue;
    }
    seenFingerprints.set(fp, now);

    try {
      await createTaskForRule(settings, rule, log);
      console.info('[DaydaymoneyGrafana] 已创建任务', { rule: rule.name, service: log.service });
    } catch (err) {
      console.error('[DaydaymoneyGrafana] 创建任务失败', { rule: rule.name, err });
    }
  }
}
