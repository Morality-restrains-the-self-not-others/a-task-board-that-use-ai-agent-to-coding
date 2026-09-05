// @vitest-environment node
import { describe, expect, it, beforeEach } from 'vitest'
import {
  collectSessionGrantTickets,
  consumeSessionGrantTicket,
  consumeSessionGrantTickets,
  gitsiteFromRepoUrl,
  hasSessionGrantForRepo,
  rememberGrantTicket,
  rememberGrantTicketFromSearch,
  sessionGrantTicketAny,
  sessionGrantTicketForRepo,
} from './grantTicketSession.js'

describe('grantTicketSession', () => {
  const memory = {}
  beforeEach(() => {
    for (const key of Object.keys(memory)) delete memory[key]
    globalThis.sessionStorage = {
      getItem: (k) => (k in memory ? memory[k] : null),
      setItem: (k, v) => { memory[k] = String(v) },
    }
  })

  it('parses gitsite host from https and ssh urls', () => {
    expect(gitsiteFromRepoUrl('https://github.com/a/b.git')).toBe('github.com')
    expect(gitsiteFromRepoUrl('git@gitlab.example:group/repo.git')).toBe('gitlab.example')
    expect(gitsiteFromRepoUrl('ssh://git@github.com/a/b.git')).toBe('github.com')
    expect(gitsiteFromRepoUrl('ssh://git@gitlab.daydaymoney.com:2222/g/r.git')).toBe('gitlab.daydaymoney.com')
  })

  it('stores a ticket per gitsite and as pending wildcard', () => {
    rememberGrantTicket('tkt-1', 'https://github.com/a/b.git')
    expect(hasSessionGrantForRepo('https://github.com/a/b.git')).toBe(true)
    expect(sessionGrantTicketForRepo('https://github.com/a/b.git')).toBe('tkt-1')
    expect(sessionGrantTicketAny()).toBe('tkt-1')
  })

  it('T1 remembers grant_ticket from search by repo gitsite', () => {
    const repo = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git'
    const ticket = rememberGrantTicketFromSearch(
      `?gitlab=ok&grant_ticket=tkt-create-1&repo_url=${encodeURIComponent(repo)}`,
    )
    expect(ticket).toBe('tkt-create-1')
    expect(sessionGrantTicketForRepo(repo)).toBe('tkt-create-1')
  })

  it('collects unique tickets for submitted repo URLs plus extras', () => {
    const repo = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git'
    rememberGrantTicketFromSearch(
      `?grant_ticket=tkt-create-1&repo_url=${encodeURIComponent(repo)}`,
    )
    expect(collectSessionGrantTickets([repo], ['tkt-create-1', ' '])).toEqual(['tkt-create-1'])
  })

  it('consumes a ticket after it was used once: site key and wildcard no longer pass the gate (OPT-20260902-025)', () => {
    const repo = 'https://github.com/acme/demo.git'
    rememberGrantTicket('tkt-single', repo)
    expect(hasSessionGrantForRepo(repo)).toBe(true)
    consumeSessionGrantTicket('tkt-single')
    expect(hasSessionGrantForRepo(repo)).toBe(false)
    expect(sessionGrantTicketForRepo(repo)).toBe('')
    expect(sessionGrantTicketAny()).toBe('')
  })

  it('consuming one ticket leaves an unrelated pending ticket usable (OPT-20260902-025)', () => {
    const gh = 'https://github.com/acme/gh.git'
    const gl = 'https://gitlab.daydaymoney.com/acme/gl.git'
    rememberGrantTicket('tkt-gh', gh)
    rememberGrantTicket('tkt-gl', gl)
    // tkt-gl 是最新 → 通配指向 tkt-gl；tkt-gh 已消费
    consumeSessionGrantTicket('tkt-gh')
    expect(hasSessionGrantForRepo(gh)).toBe(false)
    expect(sessionGrantTicketForRepo(gh)).toBe('')
    expect(hasSessionGrantForRepo(gl)).toBe(true)
    expect(sessionGrantTicketForRepo(gl)).toBe('tkt-gl')
  })

  it('consumeSessionGrantTickets removes a batch of used tickets (OPT-20260902-025)', () => {
    const gh = 'https://github.com/acme/gh.git'
    const gl = 'https://gitlab.daydaymoney.com/acme/gl.git'
    rememberGrantTicket('tkt-gh', gh)
    rememberGrantTicket('tkt-gl', gl)
    consumeSessionGrantTickets(['tkt-gh', 'tkt-gl'])
    expect(hasSessionGrantForRepo(gh)).toBe(false)
    expect(hasSessionGrantForRepo(gl)).toBe(false)
    expect(sessionGrantTicketAny()).toBe('')
  })
})
