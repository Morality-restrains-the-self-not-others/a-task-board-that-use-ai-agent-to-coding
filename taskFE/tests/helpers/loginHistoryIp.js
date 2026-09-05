// @ts-check
/**
 * RFC1918 / Docker-bridge IPv4 check for login-history E2E (OPT-20260826-002).
 * Public XFF must not show 10/8, 172.16/12, or 192.168/16.
 */

/**
 * @param {string} raw
 * @returns {boolean}
 */
export function isRfc1918Ipv4(raw) {
  const ip = String(raw || '').trim();
  const m = ip.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/);
  if (!m) return false;
  const a = Number(m[1]);
  const b = Number(m[2]);
  const c = Number(m[3]);
  const d = Number(m[4]);
  if ([a, b, c, d].some((n) => n > 255)) return false;
  if (a === 10) return true;
  if (a === 192 && b === 168) return true;
  if (a === 172 && b >= 16 && b <= 31) return true;
  return false;
}
