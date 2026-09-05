/**
 * 手机号表单：默认国家码 +86；提交 API 使用完整 E.164（含 + 与国号）。
 */
import { COUNTRY_DIAL_OPTIONS } from './countryDialCodes.js'

export const DEFAULT_PHONE_COUNTRY_PREFIX = '+86'

const _dialSorted = [...COUNTRY_DIAL_OPTIONS].sort((a, b) => b.code.length - a.code.length)

/** 去掉空白、+86、86 前缀，得到中国大陆 11 位（若可解析） */
export function normalizeCnPhoneDigits(input) {
  if (input == null) return ''
  let p = String(input).trim().replace(/\s/g, '')
  if (p.startsWith('+86')) p = p.slice(3).trim()
  else if (p.startsWith('86') && p.length === 13 && /^\d+$/.test(p)) p = p.slice(2)
  return p
}

export function isValidCnMobile11(digits) {
  return /^1\d{10}$/.test(digits)
}

/**
 * 提交 API 的完整号码：始终为 E.164，例如 +8613800138000、+85291234567
 */
export function buildPhoneForApi(prefix, national) {
  const prefRaw = (prefix || DEFAULT_PHONE_COUNTRY_PREFIX).trim()
  const digits = String(national || '').replace(/\D/g, '')
  if (!digits) return ''
  const cc = prefRaw.startsWith('+') ? prefRaw : `+${prefRaw.replace(/\D/g, '')}`
  return `${cc}${digits}`
}

export function maxNationalDigitsForPrefix(prefix) {
  const p = (prefix || '').trim()
  if (p === '+86' || p === '86') return 11
  return 15
}

export function isValidNationalForPrefix(prefix, nationalDigits) {
  const d = String(nationalDigits || '').replace(/\D/g, '')
  const p = (prefix || DEFAULT_PHONE_COUNTRY_PREFIX).trim()
  if (p === '+86' || p === '86') return /^1\d{10}$/.test(d)
  return d.length >= 6 && d.length <= 15
}

/** 客户端校验完整 E.164（与后端 canonical 宽松一致） */
export function isValidE164Loose(full) {
  const s = String(full || '').trim()
  return /^\+[1-9]\d{7,14}$/.test(s)
}

/**
 * 粘贴完整号码时解析为区号 + 国内号码段
 * @returns {{ prefix: string, national: string } | null}
 */
export function parsePasteToPrefixAndNational(pastedText) {
  let s = String(pastedText || '').trim().replace(/\s/g, '')
  if (!s) return null
  if (!s.startsWith('+')) {
    if (/^1\d{10}$/.test(s)) return { prefix: DEFAULT_PHONE_COUNTRY_PREFIX, national: s }
    if (/^861\d{10}$/.test(s)) return { prefix: DEFAULT_PHONE_COUNTRY_PREFIX, national: s.slice(2) }
    return null
  }
  for (const { code } of _dialSorted) {
    if (s.startsWith(code)) {
      const national = s.slice(code.length).replace(/\D/g, '')
      return { prefix: code, national }
    }
  }
  return null
}
