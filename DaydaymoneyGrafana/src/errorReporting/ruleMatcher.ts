import type { CaptureRule } from '../types/settings';
import type { Task2appProject } from '../api/task2appClient';

export type MatchedLog = {
  message: string;
  service: string;
  timestamp: string;
  traceId?: string;
  rawLine: string;
};

export function matchesRuleLine(rule: CaptureRule, line: string): boolean {
  if (rule.matchMode === 'custom_regex') {
    const pattern = rule.messageRegex?.trim();
    if (!pattern) {
      return false;
    }
    try {
      return new RegExp(pattern).test(line);
    } catch {
      return false;
    }
  }
  return true;
}

import { resolveMatchedProjects } from './projectMatcher';

export function resolveProjectsFromLog(
  rule: CaptureRule,
  logLine: string,
  projects: Task2appProject[]
): Task2appProject[] {
  return resolveMatchedProjects(rule, logLine, projects);
}

export function extractBranchFromLog(rule: CaptureRule, logLine: string): string {
  const pattern = rule.branchMatchRegex?.trim();
  if (!pattern) {
    return '';
  }
  try {
    const match = logLine.match(new RegExp(pattern));
    return (match?.[1] ?? match?.[0] ?? '').trim();
  } catch {
    return '';
  }
}

export function buildTaskTitle(rule: CaptureRule, log: MatchedLog): string {
  const preview = log.message.slice(0, 120);
  const prefix = rule.name ? `[Grafana:${rule.name}]` : '[Grafana告警]';
  const title = log.service ? `${prefix}[${log.service}] ${preview}` : `${prefix} ${preview}`;
  return title.slice(0, 255);
}

export function buildTaskDescription(rule: CaptureRule, log: MatchedLog): string {
  const lines = [
    '## Grafana 自动捕获任务',
    '',
    `- **规则**: ${rule.name}`,
    `- **匹配模式**: ${rule.matchMode}`,
    `- **时间**: ${log.timestamp}`,
    `- **服务**: ${log.service}`,
    `- **消息**: ${log.message}`,
  ];
  if (log.traceId) {
    lines.push(`- **traceId**: ${log.traceId}`);
  }
  lines.push('', '### 原始日志', '```', log.rawLine.slice(0, 4000), '```');
  return lines.join('\n');
}
