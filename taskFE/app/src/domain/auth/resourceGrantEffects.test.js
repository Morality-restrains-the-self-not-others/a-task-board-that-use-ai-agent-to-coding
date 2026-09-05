import { describe, expect, it } from 'vitest'
import {
  EFFECT_OPERATE,
  EFFECT_VIEW,
  grantsMapToList,
  isPageFullyGranted,
  maxGrantEffect,
  normalizeGrantEffect,
  normalizeGrantsMap,
  setGrantEffect,
  togglePageGrants,
} from './resourceGrantEffects.js'

describe('resourceGrantEffects', () => {
  it('normalizes effect and max', () => {
    expect(normalizeGrantEffect('view')).toBe(EFFECT_VIEW)
    expect(normalizeGrantEffect('')).toBe(EFFECT_OPERATE)
    expect(maxGrantEffect('view', 'operate')).toBe(EFFECT_OPERATE)
    expect(maxGrantEffect('', 'view')).toBe(EFFECT_VIEW)
  })

  it('upgrades legacy key arrays to operate grants', () => {
    expect(normalizeGrantsMap(['a.main', 'b.main'])).toEqual({
      'a.main': EFFECT_OPERATE,
      'b.main': EFFECT_OPERATE,
    })
  })

  it('reads bound rows with effect', () => {
    expect(
      normalizeGrantsMap([
        { group_key: 'a.main', effect: 'view' },
        { group_key: 'b.main', effect: 'operate' },
      ]),
    ).toEqual({ 'a.main': EFFECT_VIEW, 'b.main': EFFECT_OPERATE })
  })

  it('setGrantEffect and page toggle', () => {
    let m = setGrantEffect({}, 'x.main', EFFECT_VIEW)
    expect(m['x.main']).toBe(EFFECT_VIEW)
    m = setGrantEffect(m, 'x.main', EFFECT_OPERATE)
    expect(m['x.main']).toBe(EFFECT_OPERATE)
    m = setGrantEffect(m, 'x.main', null)
    expect(m['x.main']).toBeUndefined()

    const page = {
      group_key: 'p',
      children: [{ group_key: 'p.a' }, { group_key: 'p.b' }],
    }
    m = togglePageGrants({}, page, true)
    expect(isPageFullyGranted(m, page)).toBe(true)
    expect(grantsMapToList(m).every((g) => g.effect === EFFECT_OPERATE)).toBe(true)
    m = togglePageGrants(m, page, false)
    expect(Object.keys(m)).toHaveLength(0)
  })
})
