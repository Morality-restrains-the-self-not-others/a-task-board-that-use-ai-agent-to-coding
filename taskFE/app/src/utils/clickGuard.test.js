// @vitest-environment jsdom
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import {
  IDEMPOTENCY_HEADER,
  createClickGuard,
  mergeIdempotencyHeaders,
  newIdempotencyKey,
} from './clickGuard.js'

describe('newIdempotencyKey', () => {
  it('returns a non-empty string', () => {
    const key = newIdempotencyKey()
    expect(typeof key).toBe('string')
    expect(key.length).toBeGreaterThan(8)
  })
})

describe('mergeIdempotencyHeaders', () => {
  it('adds Idempotency-Key without dropping existing headers', () => {
    const headers = mergeIdempotencyHeaders(
      { Accept: 'application/json' },
      'ik-1',
    )
    expect(headers.Accept).toBe('application/json')
    expect(headers[IDEMPOTENCY_HEADER]).toBe('ik-1')
  })

  it('leaves headers unchanged when key is empty', () => {
    const headers = mergeIdempotencyHeaders({ Accept: 'text/plain' }, '')
    expect(headers).toEqual({ Accept: 'text/plain' })
  })
})

describe('createClickGuard', () => {
  it('skips overlapping in-flight runs', async () => {
    const guard = createClickGuard({ debounceMs: 0 })
    let release
    const gate = new Promise((resolve) => {
      release = resolve
    })
    const first = guard.run(async () => {
      await gate
      return 'ok'
    })
    const second = await guard.run(async () => 'late')
    expect(second).toEqual({ skipped: true, reason: 'in-flight' })
    release()
    const firstResult = await first
    expect(firstResult).toEqual({ skipped: false, result: 'ok' })
  })

  it('reuses one Idempotency-Key for retries inside a single accepted run', async () => {
    const keys = []
    const guard = createClickGuard({
      debounceMs: 0,
      newKey: () => 'stable-intent-key',
    })
    await guard.run(async ({ idempotencyKey, headers }) => {
      keys.push(idempotencyKey, headers[IDEMPOTENCY_HEADER])
      keys.push(idempotencyKey)
    })
    expect(keys).toEqual(['stable-intent-key', 'stable-intent-key', 'stable-intent-key'])
  })

  it('issues a new key after the previous run finishes', async () => {
    let n = 0
    const guard = createClickGuard({
      debounceMs: 0,
      newKey: () => `k-${++n}`,
    })
    const a = await guard.run(async ({ idempotencyKey }) => idempotencyKey)
    const b = await guard.run(async ({ idempotencyKey }) => idempotencyKey)
    expect(a.result).toBe('k-1')
    expect(b.result).toBe('k-2')
  })

  it('debounces a second click after the first run completes', async () => {
    let clock = 1000
    const guard = createClickGuard({
      debounceMs: 300,
      now: () => clock,
    })
    await guard.run(async () => 'first')
    clock += 100
    const skipped = await guard.run(async () => 'second')
    expect(skipped).toEqual({ skipped: true, reason: 'debounce' })
    clock += 300
    const next = await guard.run(async () => 'third')
    expect(next).toEqual({ skipped: false, result: 'third' })
  })

  it('await 结束后模板绑定 isBusy() 的按钮解除 disabled', async () => {
    const guard = createClickGuard({ debounceMs: 0 })
    const Comp = defineComponent({
      setup() {
        return { guard }
      },
      template: '<button type="button" :disabled="guard.isBusy()" data-testid="guard-btn">go</button>',
    })
    const wrapper = mount(Comp)
    const btn = () => wrapper.get('[data-testid="guard-btn"]')
    expect(btn().attributes('disabled')).toBeUndefined()

    let release
    const gate = new Promise((resolve) => {
      release = resolve
    })
    const pending = guard.run(async () => {
      await gate
    })
    await nextTick()
    expect(btn().attributes('disabled')).toBeDefined()

    release()
    await pending
    await nextTick()
    expect(btn().attributes('disabled')).toBeUndefined()
    expect(guard.isBusy()).toBe(false)
  })
})
