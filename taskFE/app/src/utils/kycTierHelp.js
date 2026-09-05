/**
 * KYC 身份等级说明（超管 drawer 等展示用）。
 * 含义与 taskAuth evaluate / 人工覆盖语义对齐；具体单笔/日限额以 auth_kyc_limit_policy 为准，不在此硬编码金额。
 */
export const KYC_TIER_HELP = [
  {
    tier: 'T0_unverified',
    label: 'T0 未验证',
    description: '默认等级。尚未完成手机号验证；充值按该档限额策略执行（额度最低）。',
  },
  {
    tier: 'T1_basic',
    label: 'T1 基础',
    description: '完成手机验证后可由系统自动评估升至该级；常规充值额度。',
  },
  {
    tier: 'T2_enhanced',
    label: 'T2 增强',
    description: '需人工审核或管理员覆盖；适用于更高合规要求与更大额充值。',
  },
]

export function kycTierHelpText() {
  return KYC_TIER_HELP.map((row) => `${row.label}：${row.description}`).join('\n')
}
