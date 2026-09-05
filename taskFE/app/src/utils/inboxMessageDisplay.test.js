// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] inboxMessageDisplay.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { inboxBodyWithoutDuplicatedReason } = await import('./inboxMessageDisplay.js')

describe('inboxBodyWithoutDuplicatedReason', () => {
  it('strips trailing 理由 suffix that matches the reason field', () => {
    const reason = '请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。'
    const body = `系统管理员（用户 ID：bootstrap-admin）以你的身份登录了本平台。\n理由：${reason}`
    expect(inboxBodyWithoutDuplicatedReason(body, reason)).toBe(
      '系统管理员（用户 ID：bootstrap-admin）以你的身份登录了本平台。',
    )
  })

  it('leaves body unchanged when reason is not embedded', () => {
    const body = '系统管理员（用户 ID：admin）以你的身份登录了本平台。'
    expect(inboxBodyWithoutDuplicatedReason(body, '排查线上工单问题')).toBe(body)
  })
})
}
