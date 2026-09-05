import { describe, expect, it } from 'vitest'
import {
  CREATE_TASK_FIELD_SETTING_KEYS,
  CREATE_TASK_FIELD_SETTING_LABELS,
  createTaskFieldSettingsFromResponse,
  defaultCreateTaskFieldSettings,
  isCreateTaskFieldEnabled,
  normalizeCreateTaskFieldSettings,
} from './createTaskFieldSettings.js'

describe('createTaskFieldSettings', () => {
  it('defaults known keys; code_lang and structured_fields off', () => {
    const d = defaultCreateTaskFieldSettings()
    expect(Object.keys(d).sort()).toEqual([...CREATE_TASK_FIELD_SETTING_KEYS].sort())
    expect(d.code_lang).toBe(false)
    expect(d.structured_fields).toBe(false)
    expect(
      CREATE_TASK_FIELD_SETTING_KEYS.filter((k) => k !== 'code_lang' && k !== 'structured_fields').every(
        (k) => d[k] === true,
      ),
    ).toBe(true)
  })

  it('normalizes partial and coerced values; drops unknown keys', () => {
    const out = normalizeCreateTaskFieldSettings({
      priority: false,
      due_date: 'false',
      auto_run: 0,
      task_kind: 'yes',
      unknown: false,
    })
    expect(out.priority).toBe(false)
    expect(out.due_date).toBe(false)
    expect(out.auto_run).toBe(false)
    expect(out.task_kind).toBe(true)
    expect(out.owner).toBe(true)
    expect(out.code_lang).toBe(false)
    expect(out.structured_fields).toBe(false)
    expect(out).not.toHaveProperty('unknown')
  })

  it('parses API response fields', () => {
    expect(createTaskFieldSettingsFromResponse({ fields: { assignees: false } }).assignees).toBe(false)
    expect(createTaskFieldSettingsFromResponse(null).priority).toBe(true)
    expect(createTaskFieldSettingsFromResponse(null).code_lang).toBe(false)
    expect(createTaskFieldSettingsFromResponse(null).structured_fields).toBe(false)
  })

  it('isCreateTaskFieldEnabled treats missing as enabled', () => {
    expect(isCreateTaskFieldEnabled(null, 'priority')).toBe(true)
    expect(isCreateTaskFieldEnabled({ priority: false }, 'priority')).toBe(false)
    expect(isCreateTaskFieldEnabled({ priority: false }, 'owner')).toBe(true)
  })

  it('feature_params 字段标签为智能体资源配置', () => {
    expect(CREATE_TASK_FIELD_SETTING_LABELS.feature_params).toBe('智能体资源配置')
  })
})
