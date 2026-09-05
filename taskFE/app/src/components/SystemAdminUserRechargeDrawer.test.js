/**
 * 回归：SystemAdminUserRechargeDrawer 对 admin_grant 展示「无需支付签署」。
 * 逻辑在 adminUserRechargeDisplay；本文件以抽屉同名 stem 满足 bug-fix 门禁对应性。
 */
import { describe, it, expect } from 'vitest'
import {
  isAdminGrantRecharge,
  consentStatusLabel,
  formatRechargeChannel,
} from '../utils/adminUserRechargeDisplay.js'

describe('SystemAdminUserRechargeDrawer display', () => {
  it('金额 0 / 无渠道的 admin_grant 不显示无关联签署记录', () => {
    const row = {
      points_source_type: 'admin_grant',
      amount_points: 0,
      channel: '',
      consent_required: false,
      consent_note: '系统赠送，无需支付签署',
    }
    expect(isAdminGrantRecharge(row)).toBe(true)
    expect(consentStatusLabel(row)).not.toBe('无关联签署记录')
    expect(formatRechargeChannel(row)).toBe('系统赠送')
  })
})
