import React, { ChangeEvent, useCallback, useEffect, useRef, useState } from 'react';
import { lastValueFrom } from 'rxjs';
import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Button, Field, FieldSet, Input, Switch, useStyles2 } from '@grafana/ui';
import { listAllWorkspaces, type Task2appWorkspace } from '../../api/task2appClient';
import type { AppPluginSettings, CaptureRule } from '../../types/settings';
import { defaultCaptureRule, normalizeCaptureRule } from '../../types/settings';
import { CaptureRulesEditor } from './CaptureRulesEditor';
import { LoginSection } from './LoginSection';
import { testIds } from '../testIds';

export type { AppPluginSettings };

type State = {
  task2appApiBaseUrl: string;
  task2appAccount: string;
  task2appUserId: string;
  task2appAccessToken: string;
  task2appSessionToken: string;
  captureRules: CaptureRule[];
  lokiPollEnabled: boolean;
  lokiDatasourceUid: string;
};

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> {}

function initialRules(jsonData?: AppPluginSettings): CaptureRule[] {
  if (jsonData?.captureRules?.length) {
    return jsonData.captureRules.map((rule) => normalizeCaptureRule(rule));
  }
  return [defaultCaptureRule()];
}

function buildPluginJsonData(state: State): AppPluginSettings {
  return {
    task2appApiBaseUrl: state.task2appApiBaseUrl.trim(),
    task2appAccount: state.task2appAccount.trim(),
    task2appUserId: state.task2appUserId.trim(),
    task2appSessionToken: state.task2appSessionToken.trim(),
    captureRules: state.captureRules,
    lokiPollEnabled: state.lokiPollEnabled,
    lokiDatasourceUid: state.lokiDatasourceUid.trim() || 'loki',
  };
}

function mergeSavedJsonData(prev: State, jsonData: AppPluginSettings): State {
  return {
    ...prev,
    task2appApiBaseUrl: jsonData.task2appApiBaseUrl?.trim() || prev.task2appApiBaseUrl,
    task2appAccount: jsonData.task2appAccount?.trim() || prev.task2appAccount,
    task2appUserId: jsonData.task2appUserId?.trim() || prev.task2appUserId,
    task2appSessionToken: jsonData.task2appSessionToken?.trim() || prev.task2appSessionToken,
    captureRules: jsonData.captureRules?.length ? jsonData.captureRules.map((rule) => normalizeCaptureRule(rule)) : prev.captureRules,
    lokiPollEnabled: jsonData.lokiPollEnabled ?? prev.lokiPollEnabled,
    lokiDatasourceUid: jsonData.lokiDatasourceUid?.trim() || prev.lokiDatasourceUid,
  };
}

const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, jsonData } = plugin.meta;
  const [state, setState] = useState<State>({
    task2appApiBaseUrl: jsonData?.task2appApiBaseUrl || 'http://183.250.1.132:18081',
    task2appAccount: jsonData?.task2appAccount || '',
    task2appUserId: jsonData?.task2appUserId || '',
    task2appAccessToken: '',
    task2appSessionToken: jsonData?.task2appSessionToken || '',
    captureRules: initialRules(jsonData),
    lokiPollEnabled: jsonData?.lokiPollEnabled ?? true,
    lokiDatasourceUid: jsonData?.lokiDatasourceUid || 'loki',
  });
  const [workspaces, setWorkspaces] = useState<Task2appWorkspace[]>([]);
  const [workspacesLoading, setWorkspacesLoading] = useState(false);
  const stateRef = useRef(state);
  stateRef.current = state;

  const fetchWorkspaces = useCallback(
    async (sessionToken: string, userId: string) => {
      const baseUrl = state.task2appApiBaseUrl.trim();
      const token = sessionToken.trim();
      const uid = userId.trim();
      if (!baseUrl || !token || !uid) {
        setWorkspaces([]);
        return;
      }
      setWorkspacesLoading(true);
      try {
        const items = await listAllWorkspaces(baseUrl, token, uid);
        setWorkspaces(items);
      } catch (err) {
        console.error('[DaydaymoneyGrafana] 拉取工作空间失败', err);
        setWorkspaces([]);
      } finally {
        setWorkspacesLoading(false);
      }
    },
    [state.task2appApiBaseUrl]
  );

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const saved = await fetchPluginJsonData(plugin.meta.id);
        if (cancelled || !saved) {
          return;
        }
        setState((prev) => mergeSavedJsonData(prev, saved));
      } catch (err) {
        console.warn('[DaydaymoneyGrafana] 读取已保存配置失败', err);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [plugin.meta.id]);

  useEffect(() => {
    if (state.task2appSessionToken.trim() && state.task2appUserId.trim()) {
      void fetchWorkspaces(state.task2appSessionToken, state.task2appUserId);
    }
  }, [state.task2appSessionToken, state.task2appUserId, fetchWorkspaces]);

  const hasValidLogin = Boolean(state.task2appApiBaseUrl.trim() && state.task2appSessionToken.trim());
  const needsUserIdRelogin = Boolean(state.task2appSessionToken.trim() && !state.task2appUserId.trim());
  const hasValidRule = state.captureRules.some(
    (rule) => rule.enabled && rule.tenantId.trim() && rule.workspaceId.trim() && rule.ownerMemberId.trim()
  );
  const isSubmitDisabled = !hasValidLogin;

  const onLokiChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { name, value } = event.target;
    setState((prev) => ({ ...prev, [name]: value.trim() }));
  };

  const onLoginChange = (patch: {
    apiBaseUrl?: string;
    account?: string;
    userId?: string;
    accessToken?: string;
    sessionToken?: string;
  }) => {
    setState((prev) => ({
      ...prev,
      ...(patch.apiBaseUrl !== undefined ? { task2appApiBaseUrl: patch.apiBaseUrl } : {}),
      ...(patch.account !== undefined ? { task2appAccount: patch.account } : {}),
      ...(patch.userId !== undefined ? { task2appUserId: patch.userId } : {}),
      ...(patch.accessToken !== undefined ? { task2appAccessToken: patch.accessToken } : {}),
      ...(patch.sessionToken !== undefined ? { task2appSessionToken: patch.sessionToken } : {}),
    }));
  };

  const handleLoginSuccess = useCallback(
    async ({
      sessionToken,
      apiBaseUrl,
      account,
      userId,
    }: {
      sessionToken: string;
      apiBaseUrl: string;
      account: string;
      userId: string;
    }) => {
      const snapshot: State = {
        ...stateRef.current,
        task2appApiBaseUrl: apiBaseUrl,
        task2appAccount: account,
        task2appUserId: userId,
        task2appSessionToken: sessionToken,
      };

      setState(snapshot);

      await updatePlugin(plugin.meta.id, {
        enabled,
        pinned,
        jsonData: buildPluginJsonData(snapshot),
      });

      await fetchWorkspaces(sessionToken, userId);
    },
    [enabled, fetchWorkspaces, pinned, plugin.meta.id]
  );

  const handleLogout = useCallback(async () => {
    const snapshot: State = {
      ...stateRef.current,
      task2appAccessToken: '',
      task2appSessionToken: '',
      task2appUserId: '',
    };

    setState(snapshot);
    setWorkspaces([]);

    await updatePlugin(plugin.meta.id, {
      enabled,
      pinned,
      jsonData: buildPluginJsonData(snapshot),
    });
  }, [enabled, pinned, plugin.meta.id]);

  const onSubmit = () => {
    if (isSubmitDisabled) {
      return;
    }

    updatePluginAndReload(plugin.meta.id, {
      enabled,
      pinned,
      jsonData: buildPluginJsonData(state),
    });
  };

  return (
    <div>
      <LoginSection
        apiBaseUrl={state.task2appApiBaseUrl}
        account={state.task2appAccount}
        accessToken={state.task2appAccessToken}
        sessionToken={state.task2appSessionToken}
        onChange={onLoginChange}
        onLoginSuccess={handleLoginSuccess}
        onLogout={handleLogout}
      />

      {needsUserIdRelogin ? (
        <p className={s.warn} data-testid={testIds.appConfig.reloginForUserId}>
          会话已保存但缺少用户 ID，无法加载工作空间列表。请填写访问令牌后重新登录以完成配置迁移。
        </p>
      ) : null}

      <div className={s.sectionGap}>
        <CaptureRulesEditor
          rules={state.captureRules}
          onChange={(captureRules) => setState((prev) => ({ ...prev, captureRules }))}
          apiBaseUrl={state.task2appApiBaseUrl}
          sessionToken={state.task2appSessionToken}
          workspaces={workspaces}
          workspacesLoading={workspacesLoading}
        />
      </div>

      <FieldSet label="Loki 轮询" className={s.sectionGap}>
        <Field label="启用 Loki 轮询" description="每 30 秒查询 Loki 并按捕获规则建任务">
          <Switch
            id="config-loki-poll-enabled"
            data-testid={testIds.appConfig.lokiPollEnabled}
            value={state.lokiPollEnabled}
            onChange={(e) => setState((prev) => ({ ...prev, lokiPollEnabled: e.currentTarget.checked }))}
          />
        </Field>

        <Field label="Loki 数据源 UID" className={s.marginTop}>
          <Input
            width={30}
            name="lokiDatasourceUid"
            data-testid={testIds.appConfig.lokiDatasourceUid}
            value={state.lokiDatasourceUid}
            placeholder="loki"
            onChange={onLokiChange}
          />
        </Field>
      </FieldSet>

      {!hasValidRule && hasValidLogin ? (
        <p className={s.warn}>请至少配置一条已启用的捕获规则（含租户、工作空间、责任人）。</p>
      ) : null}

      <div className={s.marginTop}>
        <Button type="button" data-testid={testIds.appConfig.submit} disabled={isSubmitDisabled} onClick={onSubmit}>
          保存配置
        </Button>
      </div>
    </div>
  );
};

export default AppConfig;

const getStyles = (theme: GrafanaTheme2) => ({
  marginTop: css`
    margin-top: ${theme.spacing(3)};
  `,
  sectionGap: css`
    margin-top: ${theme.spacing(4)};
  `,
  warn: css`
    color: ${theme.colors.warning.text};
    margin-top: ${theme.spacing(2)};
  `,
});

const updatePluginAndReload = async (pluginId: string, data: Partial<PluginMeta<AppPluginSettings>>) => {
  try {
    await updatePlugin(pluginId, data);
    window.location.reload();
  } catch (err) {
    console.error('Error while updating the plugin', err);
  }
};

const updatePlugin = async (pluginId: string, data: Partial<PluginMeta>) => {
  const response = await getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/settings`,
    method: 'POST',
    data,
  });

  return lastValueFrom(response);
};

const fetchPluginJsonData = async (pluginId: string): Promise<AppPluginSettings | undefined> => {
  const response = await getBackendSrv().fetch<{ jsonData?: AppPluginSettings }>({
    url: `/api/plugins/${pluginId}/settings`,
    method: 'GET',
  });
  const payload = await lastValueFrom(response);
  return payload.data.jsonData;
};
