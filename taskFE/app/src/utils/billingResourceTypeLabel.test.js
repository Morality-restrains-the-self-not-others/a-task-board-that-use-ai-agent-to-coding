import { describe, expect, it } from 'vitest'
import { billingResourceTypeLabel } from './billingResourceTypeLabel.js'

describe('billingResourceTypeLabel', () => {
  it('labels known resource types in zh-CN', () => {
    expect(billingResourceTypeLabel('task_post')).toBe('任务帖')
    expect(billingResourceTypeLabel('gitlab_disk')).toBe('GitLab 磁盘')
    expect(billingResourceTypeLabel('gitlab_traffic')).toBe('GitLab 流量')
  })

  it('falls back to the raw code for unknown types', () => {
    expect(billingResourceTypeLabel('storage_extra')).toBe('storage_extra')
    expect(billingResourceTypeLabel('')).toBe('')
    expect(billingResourceTypeLabel(null)).toBe(null)
  })
})
