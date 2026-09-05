import { isAccessTokenFormat } from '../api/task2appClient';
import {
  buildTaskTitle,
  extractBranchFromLog,
  matchesRuleLine,
  resolveProjectsFromLog,
} from '../errorReporting/ruleMatcher';
import type { CaptureRule } from '../types/settings';

describe('ruleMatcher', () => {
  const baseRule: CaptureRule = {
    id: 'r1',
    name: 'test',
    enabled: true,
    matchMode: 'custom_regex',
    messageRegex: 'connection refused',
    tenantId: 't1',
    ownerMemberId: 'm1',
    workspaceId: 'w1',
    logFieldRegex: '',
    sampleLogLine: '',
    projectMatchPattern: 'saas-backend',
    branchMatchRegex: 'branch[:=](\\S+)',
    mergeTargetBranch: 'main',
  };

  test('matchesRuleLine with custom regex', () => {
    expect(matchesRuleLine(baseRule, 'saas-backend connection refused')).toBe(true);
    expect(matchesRuleLine(baseRule, 'ok')).toBe(false);
  });

  test('resolveProjectsFromLog matches project name', () => {
    const projects = [
      { id: 'p1', name: 'saas-backend' },
      { id: 'p2', name: 'other' },
    ];
    const matched = resolveProjectsFromLog(baseRule, 'error in saas-backend', projects);
    expect(matched).toHaveLength(1);
    expect(matched[0].id).toBe('p1');
  });

  test('extractBranchFromLog uses first capture group', () => {
    expect(extractBranchFromLog(baseRule, 'deploy branch:feature-x')).toBe('feature-x');
  });

  test('buildTaskTitle truncates to 255', () => {
    const title = buildTaskTitle(baseRule, {
      message: 'x'.repeat(300),
      service: 'svc',
      timestamp: '2026-07-05',
      rawLine: 'line',
    });
    expect(title.length).toBeLessThanOrEqual(255);
    expect(title).toContain('[Grafana:test]');
  });
});

describe('task2appClient helpers', () => {
  test('isAccessTokenFormat validates at_ prefix', () => {
    expect(isAccessTokenFormat('at_abcdefghijkl')).toBe(true);
    expect(isAccessTokenFormat('token')).toBe(false);
  });
});
