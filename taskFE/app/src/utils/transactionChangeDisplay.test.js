import { describe, expect, it } from 'vitest'
import {
  ledgerSnapshotDisplay,
  ledgerSnapshotLines,
  pointsSourceTypeDisplay,
  pointsSourceTypeDisplayOrDash,
  transactionChangeDisplay,
} from './transactionChangeDisplay.js'

describe('pointsSourceTypeDisplay', () => {
  it('prefers API display when present', () => {
    expect(pointsSourceTypeDisplay({
      points_source_type: 'admin_grant',
      points_source_type_display: '后台赠送',
    })).toBe('后台赠送')
  })

  it('maps admin_grant when API display is missing', () => {
    expect(pointsSourceTypeDisplay({ points_source_type: 'admin_grant' })).toBe('后台赠送')
  })

  it('returns dash helper for empty source', () => {
    expect(pointsSourceTypeDisplayOrDash({})).toBe('—')
    expect(pointsSourceTypeDisplayOrDash({ points_source_type: '' })).toBe('—')
  })
})

describe('transactionChangeDisplay', () => {
  it('shows resource grant change_display instead of +0.00 yuan', () => {
    expect(transactionChangeDisplay({
      transaction_type: 'recharge',
      amount_points: 0,
      change_display: '任务帖 +10 帖',
    })).toBe('任务帖 +10 帖')
  })

  it('joins resource_changes when change_display is absent', () => {
    expect(transactionChangeDisplay({
      transaction_type: 'recharge',
      amount_points: 0,
      resource_changes: [
        { display: '任务帖 +10 帖' },
        { display: 'GitLab 磁盘 +5 GB（tencent-sh-1）' },
      ],
    })).toBe('任务帖 +10 帖；GitLab 磁盘 +5 GB（tencent-sh-1）')
  })

  it('formats money recharge as yuan when no quota change', () => {
    expect(transactionChangeDisplay({
      transaction_type: 'recharge',
      amount_points: 1550,
    })).toBe('+15.50 元')
  })

  it('formats money consumption as yuan', () => {
    expect(transactionChangeDisplay({
      transaction_type: 'consumption',
      amount_points: 30,
    })).toBe('-0.30 元')
  })

  it('shows em dash for zero-amount rows without resource details', () => {
    expect(transactionChangeDisplay({
      transaction_type: 'recharge',
      amount_points: 0,
      description: '管理员后台赠送资源',
    })).toBe('—')
  })
})

describe('ledgerSnapshotDisplay', () => {
  it('prefers API display_lines', () => {
    expect(ledgerSnapshotLines({
      ledger_snapshot: { display_lines: ['余额 1.00 → 0.70 元', '任务帖剩余 9'] },
    })).toEqual(['余额 1.00 → 0.70 元', '任务帖剩余 9'])
  })

  it('passes through GitLab region remaining display_lines (OPT-20260819-014)', () => {
    expect(ledgerSnapshotLines({
      ledger_snapshot: {
        display_lines: [
          '余额 0.00 元',
          '任务帖剩余 10',
          'GitLab 磁盘剩余 4 GB（tencent-shanghai-5）',
          'GitLab 流量剩余 8 GB（tencent-shanghai-5）',
        ],
      },
    })).toEqual([
      '余额 0.00 元',
      '任务帖剩余 10',
      'GitLab 磁盘剩余 4 GB（tencent-shanghai-5）',
      'GitLab 流量剩余 8 GB（tencent-shanghai-5）',
    ])
  })

  it('formats cash before/after points as yuan when snapshot missing', () => {
    expect(ledgerSnapshotDisplay({
      balance_before_points: 100,
      balance_after_points: 70,
    })).toBe('余额 1.00 → 0.70 元')
  })

  it('shows dash when no balance fields', () => {
    expect(ledgerSnapshotDisplay({})).toBe('—')
  })
})
