import { describe, expect, it } from 'vitest'
import {
  groupGitlabRegionsByProvider,
  gitlabProviderGroupLabel,
  isGitlabRegionPendingNode,
} from './gitlabRegionSelect.js'

describe('gitlabRegionSelect', () => {
  it('detects pending_node', () => {
    expect(isGitlabRegionPendingNode({ infra_status: 'pending_node' })).toBe(true)
    expect(isGitlabRegionPendingNode({ infra_status: 'ready' })).toBe(false)
    expect(isGitlabRegionPendingNode({})).toBe(false)
  })

  it('labels Aliyun pending groups for manual node', () => {
    expect(gitlabProviderGroupLabel('aliyun', [{ infra_status: 'pending_node' }])).toBe('阿里云（人工开通节点）')
    expect(gitlabProviderGroupLabel('aliyun', [{ infra_status: 'ready' }])).toBe('阿里云')
    expect(gitlabProviderGroupLabel('tencent', [])).toBe('腾讯云')
  })

  it('groups tencent then aliyun', () => {
    const groups = groupGitlabRegionsByProvider([
      { slug: 'aliyun-cn-hangzhou', name: '杭州', cloud_provider: 'aliyun', infra_status: 'pending_node' },
      { slug: 'tencent-sh-1', name: '上海一区', cloud_provider: 'tencent', infra_status: 'ready' },
    ])
    expect(groups.map((g) => g.provider)).toEqual(['tencent', 'aliyun'])
    expect(groups[1].label).toBe('阿里云（人工开通节点）')
    expect(groups[1].regions[0].slug).toBe('aliyun-cn-hangzhou')
  })
})
