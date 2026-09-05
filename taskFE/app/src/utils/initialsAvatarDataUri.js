/** 本地首字母占位头像（data URI），全站统一替代外网 Dicebear API。 */

const AVATAR_BG_PALETTE = [
  '#ffdfbf',
  '#fff1e6',
  '#fde2e4',
  '#fad2e1',
  '#e2ece9',
  '#bee1e6',
  '#dde2ff',
  '#e4e4ff',
]

/** 由 seed 稳定映射到调色板背景色（与原 Dicebear 多色效果接近）。 */
export function avatarBackgroundFromSeed(seed) {
  const s = String(seed || '')
  let hash = 0
  for (let i = 0; i < s.length; i += 1) {
    hash = (hash * 31 + s.charCodeAt(i)) >>> 0
  }
  return AVATAR_BG_PALETTE[hash % AVATAR_BG_PALETTE.length]
}

/**
 * @param {string} seed 用户名/邮箱等，用于首字母与背景色
 * @param {string} [background] 可选固定背景色；省略则按 seed 选色
 */
export function initialsAvatarDataUri(seed, background) {
  const letter = String(seed || '?').trim().charAt(0).toUpperCase() || '?'
  const bg = background || avatarBackgroundFromSeed(seed)
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">` +
    `<rect width="64" height="64" rx="32" fill="${bg}"/>` +
    `<text x="32" y="36" text-anchor="middle" font-size="28" fill="#334155" font-family="system-ui,sans-serif">${letter}</text>` +
    `</svg>`
  return `data:image/svg+xml,${encodeURIComponent(svg)}`
}
