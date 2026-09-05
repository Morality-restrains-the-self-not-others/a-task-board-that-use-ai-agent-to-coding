// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  collectLinkedRepoBranchTargets,
  githubRepoSlugFromUrl,
  intersectBranchNameLists,
  pickPreferredCommonMergeTargetBranch,
  repoCloneIdentityOptionLabel,
  resolveBranchNamePlaceholders,
  resolveDefaultCompanyGitIdentityId,
  resolveLayerGraphPushTargetBranch,
  resolveLayerGraphMergeTargetBranch,
  resolveSelectedGithubUserIdForRepo,
  validateTaskRepoSavedAssociations,
} from './taskDetailBranchAndRepoUtils.js'

describe('resolveSelectedGithubUserIdForRepo', () => {
  it('草稿优先，其次已保存绑定，仅一个已连接账号时自动选中', () => {
    expect(
      resolveSelectedGithubUserIdForRepo({
        draft: 'd1',
        boundUserId: 'b1',
        connectedOptions: [{ github_user_id: 'c1' }],
      }),
    ).toBe('d1')
    expect(
      resolveSelectedGithubUserIdForRepo({
        boundUserId: 'b1',
        connectedOptions: [{ github_user_id: 'c1' }],
      }),
    ).toBe('b1')
    expect(
      resolveSelectedGithubUserIdForRepo({
        connectedOptions: [{ github_user_id: 'c1' }],
      }),
    ).toBe('c1')
    expect(
      resolveSelectedGithubUserIdForRepo({
        connectedOptions: [{ github_user_id: 'c1' }, { github_user_id: 'c2' }],
      }),
    ).toBe('')
  })
})

describe('intersectBranchNameLists', () => {
  it('T1: 多列表求交并稳定排序', () => {
    expect(intersectBranchNameLists([['b', 'a'], ['c', 'b']])).toEqual(['b'])
    expect(intersectBranchNameLists([['main', 'develop', 'x'], ['develop', 'main']])).toEqual([
      'develop',
      'main',
    ])
  })

  it('T2: 单列表去重排序', () => {
    expect(intersectBranchNameLists([['z', 'a', 'a']])).toEqual(['a', 'z'])
  })

  it('T3: 空入参或含空列表返回 []', () => {
    expect(intersectBranchNameLists([])).toEqual([])
    expect(intersectBranchNameLists(null)).toEqual([])
    expect(intersectBranchNameLists([['main'], []])).toEqual([])
    expect(intersectBranchNameLists([['main'], null])).toEqual([])
  })
})

describe('collectLinkedRepoBranchTargets', () => {
  it('按关联项目展开全部仓库并去重', () => {
    const targets = collectLinkedRepoBranchTargets(
      [{ project_id: '1' }, { project_id: '1' }, { project_id: '' }],
      (pid) => (pid === '1' ? ['https://a.git', 'https://b.git', 'https://a.git'] : []),
    )
    expect(targets).toEqual([
      { projectId: '1', repoUrl: 'https://a.git' },
      { projectId: '1', repoUrl: 'https://b.git' },
    ])
  })
})

describe('pickPreferredCommonMergeTargetBranch', () => {
  it('优先 develop，其次最新 release/*，再次 main，否则空', () => {
    expect(pickPreferredCommonMergeTargetBranch(['main', 'develop', 'release/a'])).toBe('develop')
    expect(
      pickPreferredCommonMergeTargetBranch(['main', 'release/2026-01-01_x', 'release/2026-07-12_y']),
    ).toBe('release/2026-07-12_y')
    expect(pickPreferredCommonMergeTargetBranch(['feature/x', 'main'])).toBe('main')
    expect(pickPreferredCommonMergeTargetBranch(['feature/x', 'hotfix/y'])).toBe('')
    expect(pickPreferredCommonMergeTargetBranch([])).toBe('')
    expect(pickPreferredCommonMergeTargetBranch(null)).toBe('')
  })
})

describe('resolveDefaultCompanyGitIdentityId', () => {
  it('返回 is_default 为 true 的身份 id', () => {
    expect(
      resolveDefaultCompanyGitIdentityId([
        { id: 'a', is_default: false },
        { id: 'default-1', is_default: true },
      ]),
    ).toBe('default-1')
  })

  it('无默认身份时返回空字符串', () => {
    expect(resolveDefaultCompanyGitIdentityId([{ id: 'a', is_default: false }])).toBe('')
    expect(resolveDefaultCompanyGitIdentityId(null)).toBe('')
  })
})

describe('validateTaskRepoSavedAssociations', () => {
  const ghRepo = 'git@github.com:org/app.git'

  it('全部已保存时返回 ok', () => {
    const result = validateTaskRepoSavedAssociations(
      [ghRepo, 'https://gitlab.com/group/proj.git'],
      (url) => (url.includes('github') ? 'git-id-1' : 'git-id-2'),
      (url) => (url.includes('github') ? 1001 : null),
    )
    expect(result.ok).toBe(true)
  })

  it('缺少 Git 身份或 GitHub 绑定时返回提示', () => {
    const result = validateTaskRepoSavedAssociations(
      [ghRepo],
      () => '',
      () => null,
    )
    expect(result.ok).toBe(false)
    expect(result.missingGit).toEqual([ghRepo])
    expect(result.missingGithub).toEqual([ghRepo])
    expect(result.message).toContain('Git 提交身份')
    expect(result.message).toContain('GitHub App 授权账号')
  })

  it('非 GitHub 仓库不要求 GitHub 授权', () => {
    const gitlab = 'git@gitlab.com:group/proj.git'
    const result = validateTaskRepoSavedAssociations(
      [gitlab],
      () => 'id-1',
      () => null,
    )
    expect(result.ok).toBe(true)
    expect(result.missingGithub).toEqual([])
  })
})

describe('githubRepoSlugFromUrl', () => {
  it('解析 owner/repo', () => {
    expect(githubRepoSlugFromUrl('https://github.com/MyOrg/MyRepo.git')).toBe('myorg/myrepo')
  })
})

describe('repoCloneIdentityOptionLabel', () => {
  it('依序拼接用户名、邮箱、标签（不因 display_name/label 单独短路）', () => {
    expect(
      repoCloneIdentityOptionLabel({
        id: '859748037572919296',
        display_name: 'system-auto',
        label: 'system-auto',
        git_user_name: 'alice',
        git_user_email: 'alice@example.com',
      }),
    ).toBe('alice <alice@example.com> · system-auto')
  })

  it('有 label 时仍须带上用户名与邮箱', () => {
    expect(
      repoCloneIdentityOptionLabel({
        id: '1',
        label: '工作身份',
        git_user_name: 'bob',
        git_user_email: 'bob@example.com',
      }),
    ).toBe('bob <bob@example.com> · 工作身份')
    expect(
      repoCloneIdentityOptionLabel({
        id: '2',
        git_user_name: 'bob',
        git_user_email: 'bob@example.com',
      }),
    ).toBe('bob <bob@example.com>')
  })

  it('仅部分字段时按可用项依序拼接；全缺时未命名身份', () => {
    expect(
      repoCloneIdentityOptionLabel({
        id: '3',
        label: 'system-auto',
      }),
    ).toBe('system-auto')
    expect(
      repoCloneIdentityOptionLabel({
        id: '4',
        git_user_name: 'only-name',
        label: 'tag',
      }),
    ).toBe('only-name · tag')
    expect(repoCloneIdentityOptionLabel({ id: '859748037572919296' })).toBe('未命名身份')
    expect(repoCloneIdentityOptionLabel(null)).toBe('')
  })

  it('无 name/email/label 时回退 display_name', () => {
    expect(
      repoCloneIdentityOptionLabel({
        id: '5',
        display_name: 'legacy display',
      }),
    ).toBe('legacy display')
  })
})

describe('resolveBranchNamePlaceholders / resolveLayerGraphPushTargetBranch', () => {
  it('推送目标分支替换 ${taskTitle} 与历史 __taskTitle_', () => {
    const task = {
      title: 'Fix Push OAuth',
      branch_strategy: {
        // 与真实落库一致：模板分隔符 _ + sanitize('${taskTitle}')=__taskTitle_ → ___taskTitle_
        work_branch_name:
          'feature/2026-07-12_user_daydaymoney${taskId}___taskTitle_',
      },
    }
    const branch = resolveLayerGraphPushTargetBranch(task, 'task_129')
    expect(branch).toBe('feature/2026-07-12_user_daydaymoneytask_129_Fix_Push_OAuth')
  })

  it('resolveBranchNamePlaceholders 保留未提供的 taskId 占位符', () => {
    expect(
      resolveBranchNamePlaceholders('x_${taskId}_${taskTitle}', {
        taskTitleSegment: 't',
      }),
    ).toBe('x_${taskId}_t')
  })

  it('resolveLayerGraphMergeTargetBranch 读取 merge_target_branch_name', () => {
    const task = {
      title: 'Merge Me',
      branch_strategy: {
        merge_target_branch_name: 'develop',
        work_branch_name: 'feature/${taskId}',
      },
    }
    expect(resolveLayerGraphMergeTargetBranch(task, 'task_1')).toBe('develop')
  })
})
