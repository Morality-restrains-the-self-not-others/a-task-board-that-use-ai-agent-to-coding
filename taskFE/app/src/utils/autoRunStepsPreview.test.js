import { describe, expect, it } from 'vitest'
import { resolveAutoRunStepsPreview, shouldShowImageAutoRunSteps } from './autoRunStepsPreview.js'

describe('resolveAutoRunStepsPreview', () => {
  it('prefers live markdown', () => {
    const r = resolveAutoRunStepsPreview({
      markdown: 'cache',
      extractStatus: 'ok',
      liveMarkdown: '# live',
    })
    expect(r.markdown).toBe('# live')
    expect(r.source).toBe('live')
  })

  it('uses cache when live empty', () => {
    const r = resolveAutoRunStepsPreview({
      markdown: '# cached steps',
      extractStatus: 'ok',
      liveMarkdown: '',
    })
    expect(r.markdown).toContain('cached')
    expect(r.source).toBe('cache')
  })

  it('shows failure hint', () => {
    const r = resolveAutoRunStepsPreview({
      markdown: '',
      extractStatus: 'failed',
    })
    expect(r.failed).toBe(true)
    expect(r.emptyHint).toMatch(/抽取失败/)
  })

  it('shows not_found hint', () => {
    const r = resolveAutoRunStepsPreview({
      markdown: '',
      extractStatus: 'not_found',
    })
    expect(r.emptyHint).toMatch(/未找到/)
  })
})

describe('shouldShowImageAutoRunSteps', () => {
  it('hides empty and not_found catalog cards', () => {
    expect(shouldShowImageAutoRunSteps({ auto_run_steps_md: '', auto_run_steps_extract_status: '' })).toBe(false)
    expect(shouldShowImageAutoRunSteps({ auto_run_steps_md: '   ', auto_run_steps_extract_status: 'not_found' })).toBe(false)
    expect(shouldShowImageAutoRunSteps(null)).toBe(false)
  })

  it('shows markdown, pending extract, and failed extract', () => {
    expect(shouldShowImageAutoRunSteps({ auto_run_steps_md: '# steps', auto_run_steps_extract_status: 'ok' })).toBe(true)
    expect(shouldShowImageAutoRunSteps({ auto_run_steps_md: '', auto_run_steps_extract_status: 'pending' })).toBe(true)
    expect(shouldShowImageAutoRunSteps({ auto_run_steps_md: '', auto_run_steps_extract_status: 'failed' })).toBe(true)
    expect(shouldShowImageAutoRunSteps({ auto_run_steps_md: '', auto_run_steps_extract_status: 'auth_failed' })).toBe(true)
  })
})
