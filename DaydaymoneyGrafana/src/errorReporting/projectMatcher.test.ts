import {
  applyProjectMatchTemplate,
  extractDaydaymoneyMetaFromLogLine,
  extractLogCaptures,
  resolveMatchedProjects,
  resolveMatchedProjectsByDaydaymoneyMeta,
} from './projectMatcher';
import type { CaptureRule } from '../types/settings';
import type { Task2appProject } from '../api/task2appClient';

describe('projectMatcher', () => {
  const projects: Task2appProject[] = [
    { id: 'p1', name: 'saas-backend', description: '后端服务', tags: ['api', 'django'] },
    { id: 'p2', name: 'front-web', description: 'Vue 前端', tags: ['ui'] },
  ];

  test('extractLogCaptures returns capture groups verbatim', () => {
    expect(extractLogCaptures('service=(\\S+)', 'level=error service=saas-backend timeout')).toEqual(['saas-backend']);
  });

  test('extractLogCaptures does not rewrite user regex or strip quotes', () => {
    const log = '{"service":"task-container-gateway","trace_id":"x"}';
    expect(extractLogCaptures('"service":(\\\\S+)', log)).toEqual([]);
    expect(extractLogCaptures('"service":(\\S+)', log)).toEqual(['"task-container-gateway","trace_id":"x"}']);
    expect(extractLogCaptures('"service":"([^"]+)"', log)).toEqual(['task-container-gateway']);
  });

  test('applyProjectMatchTemplate substitutes $1', () => {
    expect(applyProjectMatchTemplate('$1', ['saas-backend'])).toBe('saas-backend');
  });

  test('resolveMatchedProjects uses log capture and project pattern against name', () => {
    const rule: Pick<CaptureRule, 'logFieldRegex' | 'projectMatchPattern' | 'sampleLogLine'> = {
      logFieldRegex: 'service=(\\S+)',
      projectMatchPattern: '$1',
      sampleLogLine: '',
    };
    const matched = resolveMatchedProjects(rule, 'error service=saas-backend timeout', projects);
    expect(matched.map((p) => p.id)).toEqual(['p1']);
  });

  test('resolveMatchedProjects matches description and tags', () => {
    const rule = {
      logFieldRegex: '',
      projectMatchPattern: 'vue',
      sampleLogLine: '',
    };
    const matched = resolveMatchedProjects(rule, 'ignored', projects);
    expect(matched.map((p) => p.id)).toEqual(['p2']);
  });

  test('resolveMatchedProjects with .* matches all projects even when log capture fails', () => {
    const log = '{"service":"task-container-gateway"}';
    const rule = {
      logFieldRegex: '"service":(\\\\S+)',
      projectMatchPattern: '.*',
      sampleLogLine: log,
    };
    const matched = resolveMatchedProjects(rule, log, projects);
    expect(matched.map((p) => p.id)).toEqual(['p1', 'p2']);
  });

  test('resolveMatchedProjects with $1 still requires successful log capture', () => {
    const log = '{"service":"task-container-gateway"}';
    const rule = {
      logFieldRegex: '"service":(\\\\S+)',
      projectMatchPattern: '$1',
      sampleLogLine: log,
    };
    expect(resolveMatchedProjects(rule, log, projects)).toEqual([]);
  });

  test('extractDaydaymoneyMetaFromLogLine parses JSON service id and comma tags', () => {
    const log =
      '{"level":"error","daydaymoney_service_id":"taskProjectService","daydaymoney_tags":"svc:taskProjectService,domain:project"}';
    expect(extractDaydaymoneyMetaFromLogLine(log)).toEqual({
      serviceId: 'taskProjectService',
      tags: ['svc:taskProjectService', 'domain:project'],
    });
  });

  test('resolveMatchedProjectsByDaydaymoneyMeta matches project tags case-insensitively', () => {
    const daydaymoneyProjects: Task2appProject[] = [
      { id: 'p1', name: 'Proj A', description: '', tags: ['SVC:taskProjectService'] },
      { id: 'p2', name: 'Proj B', description: '', tags: ['ui'] },
    ];
    const meta = extractDaydaymoneyMetaFromLogLine(
      '{"daydaymoney_service_id":"taskProjectService","daydaymoney_tags":["svc:taskProjectService"]}'
    );
    const matched = resolveMatchedProjectsByDaydaymoneyMeta(meta, daydaymoneyProjects);
    expect(matched.map((p) => p.id)).toEqual(['p1']);
  });

  test('resolveMatchedProjects prefers daydaymoney meta over regex', () => {
    const daydaymoneyProjects: Task2appProject[] = [
      { id: 'p1', name: 'saas-backend', description: '', tags: ['api'] },
      { id: 'p2', name: 'auth-service', description: '', tags: ['svc:task-auth'] },
    ];
    const log = '{"service":"saas-backend","daydaymoney_service_id":"task-auth","daydaymoney_tags":"svc:task-auth"}';
    const rule = {
      logFieldRegex: '"service":"([^"]+)"',
      projectMatchPattern: '$1',
      sampleLogLine: '',
    };
    const matched = resolveMatchedProjects(rule, log, daydaymoneyProjects);
    expect(matched.map((p) => p.id)).toEqual(['p2']);
  });

  test('resolveMatchedProjects falls back to regex when daydaymoney meta has no match', () => {
    const log = '{"daydaymoney_service_id":"unknown-service","daydaymoney_tags":"svc:unknown-service"}';
    const rule = {
      logFieldRegex: '',
      projectMatchPattern: 'vue',
      sampleLogLine: '',
    };
    const matched = resolveMatchedProjects(rule, log, projects);
    expect(matched.map((p) => p.id)).toEqual(['p2']);
  });
});
