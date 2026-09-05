// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import {
  createDatalistDismissController,
  isDatalistPickerInputEvent,
  shouldDismissDatalistPick,
} from './datalistDismiss.js'

describe('shouldDismissDatalistPick', () => {
  it('候选精确匹配时返回 true', () => {
    expect(shouldDismissDatalistPick(['main', 'develop'], 'develop')).toBe(true)
  })

  it('非候选或空值返回 false', () => {
    expect(shouldDismissDatalistPick(['main'], 'custom')).toBe(false)
    expect(shouldDismissDatalistPick(['main'], '')).toBe(false)
    expect(shouldDismissDatalistPick([], 'main')).toBe(false)
  })
})

describe('isDatalistPickerInputEvent', () => {
  it('识别 datalist 选中产生的 input', () => {
    expect(isDatalistPickerInputEvent({ type: 'input', inputType: 'insertReplacementText' })).toBe(true)
    expect(isDatalistPickerInputEvent({ type: 'input', inputType: 'insertFromDrop' })).toBe(true)
  })

  it('普通逐字输入与非 input 事件返回 false', () => {
    expect(isDatalistPickerInputEvent({ type: 'input', inputType: 'insertText' })).toBe(false)
    expect(isDatalistPickerInputEvent({ type: 'change' })).toBe(false)
    expect(isDatalistPickerInputEvent(null)).toBe(false)
  })
})

describe('createDatalistDismissController', () => {
  it('选中候选后抑制 list、同步摘掉 DOM list 并失焦，focus 后恢复 list', () => {
    const { listAttr, dismissIfPicked, restoreOnFocus } = createDatalistDismissController()
    const input = {
      id: 'task-base-branch-0-0',
      blur: vi.fn(),
      removeAttribute: vi.fn(),
    }

    expect(listAttr(input.id, 'task-base-branch-options-0-0')).toBe('task-base-branch-options-0-0')

    expect(dismissIfPicked(input, ['master', 'develop'], 'master', { type: 'change' })).toBe(true)
    expect(input.removeAttribute).toHaveBeenCalledWith('list')
    expect(input.blur).toHaveBeenCalled()
    expect(listAttr(input.id, 'task-base-branch-options-0-0')).toBeUndefined()

    restoreOnFocus(input)
    expect(listAttr(input.id, 'task-base-branch-options-0-0')).toBe('task-base-branch-options-0-0')
  })

  it('input 为 datalist 选中时立即 dismiss，逐字输入不 dismiss', () => {
    const { listAttr, dismissIfPicked } = createDatalistDismissController()
    const input = {
      id: 'task-base-branch-0-0',
      blur: vi.fn(),
      removeAttribute: vi.fn(),
    }

    expect(
      dismissIfPicked(input, ['main'], 'main', { type: 'input', inputType: 'insertText' }),
    ).toBe(false)
    expect(input.blur).not.toHaveBeenCalled()
    expect(listAttr(input.id, 'task-base-branch-options-0-0')).toBe('task-base-branch-options-0-0')

    expect(
      dismissIfPicked(input, ['main'], 'main', { type: 'input', inputType: 'insertReplacementText' }),
    ).toBe(true)
    expect(input.removeAttribute).toHaveBeenCalledWith('list')
    expect(input.blur).toHaveBeenCalled()
  })

  it('手动输入非候选时不抑制 list、不失焦', () => {
    const { listAttr, dismissIfPicked } = createDatalistDismissController()
    const input = { id: 'merge-target-name', blur: vi.fn(), removeAttribute: vi.fn() }

    expect(dismissIfPicked(input, ['main'], 'my-custom-branch', { type: 'change' })).toBe(false)
    expect(input.blur).not.toHaveBeenCalled()
    expect(input.removeAttribute).not.toHaveBeenCalled()
    expect(listAttr(input.id, 'merge-target-preset-options')).toBe('merge-target-preset-options')
  })
})
