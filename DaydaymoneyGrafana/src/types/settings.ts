export type CaptureMatchMode = 'error_default' | 'custom_loki' | 'custom_regex';

export type CaptureRule = {
  id: string;
  name: string;
  enabled: boolean;
  matchMode: CaptureMatchMode;
  lokiQuery?: string;
  messageRegex?: string;
  tenantId: string;
  ownerMemberId: string;
  workspaceId: string;
  /** 从日志行提取字段（捕获组），供 projectMatchPattern 用 $1、$2 引用 */
  logFieldRegex: string;
  /** 配置页预览用样例日志 */
  sampleLogLine: string;
  /** 匹配项目名称/描述/标签；可引用 $1、$2 */
  projectMatchPattern: string;
  branchMatchRegex: string;
  mergeTargetBranch: string;
};

export type AppPluginSettings = {
  task2appApiBaseUrl?: string;
  task2appAccount?: string;
  task2appUserId?: string;
  task2appSessionToken?: string;
  captureRules?: CaptureRule[];
  lokiPollEnabled?: boolean;
  lokiDatasourceUid?: string;
  lokiPollIntervalSec?: number;
};

export const DEFAULT_LOKI_ERROR_QUERY = '{job="runall"} | json | level=~"(?i)error|fatal"';

export function defaultCaptureRule(): CaptureRule {
  return {
    id: `rule-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    name: '默认 Error 捕获',
    enabled: true,
    matchMode: 'error_default',
    lokiQuery: DEFAULT_LOKI_ERROR_QUERY,
    messageRegex: '',
    tenantId: '',
    ownerMemberId: '',
    workspaceId: '',
    logFieldRegex: '',
    sampleLogLine: '',
    projectMatchPattern: '.*',
    branchMatchRegex: '',
    mergeTargetBranch: 'main',
  };
}

export function normalizeCaptureRule(rule: Partial<CaptureRule> & Pick<CaptureRule, 'id' | 'name' | 'enabled' | 'matchMode'>): CaptureRule {
  const base = defaultCaptureRule();
  return {
    ...base,
    ...rule,
    logFieldRegex: rule.logFieldRegex ?? base.logFieldRegex,
    sampleLogLine: rule.sampleLogLine ?? base.sampleLogLine,
    projectMatchPattern: rule.projectMatchPattern ?? base.projectMatchPattern,
  };
}

export function lokiQueryForRule(rule: CaptureRule): string {
  if (rule.matchMode === 'custom_loki' && rule.lokiQuery?.trim()) {
    return rule.lokiQuery.trim();
  }
  return DEFAULT_LOKI_ERROR_QUERY;
}
