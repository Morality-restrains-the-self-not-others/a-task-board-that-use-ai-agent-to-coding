import React, { useEffect, useMemo, useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Button, Field, FieldSet, Input, Select, Switch, useStyles2 } from '@grafana/ui';
import {
  listWorkspaceCollaborators,
  memberDisplayName,
  workspaceMembersCacheKey,
  type Task2appMember,
  type Task2appWorkspace,
} from '../../api/task2appClient';
import type { CaptureMatchMode, CaptureRule } from '../../types/settings';
import { defaultCaptureRule } from '../../types/settings';
import { testIds } from '../testIds';
import { ProjectMatchEditor } from './ProjectMatchEditor';

const MATCH_MODE_OPTIONS = [
  { label: '默认 Error（level=error|fatal）', value: 'error_default' },
  { label: '自定义 Loki 查询', value: 'custom_loki' },
  { label: '自定义日志正则', value: 'custom_regex' },
];

const MATCH_MODE_LABEL: Record<CaptureMatchMode, string> = {
  error_default: '默认 Error',
  custom_loki: '自定义 Loki',
  custom_regex: '自定义正则',
};

export type CaptureRulesEditorProps = {
  rules: CaptureRule[];
  onChange: (rules: CaptureRule[]) => void;
  apiBaseUrl: string;
  sessionToken: string;
  workspaces: Task2appWorkspace[];
  workspacesLoading?: boolean;
};

type DraftState = {
  rule: CaptureRule;
  /** null = 新建；否则为列表中已有规则 id */
  sourceId: string | null;
};

export function buildCompanyOptions(workspaces: Task2appWorkspace[]) {
  const companies = new Map<string, string>();
  for (const workspace of workspaces) {
    if (!workspace.company_id) {
      continue;
    }
    companies.set(workspace.company_id, workspace.company_name?.trim() || workspace.company_id);
  }
  return Array.from(companies.entries())
    .map(([value, label]) => ({ value, label }))
    .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN'));
}

export function workspacesForCompany(workspaces: Task2appWorkspace[], tenantId: string) {
  if (!tenantId.trim()) {
    return [];
  }
  return workspaces.filter((workspace) => workspace.company_id === tenantId);
}

export function CaptureRulesEditor({
  rules,
  onChange,
  apiBaseUrl,
  sessionToken,
  workspaces,
  workspacesLoading,
}: CaptureRulesEditorProps) {
  const s = useStyles2(getStyles);
  const [membersByWorkspace, setMembersByWorkspace] = useState<Record<string, Task2appMember[]>>({});
  const [membersLoading, setMembersLoading] = useState<Record<string, boolean>>({});
  const [draft, setDraft] = useState<DraftState | null>(null);

  const companyOptions = useMemo(() => buildCompanyOptions(workspaces), [workspaces]);
  const companyLabelById = useMemo(() => {
    const map = new Map<string, string>();
    for (const option of companyOptions) {
      map.set(option.value, option.label);
    }
    return map;
  }, [companyOptions]);

  const loadWorkspaceMembers = async (tenantId: string, workspaceId: string, force = false) => {
    const cacheKey = workspaceMembersCacheKey(tenantId, workspaceId);
    if (!tenantId || !workspaceId || !sessionToken) {
      return;
    }
    if (!force && (membersByWorkspace[cacheKey] || membersLoading[cacheKey])) {
      return;
    }
    setMembersLoading((prev) => ({ ...prev, [cacheKey]: true }));
    try {
      const members = await listWorkspaceCollaborators(apiBaseUrl, sessionToken, tenantId, workspaceId);
      setMembersByWorkspace((prev) => ({ ...prev, [cacheKey]: members }));
    } catch (err) {
      console.warn('[DaydaymoneyGrafana] 拉取工作空间协作人员失败', err);
      setMembersByWorkspace((prev) => ({ ...prev, [cacheKey]: [] }));
    } finally {
      setMembersLoading((prev) => ({ ...prev, [cacheKey]: false }));
    }
  };

  useEffect(() => {
    const targets = draft ? [draft.rule, ...rules] : rules;
    for (const rule of targets) {
      if (rule.tenantId && rule.workspaceId) {
        void loadWorkspaceMembers(rule.tenantId, rule.workspaceId);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rules, draft, sessionToken, apiBaseUrl]);

  const startAdd = () => {
    setDraft({ rule: defaultCaptureRule(), sourceId: null });
  };

  const startEdit = (rule: CaptureRule) => {
    setDraft({ rule: { ...rule }, sourceId: rule.id });
  };

  const cancelEdit = () => {
    setDraft(null);
  };

  const saveDraft = () => {
    if (!draft) {
      return;
    }
    const nextRule = draft.rule;
    if (draft.sourceId) {
      onChange(rules.map((rule) => (rule.id === draft.sourceId ? nextRule : rule)));
    } else {
      onChange([...rules, nextRule]);
    }
    setDraft(null);
  };

  const removeRule = (id: string) => {
    if (draft?.sourceId === id) {
      setDraft(null);
    }
    onChange(rules.filter((rule) => rule.id !== id));
  };

  const toggleEnabled = (id: string, enabled: boolean) => {
    onChange(rules.map((rule) => (rule.id === id ? { ...rule, enabled } : rule)));
  };

  const patchDraft = (patch: Partial<CaptureRule>) => {
    setDraft((prev) => (prev ? { ...prev, rule: { ...prev.rule, ...patch } } : prev));
  };

  const onCompanySelect = (tenantId: string) => {
    if (!draft) {
      return;
    }
    const rule = draft.rule;
    const workspaceStillValid = workspaces.some(
      (workspace) => workspace.id === rule.workspaceId && workspace.company_id === tenantId
    );
    patchDraft({
      tenantId,
      workspaceId: workspaceStillValid ? rule.workspaceId : '',
      ownerMemberId: workspaceStillValid ? rule.ownerMemberId : '',
    });
  };

  const onWorkspaceSelect = (workspaceId: string) => {
    if (!draft) {
      return;
    }
    const rule = draft.rule;
    const cacheKey = workspaceMembersCacheKey(rule.tenantId, workspaceId);
    const cachedMembers = membersByWorkspace[cacheKey];
    const ownerStillValid = cachedMembers?.some((member) => member.id === rule.ownerMemberId) ?? false;
    patchDraft({
      workspaceId,
      ownerMemberId: ownerStillValid ? rule.ownerMemberId : '',
    });
    if (rule.tenantId && workspaceId) {
      void loadWorkspaceMembers(rule.tenantId, workspaceId);
    }
  };

  const editingRule = draft?.rule;
  const membersCacheKey = editingRule
    ? workspaceMembersCacheKey(editingRule.tenantId, editingRule.workspaceId)
    : '';
  const memberOptions = (membersByWorkspace[membersCacheKey] ?? []).map((member) => ({
    label: memberDisplayName(member),
    value: member.id,
  }));
  const workspaceOptions = editingRule
    ? workspacesForCompany(workspaces, editingRule.tenantId).map((workspace) => ({
        label: workspace.name,
        value: workspace.id,
      }))
    : [];

  return (
    <FieldSet label="报警捕获规则">
      <p className={s.help}>
        每条规则定义 Loki/日志匹配方式，以及关联的公司、工作空间、责任人与分支策略。命中后自动调用 task2app
        建任务接口。在列表中编辑并保存规则后，再点页面底部「保存配置」写入 Grafana。
      </p>

      {!sessionToken ? (
        <p className={s.warn}>请先登录 task2app，再选择公司与工作空间。</p>
      ) : workspacesLoading ? (
        <p className={s.hint}>正在拉取工作空间列表…</p>
      ) : workspaces.length === 0 ? (
        <p className={s.warn}>未获取到工作空间，请确认账号权限后重新登录。</p>
      ) : null}

      {rules.length === 0 && !draft ? (
        <p className={s.hint}>暂无捕获规则，点击下方「添加捕获规则」创建。</p>
      ) : null}

      {rules.map((rule, index) => {
        const workspaceName =
          workspaces.find((workspace) => workspace.id === rule.workspaceId)?.name ||
          (rule.workspaceId ? rule.workspaceId : '未选择工作空间');
        const companyName = companyLabelById.get(rule.tenantId) || (rule.tenantId ? rule.tenantId : '未选择公司');

        return (
          <div key={rule.id} className={s.ruleCard} data-testid={testIds.appConfig.captureRuleCard}>
            <div className={s.ruleHeader}>
              <div>
                <strong data-testid={testIds.appConfig.captureRuleSummaryName}>
                  {rule.name || `规则 ${index + 1}`}
                </strong>
                <div className={s.summaryMeta}>
                  {MATCH_MODE_LABEL[rule.matchMode]} · {companyName} · {workspaceName}
                  {rule.enabled ? '' : ' · 已停用'}
                </div>
              </div>
              <div className={s.ruleHeaderActions}>
                <Switch
                  value={rule.enabled}
                  onChange={(e) => toggleEnabled(rule.id, e.currentTarget.checked)}
                />
                <Button
                  type="button"
                  variant="secondary"
                  fill="outline"
                  size="sm"
                  onClick={() => startEdit(rule)}
                  disabled={Boolean(draft)}
                  data-testid={testIds.appConfig.editCaptureRule}
                >
                  编辑规则
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  fill="outline"
                  size="sm"
                  onClick={() => removeRule(rule.id)}
                  data-testid={testIds.appConfig.removeCaptureRule}
                >
                  删除规则
                </Button>
              </div>
            </div>
          </div>
        );
      })}

      {draft && editingRule ? (
        <div className={s.editorCard} data-testid={testIds.appConfig.captureRuleEditor}>
          <div className={s.ruleHeader}>
            <strong>{draft.sourceId ? '编辑规则' : '新建规则'}</strong>
          </div>

          <Field label="规则名称">
            <Input
              width={40}
              value={editingRule.name}
              data-testid={testIds.appConfig.captureRuleName}
              onChange={(e) => patchDraft({ name: e.target.value })}
            />
          </Field>

          <Field label="匹配模式" className={s.fieldGap}>
            <Select
              inputId={`capture-rule-match-${editingRule.id}`}
              options={MATCH_MODE_OPTIONS}
              value={editingRule.matchMode}
              onChange={(v) =>
                patchDraft({ matchMode: (v.value ?? 'error_default') as CaptureMatchMode })
              }
            />
          </Field>

          {editingRule.matchMode === 'custom_loki' ? (
            <Field label="Loki 查询 (LogQL)" className={s.fieldGap}>
              <Input
                width={70}
                value={editingRule.lokiQuery ?? ''}
                placeholder='{job="runall"} |= "panic"'
                onChange={(e) => patchDraft({ lokiQuery: e.target.value })}
              />
            </Field>
          ) : null}

          {editingRule.matchMode === 'custom_regex' ? (
            <Field label="日志消息正则" className={s.fieldGap}>
              <Input
                width={50}
                value={editingRule.messageRegex ?? ''}
                placeholder="connection refused|timeout"
                onChange={(e) => patchDraft({ messageRegex: e.target.value })}
              />
            </Field>
          ) : null}

          <Field label="公司名称" description="先选择公司，再选择该公司下的工作空间" className={s.fieldGap}>
            <div data-testid={testIds.appConfig.captureRuleCompany}>
              <Select
                inputId={`capture-rule-company-${editingRule.id}`}
                options={companyOptions}
                value={companyOptions.find((option) => option.value === editingRule.tenantId) ?? null}
                placeholder={workspacesLoading ? '加载中…' : '选择公司'}
                disabled={!sessionToken || workspacesLoading || companyOptions.length === 0}
                onChange={(v) => onCompanySelect(String(v.value ?? ''))}
              />
            </div>
          </Field>

          <Field label="工作空间" className={s.fieldGap}>
            <Select
              inputId={`capture-rule-workspace-${editingRule.id}`}
              options={workspaceOptions}
              value={editingRule.workspaceId || null}
              placeholder={
                !editingRule.tenantId
                  ? '请先选择公司'
                  : workspaceOptions.length === 0
                    ? '该公司下暂无工作空间'
                    : '选择工作空间'
              }
              disabled={
                !sessionToken ||
                workspacesLoading ||
                !editingRule.tenantId ||
                workspaceOptions.length === 0
              }
              onChange={(v) => onWorkspaceSelect(String(v.value ?? ''))}
            />
          </Field>

          <Field
            label="责任人"
            description="仅列出可访问该工作空间的成员（创建者、编辑、可查看）"
            className={s.fieldGap}
          >
            <Select
              inputId={`capture-rule-owner-${editingRule.id}`}
              options={memberOptions}
              value={editingRule.ownerMemberId || null}
              placeholder={
                !editingRule.workspaceId
                  ? '请先选择工作空间'
                  : membersLoading[membersCacheKey]
                    ? '加载成员…'
                    : memberOptions.length === 0
                      ? '该工作空间暂无可选成员'
                      : '选择责任人'
              }
              disabled={
                !editingRule.tenantId ||
                !editingRule.workspaceId ||
                membersLoading[membersCacheKey] ||
                memberOptions.length === 0
              }
              onChange={(v) => patchDraft({ ownerMemberId: String(v.value ?? '') })}
            />
          </Field>

          <Field label="项目匹配" description="按名称、描述、标签匹配关联项目" className={s.fieldGap}>
            <ProjectMatchEditor
              logFieldRegex={editingRule.logFieldRegex}
              sampleLogLine={editingRule.sampleLogLine}
              projectMatchPattern={editingRule.projectMatchPattern}
              tenantId={editingRule.tenantId}
              workspaceId={editingRule.workspaceId}
              apiBaseUrl={apiBaseUrl}
              sessionToken={sessionToken}
              onChange={(patch) => patchDraft(patch)}
            />
          </Field>

          <Field label="工作分支匹配正则" description="从日志提取分支名，首捕获组优先" className={s.fieldGap}>
            <Input
              width={50}
              value={editingRule.branchMatchRegex}
              placeholder="branch[=:](\S+)"
              onChange={(e) => patchDraft({ branchMatchRegex: e.target.value })}
            />
          </Field>

          <Field label="合并目标分支" className={s.fieldGap}>
            <Input
              width={30}
              value={editingRule.mergeTargetBranch}
              onChange={(e) => patchDraft({ mergeTargetBranch: e.target.value })}
            />
          </Field>

          <div className={s.editorActions}>
            <Button type="button" variant="primary" onClick={saveDraft} data-testid={testIds.appConfig.saveCaptureRule}>
              保存规则
            </Button>
            <Button
              type="button"
              variant="secondary"
              fill="outline"
              onClick={cancelEdit}
              data-testid={testIds.appConfig.cancelCaptureRuleEdit}
            >
              取消
            </Button>
          </div>
        </div>
      ) : null}

      {!draft ? (
        <Button
          type="button"
          variant="primary"
          className={s.fieldGap}
          onClick={startAdd}
          data-testid={testIds.appConfig.addCaptureRule}
        >
          添加捕获规则
        </Button>
      ) : null}
    </FieldSet>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  help: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
    margin-bottom: ${theme.spacing(2)};
  `,
  hint: css`
    color: ${theme.colors.text.secondary};
    margin-bottom: ${theme.spacing(2)};
  `,
  warn: css`
    color: ${theme.colors.warning.text};
    margin-bottom: ${theme.spacing(2)};
  `,
  ruleCard: css`
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(2)};
  `,
  editorCard: css`
    border: 1px solid ${theme.colors.border.medium};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(2)};
    background: ${theme.colors.background.secondary};
  `,
  ruleHeader: css`
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: ${theme.spacing(1)};
    gap: ${theme.spacing(1)};
  `,
  ruleHeaderActions: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    flex-shrink: 0;
  `,
  summaryMeta: css`
    margin-top: ${theme.spacing(0.5)};
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  fieldGap: css`
    margin-top: ${theme.spacing(1.5)};
  `,
  editorActions: css`
    display: flex;
    gap: ${theme.spacing(1)};
    margin-top: ${theme.spacing(2)};
  `,
});
