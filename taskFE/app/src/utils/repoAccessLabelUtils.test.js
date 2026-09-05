import { describe, expect, it } from 'vitest'
import { mapRepoAccessApiToLabel } from './repoAccessLabelUtils.js'

describe('mapRepoAccessApiToLabel', () => {
  it('maps access_status needs_auth without GitHub wording', () => {
    expect(
      mapRepoAccessApiToLabel({
        is_accessible: false,
        access_status: 'needs_auth',
        message: '无法访问 GitLab 仓库：未检测到可用授权。',
      }),
    ).toBe('未授权')
  })

  it('maps accessible', () => {
    expect(
      mapRepoAccessApiToLabel({ is_accessible: true, access_status: 'accessible', message: '' }),
    ).toBe('可访问')
  })

  it('does not surface legacy 无 GitHub 仓库 as a label', () => {
    expect(
      mapRepoAccessApiToLabel({ is_accessible: false, message: '无 GitHub 仓库' }),
    ).toBe('不可访问')
  })
})
