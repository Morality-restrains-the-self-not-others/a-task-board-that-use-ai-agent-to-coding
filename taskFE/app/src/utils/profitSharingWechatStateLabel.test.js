import { describe, expect, it } from 'vitest'
import {
  profitSharingWechatStateLabel,
  profitSharingWechatStateTitle,
} from './profitSharingWechatStateLabel.js'

describe('profitSharingWechatStateLabel', () => {
  it('maps not_submitted machine code to 未向微信发起分账 (not user WeChat bind)', () => {
    expect(profitSharingWechatStateLabel('not_submitted')).toBe('未向微信发起分账')
    expect(profitSharingWechatStateLabel('not_submitted')).not.toBe('not_submitted')
  })

  it('still maps legacy Chinese sentinel 尚未提交微信 to same label (OPT-20260826-009)', () => {
    expect(profitSharingWechatStateLabel('尚未提交微信')).toBe('未向微信发起分账')
    expect(profitSharingWechatStateLabel('尚未提交微信')).not.toBe('尚未提交微信')
  })

  it('maps WeChat QueryOrder PROCESSING / FINISHED to zh-CN', () => {
    expect(profitSharingWechatStateLabel('PROCESSING')).toBe('微信处理中')
    expect(profitSharingWechatStateLabel('FINISHED')).toBe('微信已分账完成')
  })

  it('shows em-dash for empty wechat_state', () => {
    expect(profitSharingWechatStateLabel('')).toBe('—')
    expect(profitSharingWechatStateLabel('   ')).toBe('—')
    expect(profitSharingWechatStateLabel(null)).toBe('—')
  })

  it('keeps unknown WeChat codes so new official states still show', () => {
    expect(profitSharingWechatStateLabel('SOME_NEW_STATE')).toBe('SOME_NEW_STATE')
  })
})

describe('profitSharingWechatStateTitle', () => {
  it('explains not_submitted is profit-sharing order state, not bind status', () => {
    const title = profitSharingWechatStateTitle('not_submitted')
    expect(title).toContain('分账单')
    expect(title).toContain('绑定')
    expect(title).toContain('未向微信发起')
  })

  it('legacy 尚未提交微信 sentinel still gets the same title (OPT-20260826-009)', () => {
    const title = profitSharingWechatStateTitle('尚未提交微信')
    expect(title).toContain('未向微信发起')
  })

  it('prefers wechat_error over state title', () => {
    expect(profitSharingWechatStateTitle('FINISHED', '查询微信分账状态失败，请稍后重试')).toBe(
      '查询微信分账状态失败，请稍后重试',
    )
  })

  it('explains PROCESSING and FINISHED as QueryOrder states', () => {
    expect(profitSharingWechatStateTitle('PROCESSING')).toContain('处理中')
    expect(profitSharingWechatStateTitle('FINISHED')).toContain('终态')
  })

  it('empty state has empty title', () => {
    expect(profitSharingWechatStateTitle('')).toBe('')
    expect(profitSharingWechatStateTitle(null)).toBe('')
  })
})
