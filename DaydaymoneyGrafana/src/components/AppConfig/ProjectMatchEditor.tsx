import React, { useEffect, useMemo, useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Field, Input, useStyles2 } from '@grafana/ui';
import { listProjects, type Task2appProject } from '../../api/task2appClient';
import { extractLogCaptures, projectMatchUsesCaptures, resolveMatchedProjects } from '../../errorReporting/projectMatcher';
import type { CaptureRule } from '../../types/settings';

export type ProjectMatchEditorProps = {
  logFieldRegex: string;
  sampleLogLine: string;
  projectMatchPattern: string;
  tenantId: string;
  workspaceId: string;
  apiBaseUrl: string;
  sessionToken: string;
  onChange: (patch: {
    logFieldRegex?: string;
    sampleLogLine?: string;
    projectMatchPattern?: string;
  }) => void;
};

export function formatLogCapturePreview(logFieldRegex: string, sampleLogLine: string): string {
  if (!sampleLogLine.trim()) {
    return '请先填写样例日志';
  }
  if (!logFieldRegex.trim()) {
    return '';
  }
  const captures = extractLogCaptures(logFieldRegex, sampleLogLine);
  if (captures.length === 0) {
    return '未从样例日志中匹配到捕获组';
  }
  return captures.map((value, index) => `$${index + 1} = ${value}`).join('，');
}

export function isLogCapturePreviewFailure(message: string): boolean {
  return message === '未从样例日志中匹配到捕获组';
}

export type ProjectMatchPreviewState =
  | { kind: 'idle'; message: string }
  | { kind: 'loading'; message: string }
  | { kind: 'error'; message: string }
  | { kind: 'failure'; message: string }
  | { kind: 'success'; projects: Task2appProject[] };

export function getProjectMatchPreviewState(params: {
  tenantId: string;
  workspaceId: string;
  projectsLoading: boolean;
  projectsError: string;
  sampleLogLine: string;
  logFieldRegex: string;
  projectMatchPattern: string;
  projects: Task2appProject[];
  matchedProjects: Task2appProject[];
}): ProjectMatchPreviewState {
  const {
    tenantId,
    workspaceId,
    projectsLoading,
    projectsError,
    sampleLogLine,
    logFieldRegex,
    projectMatchPattern,
    projects,
    matchedProjects,
  } = params;

  if (!tenantId || !workspaceId) {
    return { kind: 'idle', message: '请先选择公司与工作空间' };
  }
  if (projectsLoading) {
    return { kind: 'loading', message: '正在加载项目列表…' };
  }
  if (projectsError) {
    return { kind: 'error', message: `加载项目失败：${projectsError}` };
  }

  const needsLogLine = projectMatchUsesCaptures(projectMatchPattern);
  if (!sampleLogLine.trim() && needsLogLine) {
    return { kind: 'idle', message: '项目匹配引用了 $1 等捕获组，请先填写样例日志' };
  }

  if (needsLogLine && logFieldRegex.trim() && sampleLogLine.trim()) {
    if (extractLogCaptures(logFieldRegex, sampleLogLine).length === 0) {
      return {
        kind: 'failure',
        message: '项目匹配引用了 $1 等捕获组，但日志字段捕获未成功，无法匹配项目',
      };
    }
  }

  if (projects.length === 0) {
    return { kind: 'failure', message: '该工作空间下暂无项目' };
  }

  if (matchedProjects.length === 0) {
    return {
      kind: 'failure',
      message: `未匹配到任何项目（共 ${projects.length} 个），请检查项目匹配表达式`,
    };
  }

  return { kind: 'success', projects: matchedProjects };
}

export function ProjectMatchEditor({
  logFieldRegex,
  sampleLogLine,
  projectMatchPattern,
  tenantId,
  workspaceId,
  apiBaseUrl,
  sessionToken,
  onChange,
}: ProjectMatchEditorProps) {
  const s = useStyles2(getStyles);
  const [projects, setProjects] = useState<Task2appProject[]>([]);
  const [projectsLoading, setProjectsLoading] = useState(false);
  const [projectsError, setProjectsError] = useState('');

  useEffect(() => {
    if (!tenantId || !workspaceId || !sessionToken || !apiBaseUrl.trim()) {
      setProjects([]);
      setProjectsError('');
      return;
    }

    let cancelled = false;
    setProjectsLoading(true);
    setProjectsError('');

    void listProjects(apiBaseUrl, sessionToken, tenantId, workspaceId)
      .then((items) => {
        if (!cancelled) {
          setProjects(items);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setProjects([]);
          setProjectsError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setProjectsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl, sessionToken, tenantId, workspaceId]);

  const previewRule = useMemo(
    (): Pick<CaptureRule, 'logFieldRegex' | 'projectMatchPattern' | 'sampleLogLine'> => ({
      logFieldRegex,
      sampleLogLine,
      projectMatchPattern,
    }),
    [logFieldRegex, sampleLogLine, projectMatchPattern]
  );

  const matchedProjects = useMemo(() => {
    if (!tenantId || !workspaceId || projectsLoading || projectsError) {
      return [];
    }
    const logLine = sampleLogLine.trim();
    if (!logLine && projectMatchUsesCaptures(projectMatchPattern)) {
      return [];
    }
    return resolveMatchedProjects(previewRule, logLine, projects);
  }, [previewRule, projects, sampleLogLine, tenantId, workspaceId, projectsLoading, projectsError, projectMatchPattern]);

  const capturePreview = useMemo(
    () => formatLogCapturePreview(logFieldRegex, sampleLogLine),
    [logFieldRegex, sampleLogLine]
  );

  const projectPreview = useMemo(
    () =>
      getProjectMatchPreviewState({
        tenantId,
        workspaceId,
        projectsLoading,
        projectsError,
        sampleLogLine,
        logFieldRegex,
        projectMatchPattern,
        projects,
        matchedProjects,
      }),
    [
      tenantId,
      workspaceId,
      projectsLoading,
      projectsError,
      sampleLogLine,
      logFieldRegex,
      projectMatchPattern,
      projects,
      matchedProjects,
    ]
  );

  return (
    <div className={s.root}>
      <Field label="样例日志" description="用于预览捕获与项目匹配；运行时对真实告警日志执行相同规则">
        <Input
          width={70}
          value={sampleLogLine}
          placeholder='level=error service=saas-backend msg=timeout'
          onChange={(e) => onChange({ sampleLogLine: e.currentTarget.value })}
        />
      </Field>

      <Field
        label="日志字段捕获正则"
        description="从样例日志提取字段，捕获组可用 $1、$2 在项目匹配中引用；正则按原样执行，不做自动改写"
        className={s.fieldGap}
      >
        <div>
          <Input
            width={50}
            value={logFieldRegex}
            placeholder='"service":"([^"]+)"'
            onChange={(e) => onChange({ logFieldRegex: e.currentTarget.value })}
          />
          {capturePreview ? (
            <p className={isLogCapturePreviewFailure(capturePreview) ? s.error : s.capturePreview}>
              捕获结果：{capturePreview}
            </p>
          ) : null}
        </div>
      </Field>

      <Field
        label="项目匹配表达式"
        description="对项目名称、描述、标签做正则匹配；可引用 $1、$2 等捕获组"
        className={s.fieldGap}
      >
        <Input
          width={50}
          value={projectMatchPattern}
          placeholder="$1 或 api|backend"
          onChange={(e) => onChange({ projectMatchPattern: e.currentTarget.value })}
        />
      </Field>

      <div className={s.preview}>
        <strong className={s.previewTitle}>匹配项目预览</strong>
        {projectPreview.kind === 'success' ? (
          <ul className={s.projectList}>
            {projectPreview.projects.map((project) => (
              <li key={project.id} className={s.projectItem}>
                <div className={s.projectName}>{project.name}</div>
                {project.description ? <div className={s.projectDesc}>{project.description}</div> : null}
                {project.tags?.length ? (
                  <div className={s.projectTags}>{project.tags.join(' · ')}</div>
                ) : null}
              </li>
            ))}
          </ul>
        ) : (
          <p
            className={
              projectPreview.kind === 'failure' || projectPreview.kind === 'error' ? s.error : s.hint
            }
          >
            {projectPreview.message}
          </p>
        )}
      </div>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  root: css`
    margin-top: ${theme.spacing(0.5)};
  `,
  fieldGap: css`
    margin-top: ${theme.spacing(1.5)};
  `,
  capturePreview: css`
    margin: ${theme.spacing(0.75)} 0 0;
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  preview: css`
    margin-top: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.secondary};
  `,
  previewTitle: css`
    display: block;
    margin-bottom: ${theme.spacing(1)};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  hint: css`
    margin: 0;
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  error: css`
    margin: 0;
    color: ${theme.colors.error.text};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  warn: css`
    margin: 0;
    color: ${theme.colors.warning.text};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  projectList: css`
    margin: 0;
    padding-left: ${theme.spacing(2)};
  `,
  projectItem: css`
    margin-bottom: ${theme.spacing(1)};

    &:last-child {
      margin-bottom: 0;
    }
  `,
  projectName: css`
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  projectDesc: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
    margin-top: ${theme.spacing(0.25)};
  `,
  projectTags: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
    margin-top: ${theme.spacing(0.25)};
  `,
});
