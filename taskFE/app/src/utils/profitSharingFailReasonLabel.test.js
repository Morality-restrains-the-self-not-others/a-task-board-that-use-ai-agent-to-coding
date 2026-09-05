import { describe, expect, it } from 'vitest'
import { profitSharingFailReasonLabel } from './profitSharingFailReasonLabel.js'

describe('profitSharingFailReasonLabel', () => {
  it('maps qualification_revoked to zh-CN', () => {
    expect(profitSharingFailReasonLabel('qualification_revoked')).toBe('推荐资格已撤销')
  })

  it('maps referrer_openid_missing to zh-CN', () => {
    expect(profitSharingFailReasonLabel('referrer_openid_missing')).toBe('推荐人未绑定微信收款账号')
  })

  it('maps WeChat empty receiver account PARAM_ERROR dump to bind-wechat hint', () => {
    const dump = 'create profit sharing failed: HTTP 400, {"code":"PARAM_ERROR","detail":{"location":"body","value":0},"message":"输入源“/body/receivers/0/account”映射到值字段“分账接收方帐号”字符串规则校验失败，字符数 0，小于最小值 1"}'
    expect(profitSharingFailReasonLabel(dump)).toBe('推荐人未绑定微信收款账号')
    expect(profitSharingFailReasonLabel(dump)).not.toContain('PARAM_ERROR')
    expect(profitSharingFailReasonLabel(dump)).not.toContain('HTTP 400')
  })

  it('maps WeChat profit-sharing fail codes to zh-CN', () => {
    expect(profitSharingFailReasonLabel('ACCOUNT_ABNORMAL')).toBe('分账接收账户异常')
    expect(profitSharingFailReasonLabel('NO_RELATION')).toBe('分账关系已解除')
    expect(profitSharingFailReasonLabel('RECEIVER_HIGH_RISK')).toBe('高风险接收方')
    expect(profitSharingFailReasonLabel('RECEIVER_REAL_NAME_NOT_VERIFIED')).toBe('接收方未实名')
    expect(profitSharingFailReasonLabel('NO_AUTH')).toBe('分账权限已解除')
    expect(profitSharingFailReasonLabel('RECEIVER_RECEIPT_LIMIT')).toBe('超出用户月收款限额')
    expect(profitSharingFailReasonLabel('PAYER_ACCOUNT_ABNORMAL')).toBe('分出方账户异常')
    expect(profitSharingFailReasonLabel('INVALID_REQUEST')).toBe('描述参数设置失败')
    expect(profitSharingFailReasonLabel('BALANCE_NOT_ENOUGH')).toBe('余额不足')
    expect(profitSharingFailReasonLabel('TIME_OUT_CLOSED')).toBe('超时关单')
  })

  it('keeps already-Chinese reasons and unknown codes', () => {
    expect(profitSharingFailReasonLabel('微信侧关闭')).toBe('微信侧关闭')
    expect(profitSharingFailReasonLabel('some_new_code')).toBe('some_new_code')
  })

  it('shows em-dash for empty fail_reason', () => {
    expect(profitSharingFailReasonLabel('')).toBe('—')
    expect(profitSharingFailReasonLabel('   ')).toBe('—')
    expect(profitSharingFailReasonLabel(null)).toBe('—')
  })

  it('净化为 SDK dump 且不泄漏微信签名头（OPT-20260824-084）', () => {
    const dump = `error http response:[StatusCode: 403 Code: "NOT_ENOUGH"
Message: 基本账户余额不足，请充值后重新发起
Header:
 - Wechatpay-Signature=[NVkk5cJ3rdfE9Yl2OJ0s1RpF0cRBqqXxO8y6SrL3ESVtk]
 - Wechatpay-Nonce=[abc123]
]`
    const out = profitSharingFailReasonLabel(dump)
    expect(out).toContain('分账失败')
    expect(out).toContain('余额不足')
    expect(out).not.toContain('Wechatpay-Signature')
    expect(out).not.toContain('NVkk5cJ3rdfE9')
    expect(out).not.toContain('error http response')
  })

  it('SDK dump 带 Message 无 NOT_ENOUGH 时展示 Message 摘要', () => {
    const dump = `error http response:[StatusCode: 400 Code: "PARAM_ERROR"
Message: 参数错误
Header:
 - Wechatpay-Signature=[sig-secret]
]`
    expect(profitSharingFailReasonLabel(dump)).toBe('分账失败：参数错误')
  })

  it('SDK dump 无 Message 时展示通用失败文案', () => {
    const dump = `error http response:[StatusCode: 500 Code: "SYSTEM_ERROR"
Header:
 - Wechatpay-Signature=[sig-secret]
]`
    expect(profitSharingFailReasonLabel(dump)).toBe('分账失败，请稍后重试或联系财务')
  })

  it('机器码 BALANCE_NOT_ENOUGH 仍映射为「余额不足」不被 dump 清洗劫持', () => {
    expect(profitSharingFailReasonLabel('BALANCE_NOT_ENOUGH')).toBe('余额不足')
  })
})
