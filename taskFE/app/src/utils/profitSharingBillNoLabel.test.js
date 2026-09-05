import { describe, expect, it } from 'vitest'
import {
  profitSharingBillNoLabel,
  profitSharingBillNoTitle,
} from './profitSharingBillNoLabel.js'

describe('profitSharingBillNoLabel', () => {
  it('prefers WeChat profit-sharing order_id over merchant out_order_no', () => {
    expect(profitSharingBillNoLabel({
      wechat_profit_sharing_id: '3000002222',
      out_profit_sharing_no: 'PS880041686850371586',
    })).toBe('3000002222')
  })

  it('falls back to merchant out_order_no when WeChat id is empty', () => {
    expect(profitSharingBillNoLabel({
      wechat_profit_sharing_id: '',
      out_profit_sharing_no: 'PS880041686850371586',
    })).toBe('PS880041686850371586')
  })

  it('shows em-dash when both identifiers are empty', () => {
    expect(profitSharingBillNoLabel({ wechat_profit_sharing_id: '', out_profit_sharing_no: '' })).toBe('—')
    expect(profitSharingBillNoLabel({ wechat_profit_sharing_id: '  ', out_profit_sharing_no: '  ' })).toBe('—')
    expect(profitSharingBillNoLabel(null)).toBe('—')
  })
})

describe('profitSharingBillNoTitle', () => {
  it('notes merchant out_order_no is not a separate entity when both ids exist', () => {
    const title = profitSharingBillNoTitle({
      wechat_profit_sharing_id: '3000002222',
      out_profit_sharing_no: 'PS880041686850371586',
    })
    expect(title).toContain('PS880041686850371586')
    expect(title).toContain('out_order_no')
    expect(title).toContain('非独立实体')
  })

  it('explains fallback when only merchant out_order_no is present', () => {
    const title = profitSharingBillNoTitle({
      wechat_profit_sharing_id: '',
      out_profit_sharing_no: 'PS880041686850371586',
    })
    expect(title).toContain('尚未返回微信分账单号')
    expect(title).toContain('out_order_no')
  })

  it('empty title when no identifiers', () => {
    expect(profitSharingBillNoTitle({ wechat_profit_sharing_id: '', out_profit_sharing_no: '' })).toBe('')
    expect(profitSharingBillNoTitle(null)).toBe('')
  })
})
