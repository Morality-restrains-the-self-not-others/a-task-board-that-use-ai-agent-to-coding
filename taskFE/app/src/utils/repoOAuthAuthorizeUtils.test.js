// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  GENERIC_REPO_BRANCHES_WITHOUT_CREDENTIALS_ERROR,
  resolveRepoOAuthAuthorizeLabel,
  resolveRepoOAuthProvider,
  resolveRepoOAuthProviderInfo,
  resolveRepoOAuthAuthorizeUrl,
  buildRepoOAuthStartHref,
  supportsRepoOAuthAuthorize,
  shouldShowRepoOAuthAuthorizeButton,
  setGitOAuthProviderCatalogForTests,
  loadGitOAuthProviderCatalog,
} from './repoOAuthAuthorizeUtils.js'

const TENCENT_CATALOG = [
  {
    provider: 'gitlab',
    service_provider: 'tencent-gitlab',
    provider_key: 'gitlab:tencent-gitlab',
    website: 'http://1.117.67.121:8012',
    website_origin: 'http://1.117.67.121:8012',
  },
]

describe('shouldShowRepoOAuthAuthorizeButton', () => {
  it('仅在 generic 仓库无凭据报错时显示按钮', () => {
    expect(
      shouldShowRepoOAuthAuthorizeButton(
        `${GENERIC_REPO_BRANCHES_WITHOUT_CREDENTIALS_ERROR} (repo: https://example.com/a/b.git)`,
      ),
    ).toBe(true)
    expect(shouldShowRepoOAuthAuthorizeButton('GitHub repository not found')).toBe(false)
  })

  it('GitLab OAuth 失效或会话无效时应显示按钮', () => {
    expect(
      shouldShowRepoOAuthAuthorizeButton(
        '无法获取 GitLab 分支：GitLab 会话无效或无权访问该仓库，请重新登录 GitLab 或完成 OAuth 授权',
      ),
    ).toBe(true)
    expect(
      shouldShowRepoOAuthAuthorizeButton(
        'GitLab OAuth 令牌已失效或 refresh 失败，请在项目页点击「OAuth 授权」重新绑定',
      ),
    ).toBe(true)
    expect(shouldShowRepoOAuthAuthorizeButton('git oauth not connected')).toBe(true)
    expect(shouldShowRepoOAuthAuthorizeButton('尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再合并')).toBe(true)
    expect(shouldShowRepoOAuthAuthorizeButton('无法获取 Git 访问令牌：gitlab refresh http 400')).toBe(true)
    expect(shouldShowRepoOAuthAuthorizeButton('Git OAuth 授权已失效，请重新绑定后再合并')).toBe(true)
    expect(
      shouldShowRepoOAuthAuthorizeButton(
        '无法获取 GitLab 分支：未检测到可用授权。请先在个人资料完成 GitLab 绑定，或先登录本地 GitLab（localhost）后重试。',
      ),
    ).toBe(true)
  })
})

describe('resolveRepoOAuthAuthorizeUrl', () => {
  it('支持的 provider 返回 task2app 授权入口', () => {
    expect(resolveRepoOAuthAuthorizeUrl('github')).toBe(
      '/api/git-oauth/github-start-from-gateway/',
    )
    expect(resolveRepoOAuthAuthorizeUrl('gitlab')).toBe(
      '/api/git-oauth/gitlab-start-from-gateway/',
    )
    expect(resolveRepoOAuthAuthorizeUrl('bitbucket')).toBe('')
  })

  it('未知 provider 不返回授权页', () => {
    expect(resolveRepoOAuthAuthorizeUrl('')).toBe('')
    expect(resolveRepoOAuthAuthorizeUrl('unknown')).toBe('')
  })
})

describe('buildRepoOAuthStartHref', () => {
  it('github 仓指向 github-start-from-gateway 并带 repo_url', () => {
    const href = buildRepoOAuthStartHref('https://github.com/acme/demo.git', {
      nextPath: '/tenant/t1/task-detail/',
    })
    expect(href).toContain('/api/git-oauth/github-start-from-gateway/')
    expect(href).toContain('repo_url=')
    expect(decodeURIComponent(href)).toContain('https://github.com/acme/demo.git')
    expect(href).toContain('next=')
    expect(href).toContain('grant_kind=pending')
    expect(href).not.toContain('git-site-oauth')
    expect(href).not.toContain('/user/')
  })

  it('gitlab 仓指向 gitlab-start-from-gateway', () => {
    const href = buildRepoOAuthStartHref('https://gitlab.daydaymoney.com/group/repo.git')
    expect(href).toContain('/api/git-oauth/gitlab-start-from-gateway/')
    expect(decodeURIComponent(href)).toContain('https://gitlab.daydaymoney.com/group/repo.git')
    expect(href).toContain('grant_kind=pending')
    expect(href).not.toContain('git-site-oauth')
  })

  it('catalog 命中区域 GitLab 时带 service_provider=tencent-sh-1', () => {
    setGitOAuthProviderCatalogForTests([
      {
        provider: 'gitlab',
        service_provider: 'tencent-sh-1',
        provider_key: 'gitlab:tencent-sh-1',
        website: 'https://gitlab-tencent-sh-1.daydaymoney.com',
        website_origin: 'https://gitlab-tencent-sh-1.daydaymoney.com',
      },
    ])
    try {
      const href = buildRepoOAuthStartHref(
        'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work',
      )
      expect(href).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(href).toContain('service_provider=tencent-sh-1')
      expect(decodeURIComponent(href)).toContain(
        'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work',
      )
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })

  it('不支持的仓返回空', () => {
    expect(buildRepoOAuthStartHref('https://bitbucket.org/team/repo')).toBe('')
  })

  it('Path A IP catalog 命中带 service_provider=tenant-…', () => {
    setGitOAuthProviderCatalogForTests([
      {
        provider: 'gitlab',
        service_provider: 'tenant-877397588196749312',
        provider_key: 'gitlab:tenant-877397588196749312',
        website: 'http://115.29.110.74',
        website_origin: 'http://115.29.110.74',
      },
    ])
    try {
      const href = buildRepoOAuthStartHref('http://115.29.110.74/example-user/somanyad.git')
      expect(href).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(href).toContain('service_provider=tenant-877397588196749312')
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })
})

describe('supportsRepoOAuthAuthorize', () => {
  it('当前支持 GitHub 与 GitLab 仓库 OAuth', () => {
    expect(supportsRepoOAuthAuthorize('https://github.com/org/repo.git')).toBe(true)
    expect(supportsRepoOAuthAuthorize('https://gitlab.daydaymoney.com/group/repo.git')).toBe(true)
    expect(supportsRepoOAuthAuthorize('git@github.com:org/repo.git')).toBe(true)
    expect(supportsRepoOAuthAuthorize('ssh://git@github.com/org/repo.git')).toBe(true)
    expect(supportsRepoOAuthAuthorize('git@localhost:group/repo.git')).toBe(true)
    expect(supportsRepoOAuthAuthorize('https://bitbucket.org/team/repo')).toBe(false)
  })
})

describe('resolveRepoOAuthProvider', () => {
  it('可从仓库 URL 识别 provider', () => {
    expect(resolveRepoOAuthProvider('https://github.com/org/repo.git')).toBe('github')
    expect(resolveRepoOAuthProvider('https://gitlab.daydaymoney.com/group/repo.git')).toBe('gitlab')
    expect(resolveRepoOAuthProvider('git@github.com:org/repo.git')).toBe('github')
    expect(resolveRepoOAuthProvider('ssh://git@github.com/org/repo.git')).toBe('github')
    expect(resolveRepoOAuthProvider('git@localhost:group/repo.git')).toBe('gitlab')
    expect(resolveRepoOAuthProvider('https://bitbucket.org/team/repo')).toBe('')
  })

  it('catalog 命中 IP 自托管 GitLab', () => {
    setGitOAuthProviderCatalogForTests(TENCENT_CATALOG)
    try {
      expect(resolveRepoOAuthProvider('http://1.117.67.121:8012/group/repo')).toBe('gitlab')
      expect(resolveRepoOAuthAuthorizeUrl(resolveRepoOAuthProvider('http://1.117.67.121:8012/group/repo'))).toBe(
        '/api/git-oauth/gitlab-start-from-gateway/',
      )
      expect(supportsRepoOAuthAuthorize('http://1.117.67.121:8012/group/repo')).toBe(true)
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })

  it('catalog 可按 hostname 命中 SSH IP 仓库（无端口）', () => {
    const synologyCatalog = [
      {
        provider: 'gitlab',
        service_provider: 'synology-gitlab',
        provider_key: 'gitlab:synology-gitlab',
        website: 'http://127.0.0.1:8012',
        website_origin: 'http://127.0.0.1:8012',
      },
    ]
    setGitOAuthProviderCatalogForTests(synologyCatalog)
    try {
      const sshUrl = 'git@gitlab.daydaymoney.com:example-user/somanyad.git'
      expect(resolveRepoOAuthProvider(sshUrl)).toBe('gitlab')
      expect(supportsRepoOAuthAuthorize(sshUrl)).toBe(true)
      expect(resolveRepoOAuthAuthorizeUrl(resolveRepoOAuthProvider(sshUrl))).toBe(
        '/api/git-oauth/gitlab-start-from-gateway/',
      )
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })
})

describe('Path A GitLab IP catalog', () => {
  const pathARepo = 'http://115.29.110.74/example-user/somanyad.git'

  it('无 catalog 时 IP 仓不能靠 heuristic 当成 GitLab', () => {
    setGitOAuthProviderCatalogForTests(null)
    expect(resolveRepoOAuthProvider(pathARepo)).toBe('')
    expect(resolveRepoOAuthProviderInfo(pathARepo)).toBe(null)
    expect(supportsRepoOAuthAuthorize(pathARepo)).toBe(false)
  })

  it('catalog 按 origin 命中 Path A 并带 tenant service_provider', () => {
    setGitOAuthProviderCatalogForTests([
      {
        provider: 'gitlab',
        service_provider: 'tenant-877397588196749312',
        provider_key: 'gitlab:tenant-877397588196749312',
        website: 'http://115.29.110.74',
        website_origin: 'http://115.29.110.74',
      },
    ])
    try {
      expect(resolveRepoOAuthProvider(pathARepo)).toBe('gitlab')
      expect(resolveRepoOAuthProviderInfo(pathARepo)).toEqual({
        provider: 'gitlab',
        service_provider: 'tenant-877397588196749312',
      })
      expect(supportsRepoOAuthAuthorize(pathARepo)).toBe(true)
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })

  it('loadGitOAuthProviderCatalog 带 company_id 与 X-Tenant-Id', async () => {
    setGitOAuthProviderCatalogForTests(null)
    let seenPath = ''
    let seenHeaders = {}
    const fetchFn = async (path, opts) => {
      seenPath = String(path || '')
      seenHeaders = opts?.headers || {}
      return {
        ok: true,
        json: async () => ({
          providers: [
            {
              provider: 'gitlab',
              service_provider: 'tenant-877397588196749312',
              provider_key: 'gitlab:tenant-877397588196749312',
              website: 'http://115.29.110.74',
            },
          ],
        }),
      }
    }
    try {
      await loadGitOAuthProviderCatalog(fetchFn, '877397588196749312')
      expect(seenPath).toContain('company_id=877397588196749312')
      expect(seenHeaders['X-Tenant-Id']).toBe('877397588196749312')
      expect(resolveRepoOAuthProviderInfo(pathARepo)?.service_provider).toBe(
        'tenant-877397588196749312',
      )
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })
})

describe('resolveRepoOAuthAuthorizeLabel', () => {
  it('从仓库 URL 解析域名标签', () => {
    expect(resolveRepoOAuthAuthorizeLabel('https://gitlab.daydaymoney.com/group/repo.git')).toBe(
      'gitlab.daydaymoney.com',
    )
    expect(resolveRepoOAuthAuthorizeLabel('git@github.com:org/repo.git')).toBe('github.com')
    expect(resolveRepoOAuthAuthorizeLabel('git@localhost:group/repo.git')).toBe('localhost')
    expect(resolveRepoOAuthAuthorizeLabel('')).toBe('仓库站点')
  })
})
