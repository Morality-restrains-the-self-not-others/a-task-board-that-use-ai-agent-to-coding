import {
  formatLogCapturePreview,
  getProjectMatchPreviewState,
  isLogCapturePreviewFailure,
} from './ProjectMatchEditor';

describe('formatLogCapturePreview', () => {
  test('returns empty string when regex is missing', () => {
    expect(formatLogCapturePreview('', 'level=error service=saas-backend')).toBe('');
  });

  test('prompts for sample log first', () => {
    expect(formatLogCapturePreview('service=(\\S+)', '')).toBe('请先填写样例日志');
  });

  test('formats capture groups from sample log', () => {
    expect(formatLogCapturePreview('service=(\\S+)', 'level=error service=saas-backend timeout')).toBe(
      '$1 = saas-backend'
    );
  });

  test('reports when regex does not match sample log', () => {
    expect(formatLogCapturePreview('service=(\\S+)', 'level=error only')).toBe('未从样例日志中匹配到捕获组');
  });
});

describe('isLogCapturePreviewFailure', () => {
  test('detects capture failure message', () => {
    expect(isLogCapturePreviewFailure('未从样例日志中匹配到捕获组')).toBe(true);
    expect(isLogCapturePreviewFailure('$1 = saas-backend')).toBe(false);
  });
});

describe('getProjectMatchPreviewState', () => {
  const projects = [{ id: 'p1', name: 'saas-backend', description: '后端' }];

  test('returns failure in red-worthy state when no projects match', () => {
    const state = getProjectMatchPreviewState({
      tenantId: 't1',
      workspaceId: 'w1',
      projectsLoading: false,
      projectsError: '',
      sampleLogLine: 'log',
      logFieldRegex: '',
      projectMatchPattern: 'nomatch',
      projects,
      matchedProjects: [],
    });
    expect(state.kind).toBe('failure');
    expect(state).toMatchObject({
      message: '未匹配到任何项目（共 1 个），请检查项目匹配表达式',
    });
  });

  test('returns failure when capture is required but missing', () => {
    const state = getProjectMatchPreviewState({
      tenantId: 't1',
      workspaceId: 'w1',
      projectsLoading: false,
      projectsError: '',
      sampleLogLine: 'log',
      logFieldRegex: 'missing',
      projectMatchPattern: '$1',
      projects,
      matchedProjects: [],
    });
    expect(state.kind).toBe('failure');
    expect(state.message).toContain('日志字段捕获未成功');
  });

  test('returns success for direct project match without sample log', () => {
    const state = getProjectMatchPreviewState({
      tenantId: 't1',
      workspaceId: 'w1',
      projectsLoading: false,
      projectsError: '',
      sampleLogLine: '',
      logFieldRegex: '"service":"([^"]+)"',
      projectMatchPattern: '.*',
      projects,
      matchedProjects: projects,
    });
    expect(state.kind).toBe('success');
  });
});
