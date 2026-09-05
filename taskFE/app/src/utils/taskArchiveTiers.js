export const ARCHIVE_TIER_OPTIONS = [
  { value: '7d', label: '免费存放期(7天)' },
  { value: '1m', label: '一个月' },
  { value: '2m', label: '两个月' },
  { value: '3m', label: '三个月' },
  { value: '4m', label: '四个月' },
  { value: '5m', label: '五个月' },
  { value: '6m', label: '六个月' },
  { value: '7m', label: '七个月' },
  { value: '8m', label: '八个月' },
  { value: '9m', label: '九个月' },
  { value: '10m', label: '十个月' },
  { value: '11m', label: '十一个月' },
  { value: '12m', label: '一年' },
  { value: '24m', label: '两年' },
  { value: '36m', label: '三年' },
  { value: 'unlimited', label: '不限制' },
]

export function archiveTierLabel(tier) {
  const t = tier || '7d'
  const o = ARCHIVE_TIER_OPTIONS.find((x) => x.value === t)
  return o ? o.label : t
}
