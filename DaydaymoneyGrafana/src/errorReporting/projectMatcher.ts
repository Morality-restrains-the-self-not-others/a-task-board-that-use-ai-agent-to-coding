import type { Task2appProject } from '../api/task2appClient';
import type { CaptureRule } from '../types/settings';

export type DaydaymoneyLogMeta = {
  serviceId?: string;
  tags: string[];
};

function parseDaydaymoneyTagsValue(value: unknown): string[] {
  if (typeof value === 'string') {
    return value
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean);
  }
  if (Array.isArray(value)) {
    return value.map((tag) => String(tag).trim()).filter(Boolean);
  }
  return [];
}

function readDaydaymoneyMetaFromObject(obj: Record<string, unknown>): DaydaymoneyLogMeta {
  const serviceId =
    typeof obj.daydaymoney_service_id === 'string' ? obj.daydaymoney_service_id.trim() : undefined;
  const tags = parseDaydaymoneyTagsValue(obj.daydaymoney_tags);
  if (!serviceId && tags.length === 0) {
    return { tags: [] };
  }
  return { serviceId: serviceId || undefined, tags };
}

export function extractDaydaymoneyMetaFromLogLine(logLine: string): DaydaymoneyLogMeta {
  const trimmed = logLine.trim();
  if (!trimmed) {
    return { tags: [] };
  }

  try {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>;
    if (parsed && typeof parsed === 'object') {
      const meta = readDaydaymoneyMetaFromObject(parsed);
      if (meta.serviceId || meta.tags.length > 0) {
        return meta;
      }
    }
  } catch {
    // fall through to embedded JSON extraction
  }

  const jsonMatch = trimmed.match(/\{[\s\S]*\}/);
  if (jsonMatch) {
    try {
      const parsed = JSON.parse(jsonMatch[0]) as Record<string, unknown>;
      if (parsed && typeof parsed === 'object') {
        return readDaydaymoneyMetaFromObject(parsed);
      }
    } catch {
      return { tags: [] };
    }
  }

  return { tags: [] };
}

function tagsEqual(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

export function resolveMatchedProjectsByDaydaymoneyMeta(
  meta: DaydaymoneyLogMeta,
  projects: Task2appProject[]
): Task2appProject[] {
  const queryTags = new Set<string>();
  for (const tag of meta.tags) {
    const trimmed = tag.trim();
    if (trimmed) {
      queryTags.add(trimmed);
    }
  }
  if (meta.serviceId?.trim()) {
    queryTags.add(`svc:${meta.serviceId.trim()}`);
  }
  if (queryTags.size === 0) {
    return [];
  }

  return projects.filter((project) => {
    const projectTags = (project.tags ?? []).map((tag) => tag.trim()).filter(Boolean);
    return projectTags.some((projectTag) =>
      [...queryTags].some((queryTag) => tagsEqual(projectTag, queryTag))
    );
  });
}

export function extractLogCaptures(regexStr: string, logLine: string): string[] {
  const pattern = regexStr.trim();
  if (!pattern) {
    return [];
  }
  try {
    const match = logLine.match(new RegExp(pattern));
    if (!match) {
      return [];
    }
    return match.slice(1).map((value) => value ?? '');
  } catch {
    return [];
  }
}

export function applyProjectMatchTemplate(pattern: string, captures: string[]): string {
  return pattern.replace(/\$(\d+)/g, (_, index) => captures[Number(index) - 1] ?? '');
}

export function projectMatchUsesCaptures(pattern: string): boolean {
  return /\$\d+/.test(pattern);
}

function projectSearchFields(project: Task2appProject): string[] {
  return [project.name, project.description ?? '', ...(project.tags ?? [])]
    .map((value) => value.trim())
    .filter(Boolean);
}

function resolveMatchedProjectsByRegex(
  rule: Pick<CaptureRule, 'logFieldRegex' | 'projectMatchPattern' | 'sampleLogLine'>,
  logLine: string,
  projects: Task2appProject[]
): Task2appProject[] {
  const logCapture = rule.logFieldRegex?.trim() ?? '';
  const projectPattern = rule.projectMatchPattern?.trim() || '.*';
  const captures = logCapture ? extractLogCaptures(logCapture, logLine) : [];

  if (logCapture && projectMatchUsesCaptures(projectPattern) && captures.length === 0) {
    return [];
  }

  const resolvedPattern = applyProjectMatchTemplate(projectPattern, captures);
  let regex: RegExp;
  try {
    regex = new RegExp(resolvedPattern, 'i');
  } catch {
    return [];
  }

  return projects.filter((project) => projectSearchFields(project).some((field) => regex.test(field)));
}

export function resolveMatchedProjects(
  rule: Pick<CaptureRule, 'logFieldRegex' | 'projectMatchPattern' | 'sampleLogLine'>,
  logLine: string,
  projects: Task2appProject[]
): Task2appProject[] {
  const daydaymoneyMeta = extractDaydaymoneyMetaFromLogLine(logLine);
  if (daydaymoneyMeta.serviceId || daydaymoneyMeta.tags.length > 0) {
    const daydaymoneyMatched = resolveMatchedProjectsByDaydaymoneyMeta(daydaymoneyMeta, projects);
    if (daydaymoneyMatched.length > 0) {
      return daydaymoneyMatched;
    }
  }

  return resolveMatchedProjectsByRegex(rule, logLine, projects);
}
