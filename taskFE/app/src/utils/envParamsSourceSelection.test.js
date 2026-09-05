import { describe, expect, it } from 'vitest'
import {
  buildFeatureParamsPayloadFields,
  featureParamsSettingsHrefs,
  normalizeFeatureParamsSourceForSelect,
  readEnvVarSourcesAvailableFlag,
  resolveCreateTaskFeatureParamsBlockedReason,
  resolveEnvParamsSourceRequiredHint,
} from './envParamsSourceSelection.js'

describe('resolveEnvParamsSourceRequiredHint', () => {
  it('空来源时提示先选择智能体资源配置', () => {
    expect(resolveEnvParamsSourceRequiredHint({ featureParamsSource: '' }))
      .toBe('请先选择智能体资源配置')
    expect(resolveEnvParamsSourceRequiredHint({}))
      .toBe('请先选择智能体资源配置')
  })

  it('个人配置未选具体配置时同样提示', () => {
    expect(resolveEnvParamsSourceRequiredHint({
      featureParamsSource: 'personal',
      selectedPersonalConfigId: '',
    })).toBe('请先选择智能体资源配置')
  })

  it('公司/工作空间或已选个人配置时无提示', () => {
    expect(resolveEnvParamsSourceRequiredHint({ featureParamsSource: 'company' })).toBe('')
    expect(resolveEnvParamsSourceRequiredHint({ featureParamsSource: 'workspace' })).toBe('')
    expect(resolveEnvParamsSourceRequiredHint({
      featureParamsSource: 'personal',
      selectedPersonalConfigId: 'cfg-1',
    })).toBe('')
  })
})

describe('resolveCreateTaskFeatureParamsBlockedReason', () => {
  it('未选时按创建/编辑返回不同文案', () => {
    expect(resolveCreateTaskFeatureParamsBlockedReason({
      featureParamsSource: '',
      isEdit: false,
    })).toBe('请先选择智能体资源配置后再创建')
    expect(resolveCreateTaskFeatureParamsBlockedReason({
      featureParamsSource: '',
      isEdit: true,
    })).toBe('请先选择智能体资源配置后再保存')
  })

  it('已选时返回空', () => {
    expect(resolveCreateTaskFeatureParamsBlockedReason({
      featureParamsSource: 'company',
      isEdit: false,
    })).toBe('')
  })
})

describe('normalizeFeatureParamsSourceForSelect', () => {
  it('将 none/空/未知规范为空字符串', () => {
    expect(normalizeFeatureParamsSourceForSelect('')).toBe('')
    expect(normalizeFeatureParamsSourceForSelect('none')).toBe('')
    expect(normalizeFeatureParamsSourceForSelect('other')).toBe('')
    expect(normalizeFeatureParamsSourceForSelect(null)).toBe('')
  })

  it('保留合法来源', () => {
    expect(normalizeFeatureParamsSourceForSelect('company')).toBe('company')
    expect(normalizeFeatureParamsSourceForSelect('workspace')).toBe('workspace')
    expect(normalizeFeatureParamsSourceForSelect('personal')).toBe('personal')
  })
})

describe('buildFeatureParamsPayloadFields', () => {
  it('未选来源时不写入字段', () => {
    expect(buildFeatureParamsPayloadFields({})).toEqual({})
    expect(buildFeatureParamsPayloadFields({ feature_params_source: 'none' })).toEqual({})
  })

  it('公司/工作空间写入 source', () => {
    expect(buildFeatureParamsPayloadFields({ feature_params_source: 'company' }))
      .toEqual({ feature_params_source: 'company' })
    expect(buildFeatureParamsPayloadFields({ feature_params_source: 'workspace' }))
      .toEqual({ feature_params_source: 'workspace' })
  })

  it('个人配置须同时有 config id', () => {
    expect(buildFeatureParamsPayloadFields({ feature_params_source: 'personal' })).toEqual({})
    expect(buildFeatureParamsPayloadFields({
      feature_params_source: 'personal',
      personal_feature_params_config_id: 'cfg-9',
    })).toEqual({
      feature_params_source: 'personal',
      personal_feature_params_config_id: 'cfg-9',
    })
  })
})

describe('readEnvVarSourcesAvailableFlag', () => {
  it('读取真实响应 data.env_var_sources_available', () => {
    expect(readEnvVarSourcesAvailableFlag({
      data: { env_var_sources_available: { company: true, workspace: false } },
    })).toEqual({ company: true, workspace: false })
  })

  it('兼容顶层 flag', () => {
    expect(readEnvVarSourcesAvailableFlag({
      env_var_sources_available: { company: false, workspace: true },
    })).toEqual({ company: false, workspace: true })
  })

  it('优先 data 内标志', () => {
    expect(readEnvVarSourcesAvailableFlag({
      env_var_sources_available: { company: false, workspace: false },
      data: { env_var_sources_available: { company: true, workspace: false } },
    })).toEqual({ company: true, workspace: false })
  })

  it('缺标志返回 null', () => {
    expect(readEnvVarSourcesAvailableFlag(null)).toBe(null)
    expect(readEnvVarSourcesAvailableFlag({})).toBe(null)
    expect(readEnvVarSourcesAvailableFlag({ data: { extra_env_vars: [] } })).toBe(null)
  })
})

describe('featureParamsSettingsHrefs', () => {
  it('按租户拼公司/工作空间设置页，个人为 /profile/feature-params/', () => {
    expect(featureParamsSettingsHrefs('875561774391259136')).toEqual({
      company: '/tenant/875561774391259136/settings/feature-params/',
      workspace: '/tenant/875561774391259136/settings/task-panel/',
      personal: '/profile/feature-params/',
    })
  })

  it('无租户时公司/工作空间 href 为空，个人链接仍可用', () => {
    expect(featureParamsSettingsHrefs('')).toEqual({
      company: '',
      workspace: '',
      personal: '/profile/feature-params/',
    })
  })
})
