import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { PluginType } from '@grafana/data';
import { of } from 'rxjs';
import AppConfig, { AppConfigProps } from './AppConfig';
import { testIds } from 'components/testIds';
import { getBackendSrv } from '@grafana/runtime';
import { listAllWorkspaces, loginWithAccessToken } from '../../api/task2appClient';

jest.mock('@grafana/runtime', () => ({
  getBackendSrv: jest.fn(),
}));

jest.mock('../../api/task2appClient', () => ({
  ...jest.requireActual('../../api/task2appClient'),
  loginWithAccessToken: jest.fn(),
  listAllWorkspaces: jest.fn(),
  listWorkspaceCollaborators: jest.fn().mockResolvedValue([]),
  listProjects: jest.fn().mockResolvedValue([]),
}));

const mockFetch = jest.fn();
const mockGetBackendSrv = getBackendSrv as jest.Mock;
const mockLoginWithAccessToken = loginWithAccessToken as jest.Mock;
const mockListAllWorkspaces = listAllWorkspaces as jest.Mock;

describe('Components/AppConfig', () => {
  let props: AppConfigProps;

  beforeEach(() => {
    jest.resetAllMocks();
    mockFetch.mockReturnValue(of({ data: {} }));
    mockGetBackendSrv.mockReturnValue({ fetch: mockFetch });
    mockLoginWithAccessToken.mockResolvedValue({
      token: 'session-token-abc',
      user: { id: '850256677331562496', companies: [{ id: '850256677331562496', name: 'demo-co' }] },
    });
    mockListAllWorkspaces.mockResolvedValue([]);

    props = {
      plugin: {
        meta: {
          id: 'sample-app',
          name: 'Sample App',
          type: PluginType.app,
          enabled: true,
          jsonData: {},
        },
      },
      query: {},
    } as unknown as AppConfigProps;
  });

  test('renders login, capture rules and save button', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.getByTestId(testIds.appConfig.apiBaseUrl)).toBeInTheDocument();
    expect(screen.getByTestId(testIds.appConfig.account)).toBeInTheDocument();
    expect(screen.getByTestId(testIds.appConfig.addCaptureRule)).toBeInTheDocument();
    expect(screen.getByTestId(testIds.appConfig.lokiPollEnabled)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /登录/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /保存配置/i })).toBeInTheDocument();
    expect(screen.queryByText(/旧版回退/)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/grafana-errors URL/i)).not.toBeInTheDocument();
  });

  test('login section account and access token inputs accept typing', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    const accountInput = screen.getByTestId(testIds.appConfig.account);
    const accessTokenInput = screen.getByTestId(testIds.appConfig.accessToken);

    fireEvent.change(accountInput, { target: { name: 'account', value: 'demo-user' } });
    fireEvent.change(accessTokenInput, { target: { name: 'accessToken', value: 'at_demo_token' } });

    expect(accountInput).toHaveValue('demo-user');
    expect(accessTokenInput).toHaveValue('at_demo_token');
    expect(screen.getByRole('button', { name: /登录/i })).toBeEnabled();
  });

  test('persists login session to plugin settings after successful login', async () => {
    const plugin = { meta: { ...props.plugin.meta, id: 'sample-app', enabled: true } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    fireEvent.change(screen.getByTestId(testIds.appConfig.account), {
      target: { name: 'account', value: 'demo-user' },
    });
    fireEvent.change(screen.getByTestId(testIds.appConfig.accessToken), {
      target: { name: 'accessToken', value: 'at_demo_token' },
    });
    fireEvent.click(screen.getByRole('button', { name: /登录/i }));

    await waitFor(() => {
      expect(mockLoginWithAccessToken).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/sample-app/settings',
          method: 'POST',
          data: expect.objectContaining({
            jsonData: expect.objectContaining({
              task2appAccount: 'demo-user',
              task2appSessionToken: 'session-token-abc',
            }),
          }),
        })
      );
    });

    expect(await screen.findByText(/已登录/i)).toBeInTheDocument();
    expect(screen.getByTestId(testIds.appConfig.loggedInAccount)).toHaveTextContent('demo-user');
    expect(screen.queryByTestId(testIds.appConfig.accessToken)).not.toBeInTheDocument();
    expect(screen.getByTestId(testIds.appConfig.logout)).toBeInTheDocument();
  });

  test('hydrates login session from plugin settings API on mount', async () => {
    mockFetch.mockImplementation(({ method, url }: { method?: string; url?: string }) => {
      if (method === 'GET' && url?.includes('/settings')) {
        return of({
          data: {
            jsonData: {
              task2appAccount: 'saved-user@example.com',
              task2appSessionToken: 'saved-session-token',
            },
          },
        });
      }
      return of({ data: {} });
    });

    const plugin = { meta: { ...props.plugin.meta, id: 'sample-app', enabled: true, jsonData: {} } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(await screen.findByText(/已登录/i)).toBeInTheDocument();
    expect(screen.getByTestId(testIds.appConfig.loggedInAccount)).toHaveTextContent('saved-user@example.com');
  });

  test('shows relogin hint when session exists without user id', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        id: 'sample-app',
        enabled: true,
        jsonData: {
          task2appAccount: 'legacy-user',
          task2appSessionToken: 'legacy-session-token',
        },
      },
    };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(await screen.findByTestId(testIds.appConfig.reloginForUserId)).toHaveTextContent(/重新登录/);
    expect(mockListAllWorkspaces).not.toHaveBeenCalled();
  });

  test('logout clears session and shows login form again', async () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        id: 'sample-app',
        enabled: true,
        jsonData: {
          task2appAccount: 'demo-user',
          task2appSessionToken: 'session-token-abc',
        },
      },
    };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.getByText(/已登录/i)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId(testIds.appConfig.logout));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/plugins/sample-app/settings',
          method: 'POST',
          data: expect.objectContaining({
            jsonData: expect.objectContaining({
              task2appSessionToken: '',
            }),
          }),
        })
      );
    });

    expect(screen.getByTestId(testIds.appConfig.accessToken)).toBeInTheDocument();
    expect(screen.queryByText(/已登录/i)).not.toBeInTheDocument();
  });
});
