import React, { ChangeEvent, useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Button, Field, FieldSet, Input, useStyles2 } from '@grafana/ui';
import { loginWithAccessToken } from '../../api/task2appClient';
import { testIds } from '../testIds';

export type LoginSuccessPayload = {
  sessionToken: string;
  apiBaseUrl: string;
  account: string;
  userId: string;
};

export type LoginSectionProps = {
  apiBaseUrl: string;
  account: string;
  accessToken: string;
  sessionToken: string;
  onChange: (patch: {
    apiBaseUrl?: string;
    account?: string;
    accessToken?: string;
    sessionToken?: string;
    userId?: string;
  }) => void;
  onLoginSuccess?: (payload: LoginSuccessPayload) => void | Promise<void>;
  onLogout?: () => void | Promise<void>;
};

export function LoginSection({
  apiBaseUrl,
  account,
  accessToken,
  sessionToken,
  onChange,
  onLoginSuccess,
  onLogout,
}: LoginSectionProps) {
  const s = useStyles2(getStyles);
  const [status, setStatus] = useState('');
  const [statusTraceId, setStatusTraceId] = useState('');
  const [testing, setTesting] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const isLoggedIn = Boolean(sessionToken.trim());

  const onFieldChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { name, value } = event.target;
    if (name === 'apiBaseUrl') {
      onChange({ apiBaseUrl: value });
    } else if (name === 'account') {
      onChange({ account: value });
    } else if (name === 'accessToken') {
      onChange({ accessToken: value });
    }
  };

  const onVerifyLogin = async () => {
    setTesting(true);
    setStatus('');
    setStatusTraceId('');
    try {
      const result = await loginWithAccessToken(apiBaseUrl, account, accessToken);
      const token = result.token?.trim();
      if (!token) {
        throw new Error(result.error || '登录成功但未返回 token');
      }
      const userId = String(result.user?.id || result.user_id || result.userId || '').trim();
      if (!userId) {
        throw new Error('登录成功但未返回 userId，请检查 task2app 登录 enrich 响应');
      }
      onChange({ sessionToken: token, userId });
      if (onLoginSuccess) {
        await onLoginSuccess({ sessionToken: token, apiBaseUrl, account, userId });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      const tid =
        err && typeof err === 'object' && 'traceId' in err && typeof (err as { traceId?: unknown }).traceId === 'string'
          ? String((err as { traceId: string }).traceId).trim()
          : '';
      const isNetworkFetchFailure =
        /failed to fetch/i.test(message) || message === 'NetworkError when attempting to fetch resource.';
      const grafanaOrigin = typeof window !== 'undefined' ? window.location.origin : '';
      const gatewayHint =
        typeof window !== 'undefined' && window.location.hostname
          ? `http://${window.location.hostname}:18081`
          : 'http://${INFRA_HOST}:18081';
      const hint = isNetworkFetchFailure
        ? `（多为 CORS：请确认 API 网关 cors.allowedOrigins 含当前 Grafana Origin${grafanaOrigin ? `（${grafanaOrigin}）` : ''}；也可改用已放行的网关基地址 ${gatewayHint}）`
        : '';
      setStatus(`登录失败：${message}${hint}`);
      setStatusTraceId(tid);
    } finally {
      setTesting(false);
    }
  };

  const onLogoutClick = async () => {
    if (!onLogout) {
      onChange({ sessionToken: '', accessToken: '', userId: '' });
      return;
    }

    setLoggingOut(true);
    setStatus('');
    setStatusTraceId('');
    try {
      await onLogout();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      const tid =
        err && typeof err === 'object' && 'traceId' in err && typeof (err as { traceId?: unknown }).traceId === 'string'
          ? String((err as { traceId: string }).traceId).trim()
          : '';
      setStatus(`退出失败：${message}`);
      setStatusTraceId(tid);
    } finally {
      setLoggingOut(false);
    }
  };

  if (isLoggedIn) {
    return (
      <FieldSet label="task2app 登录">
        <div className={s.loggedInPanel} data-testid={testIds.appConfig.sessionConfigured}>
          <p className={s.loggedInAccount} data-testid={testIds.appConfig.loggedInAccount}>
            账号：{account.trim() || '—'}
          </p>
          <p className={s.loggedInStatus}>已登录</p>
          <Button
            type="button"
            variant="secondary"
            data-testid={testIds.appConfig.logout}
            disabled={loggingOut}
            onClick={onLogoutClick}
          >
            {loggingOut ? '退出中…' : '退出'}
          </Button>
        </div>
      </FieldSet>
    );
  }

  return (
    <FieldSet label="task2app 登录">
      <Field label="API 基地址" description="task2app API 网关根 URL，如 http://183.250.1.132:18081（勿填 :4000 前端站点）">
        <Input
          width={60}
          name="apiBaseUrl"
          data-testid={testIds.appConfig.apiBaseUrl}
          value={apiBaseUrl}
          placeholder="http://183.250.1.132:18081"
          onChange={onFieldChange}
        />
      </Field>

      <Field label="账号" description="用户名或邮箱，与访问令牌所属用户一致" className={s.marginTop}>
        <Input
          width={40}
          name="account"
          data-testid={testIds.appConfig.account}
          value={account}
          placeholder="username"
          onChange={onFieldChange}
        />
      </Field>

      <Field
        label="访问令牌"
        description="个人资料中生成的 at_ 开头访问令牌；验证后换取 Session Token"
        className={s.marginTop}
      >
        <Input
          width={60}
          type="password"
          name="accessToken"
          data-testid={testIds.appConfig.accessToken}
          value={accessToken}
          placeholder="at_..."
          onChange={onFieldChange}
        />
      </Field>

      <div className={s.marginTop}>
        <Button type="button" onClick={onVerifyLogin} disabled={testing || !apiBaseUrl.trim() || !account.trim() || !accessToken.trim()}>
          {testing ? '登录中…' : '登录'}
        </Button>
        {status ? (
          <span className={s.status} {...(statusTraceId ? { 'data-traceId': statusTraceId } : {})}>
            {status}
          </span>
        ) : null}
      </div>
    </FieldSet>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  marginTop: css`
    margin-top: ${theme.spacing(2)};
  `,
  loggedInPanel: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1.5)};
  `,
  loggedInAccount: css`
    margin: 0;
    font-size: ${theme.typography.body.fontSize};
  `,
  loggedInStatus: css`
    margin: 0;
    color: ${theme.colors.success.text};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  status: css`
    margin-left: ${theme.spacing(2)};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
});
