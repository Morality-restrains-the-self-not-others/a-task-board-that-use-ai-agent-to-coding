// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] referralAccessCodeUtils.normalize.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const {
  fetchOwnReferralAccessCode,
  normalizeUrlAccessCodeToOwn,
  readCachedOwnReferralAccessCode,
} = await import('./referralAccessCodeUtils.js')

const REFERRER_CODE = 'w25brikFiT'
const OWN_CODE = '9aaHjbryhL'
const PAGE = '/profile/referral/'

function jsonOk(body) {
  return Promise.resolve({ ok: true, status: 200, json: async () => body })
}

function resetUrl(url = PAGE) {
  window.history.replaceState({}, '', url)
}

function storedReferralCode() {
  return localStorage.getItem('referralCode') || ''
}

describe('fetchOwnReferralAccessCode', () => {
  it('returns { ok: true, code } when status API returns access_code', async () => {
    const fetchImpl = vi.fn(async () => jsonOk({ access_code: OWN_CODE }))
    const res = await fetchOwnReferralAccessCode(fetchImpl)
    expect(res).toEqual({ ok: true, code: OWN_CODE })
    expect(fetchImpl).toHaveBeenCalledWith(
      '/api/accounts/users/referral-codes/status/',
      expect.objectContaining({ credentials: 'include' }),
    )
  })

  it('returns { ok: true, code: "" } when status API omits access_code', async () => {
    const fetchImpl = vi.fn(async () => jsonOk({ has_active_code: false }))
    const res = await fetchOwnReferralAccessCode(fetchImpl)
    expect(res).toEqual({ ok: true, code: '' })
  })

  it('returns { ok: false } on non-ok response (e.g. 401)', async () => {
    const fetchImpl = vi.fn(async () => ({ ok: false, status: 401, json: async () => ({}) }))
    const res = await fetchOwnReferralAccessCode(fetchImpl)
    expect(res).toEqual({ ok: false, code: '' })
  })

  it('returns { ok: false } on network failure', async () => {
    const fetchImpl = vi.fn(async () => { throw new Error('net down') })
    const res = await fetchOwnReferralAccessCode(fetchImpl)
    expect(res).toEqual({ ok: false, code: '' })
  })
})

describe('normalizeUrlAccessCodeToOwn (OPT-20260824-005)', () => {
  beforeEach(() => {
    resetUrl(PAGE)
    localStorage.removeItem('referralCode')
    sessionStorage.removeItem('referral_own_access_code')
  })

  it('no-ops without URL param and empty localStorage (no fetch)', async () => {
    const fetchImpl = vi.fn()
    const changed = await normalizeUrlAccessCodeToOwn({ fetchImpl })
    expect(changed).toBe(false)
    expect(fetchImpl).not.toHaveBeenCalled()
    expect(window.location.search).toBe('')
  })

  it('rewrites URL accessCode to own code when it differs', async () => {
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}`)
    const changed = await normalizeUrlAccessCodeToOwn({ ownCode: OWN_CODE, fetchImpl: vi.fn() })
    expect(changed).toBe(true)
    expect(window.location.search).toBe(`?accessCode=${OWN_CODE}`)
  })

  it('keeps other URL params when normalizing accessCode', async () => {
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}&utm_source=e2e`)
    await normalizeUrlAccessCodeToOwn({ ownCode: OWN_CODE, fetchImpl: vi.fn() })
    expect(window.location.search).toBe(`?accessCode=${OWN_CODE}&utm_source=e2e`)
  })

  it('does not rewrite when URL accessCode already equals own code, but syncs localStorage', async () => {
    resetUrl(`${PAGE}?accessCode=${OWN_CODE}`)
    localStorage.setItem('referralCode', REFERRER_CODE)
    const changed = await normalizeUrlAccessCodeToOwn({ ownCode: OWN_CODE, fetchImpl: vi.fn() })
    expect(changed).toBe(false)
    expect(window.location.search).toBe(`?accessCode=${OWN_CODE}`)
    expect(storedReferralCode()).toBe(OWN_CODE)
  })

  it('fetches own code from status API when ownCode omitted, rewrites and caches', async () => {
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}`)
    const fetchImpl = vi.fn(async () => jsonOk({ access_code: OWN_CODE }))
    const changed = await normalizeUrlAccessCodeToOwn({ fetchImpl })
    expect(changed).toBe(true)
    expect(fetchImpl).toHaveBeenCalledTimes(1)
    expect(window.location.search).toBe(`?accessCode=${OWN_CODE}`)
    expect(readCachedOwnReferralAccessCode()).toBe(OWN_CODE)
  })

  it('reuses session cache instead of fetching again', async () => {
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}`)
    const fetchImpl = vi.fn(async () => jsonOk({ access_code: OWN_CODE }))
    await normalizeUrlAccessCodeToOwn({ fetchImpl })
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}`)
    const changed = await normalizeUrlAccessCodeToOwn({ fetchImpl })
    expect(changed).toBe(true)
    expect(fetchImpl).toHaveBeenCalledTimes(1)
  })

  it('leaves URL untouched when API is unreachable', async () => {
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}`)
    const fetchImpl = vi.fn(async () => { throw new Error('net down') })
    const changed = await normalizeUrlAccessCodeToOwn({ fetchImpl })
    expect(changed).toBe(false)
    expect(window.location.search).toBe(`?accessCode=${REFERRER_CODE}`)
    expect(storedReferralCode()).toBe('')
  })

  it('removes URL accessCode when API confirms user has no own code', async () => {
    resetUrl(`${PAGE}?accessCode=${REFERRER_CODE}`)
    const fetchImpl = vi.fn(async () => jsonOk({ has_active_code: false }))
    const changed = await normalizeUrlAccessCodeToOwn({ fetchImpl })
    expect(changed).toBe(true)
    expect(window.location.search).toBe('')
    expect(storedReferralCode()).toBe('')
  })

  it('handles access_code underscore variant', async () => {
    resetUrl(`${PAGE}?access_code=${REFERRER_CODE}`)
    await normalizeUrlAccessCodeToOwn({ ownCode: OWN_CODE, fetchImpl: vi.fn() })
    expect(window.location.search).toBe(`?access_code=${OWN_CODE}`)
  })

  it('syncs stale localStorage referralCode even when URL has no param', async () => {
    localStorage.setItem('referralCode', REFERRER_CODE)
    const changed = await normalizeUrlAccessCodeToOwn({ ownCode: OWN_CODE, fetchImpl: vi.fn() })
    expect(changed).toBe(false)
    expect(window.location.search).toBe('')
    expect(storedReferralCode()).toBe(OWN_CODE)
  })
})

}
