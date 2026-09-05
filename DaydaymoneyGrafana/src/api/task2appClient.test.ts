import {
  fetchCurrentUser,
  listAllWorkspaces,
  listMembers,
  listWorkspaceCollaborators,
  resolveWorkspaceCompanyName,
} from './task2appClient';

describe('task2appClient workspace normalization', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  test('listAllWorkspaces aggregates tenant-scoped workspace lists', async () => {
    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: '850256677331562496',
          companies: [{ id: '850256677331562496', name: 'example-user' }],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [
          {
            id: '857903329669984256',
            name: '默认工作空间',
            company_name: 'example-user',
            company_id: '850256677331562496',
          },
        ],
      });

    const workspaces = await listAllWorkspaces('http://example.test', 'session-token', '850256677331562496');

    expect(global.fetch).toHaveBeenNthCalledWith(
      1,
      'http://example.test/api/user/850256677331562496/accounts/users/me/',
      expect.objectContaining({ method: 'GET' })
    );
    expect(global.fetch).toHaveBeenNthCalledWith(
      2,
      'http://example.test/api/tenant/850256677331562496/workspaces/',
      expect.objectContaining({ method: 'GET' })
    );
    expect(workspaces).toEqual([
      {
        id: '857903329669984256',
        name: '默认工作空间',
        company_id: '850256677331562496',
        company_name: 'example-user',
      },
    ]);
  });

  test('listAllWorkspaces prefers /me company name when workspace company_name is the id', async () => {
    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: 'u1',
          companies: [{ id: '850256677331562496', name: 'example-user' }],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [
          {
            id: '857903329669984256',
            name: '默认工作空间',
            // taskProjectService historically echoed company_id into company_name
            company_name: '850256677331562496',
            company_id: '850256677331562496',
          },
        ],
      });

    const workspaces = await listAllWorkspaces('http://example.test', 'session-token', 'u1');

    expect(workspaces).toEqual([
      {
        id: '857903329669984256',
        name: '默认工作空间',
        company_id: '850256677331562496',
        company_name: 'example-user',
      },
    ]);
  });

  test('resolveWorkspaceCompanyName prefers /me name over id-echoed company_name', () => {
    expect(
      resolveWorkspaceCompanyName(
        { id: '850256677331562496', name: 'example-user' },
        { company_id: '850256677331562496', company_name: '850256677331562496' }
      )
    ).toBe('example-user');
    expect(
      resolveWorkspaceCompanyName(
        { id: 'c1', name: '' },
        { company_id: 'c1', company_name: 'Acme' }
      )
    ).toBe('Acme');
  });

  test('fetchCurrentUser normalizes companies from me endpoint', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => ({
        id: '850256677331562496',
        companies: [{ id: 42, name: 'Alpha 科技' }],
      }),
    });

    const user = await fetchCurrentUser('http://example.test', 'session-token', '850256677331562496');

    expect(user).toEqual({
      id: '850256677331562496',
      companies: [{ id: '42', name: 'Alpha 科技' }],
    });
  });
});

describe('task2appClient member normalization', () => {
  beforeEach(() => {
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  test('listMembers maps member_name from company_members API', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => ({
        members: [
          {
            id: '850256677331562496',
            company_member_id: '850256677331562496',
            member_name: '张三',
            email: 'zhang@example.com',
          },
        ],
      }),
    });

    const members = await listMembers('http://example.test', 'session-token', 'tenant-1');

    expect(members).toEqual([
      {
        id: '850256677331562496',
        user_id: undefined,
        username: undefined,
        member_name: '张三',
        email: 'zhang@example.com',
        display_name: '张三',
      },
    ]);
  });

  test('listMembers falls back to email when member_name is missing', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => ({
        members: [
          {
            id: '123',
            email: 'dev@example.com',
          },
        ],
      }),
    });

    const members = await listMembers('http://example.test', 'session-token', 'tenant-1');

    expect(members[0]?.display_name).toBe('dev@example.com');
  });

  test('listWorkspaceCollaborators maps company member array from workspace access API', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => [
        {
          id: '850256677331562496',
          user: '123456789',
          member_name: '研发负责人',
          username: '研发负责人',
        },
      ],
    });

    const members = await listWorkspaceCollaborators('http://example.test', 'session-token', 'tenant-1', 'ws-9');

    expect(global.fetch).toHaveBeenCalledWith(
      'http://example.test/api/tenant/tenant-1/projects/workspace-access/workspace-collaborators/?workspace_id=ws-9',
      expect.objectContaining({ method: 'GET' })
    );
    expect(members[0]?.display_name).toBe('研发负责人');
  });

  test('listProjects preserves description and tags', async () => {
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => [
        {
          id: 'p1',
          name: 'saas-backend',
          description: '后端服务',
          tags: ['api', 'django'],
        },
      ],
    });

    const { listProjects } = await import('./task2appClient');
    const projects = await listProjects('http://example.test', 'session-token', 'tenant-1', 'ws-1');

    expect(projects).toEqual([
      {
        id: 'p1',
        name: 'saas-backend',
        description: '后端服务',
        tags: ['api', 'django'],
      },
    ]);
  });
});
