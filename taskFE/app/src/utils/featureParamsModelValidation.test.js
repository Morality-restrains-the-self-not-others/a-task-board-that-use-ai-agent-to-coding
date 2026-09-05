/**
 * 功能参数：模型名称须在供应商支持列表内（前端共享校验）
 */
import { describe, it, expect } from 'vitest'
import {
  normalizeSupportedModelsText,
  parseSupportedModels,
  validateModelsAgainstProviders,
} from './featureParamsModelValidation.js'

const providers = [
  {
    provider: 'openai',
    supported_models: ['gpt-4', 'gpt-4.1'],
  },
]

describe('validateModelsAgainstProviders', () => {
  it('accepts agent model in supported list', () => {
    expect(
      validateModelsAgainstProviders(providers, {
        agentModel: 'gpt-4',
        agentModelProvider: 'openai',
        summaryModel: '',
        summaryModelProvider: '',
      }),
    ).toBeNull()
  })

  it('rejects agent model not in supported list', () => {
    const err = validateModelsAgainstProviders(providers, {
      agentModel: 'not-a-real-model',
      agentModelProvider: 'openai',
      summaryModel: '',
      summaryModelProvider: '',
    })
    expect(err).toContain('not-a-real-model')
    expect(err).toContain('openai')
  })

  it('allows empty model names', () => {
    expect(
      validateModelsAgainstProviders(providers, {
        agentModel: '',
        agentModelProvider: 'openai',
        summaryModel: '',
        summaryModelProvider: '',
      }),
    ).toBeNull()
  })

  it('rejects summary model not in list', () => {
    const err = validateModelsAgainstProviders(providers, {
      agentModel: '',
      agentModelProvider: '',
      summaryModel: 'fake-summary',
      summaryModelProvider: 'openai',
    })
    expect(err).toContain('fake-summary')
  })

  it('parses supported_models_text', () => {
    const withText = [
      { provider: 'openai', supported_models_text: 'gpt-4\ngpt-4.1' },
    ]
    expect(
      validateModelsAgainstProviders(withText, {
        agentModel: 'gpt-4.1',
        agentModelProvider: 'openai',
        summaryModel: '',
        summaryModelProvider: '',
      }),
    ).toBeNull()
  })

  it('parses Chinese comma and mixed separators in supported_models_text', () => {
    expect(parseSupportedModels('gpt-4.1，gpt-4.1-mini')).toEqual([
      'gpt-4.1',
      'gpt-4.1-mini',
    ])
    expect(parseSupportedModels('gpt-4.1, gpt-4.1-mini；o3-mini')).toEqual([
      'gpt-4.1',
      'gpt-4.1-mini',
      'o3-mini',
    ])
    const withText = [
      { provider: 'openai', supported_models_text: 'gpt-4.1，gpt-4.1-mini' },
    ]
    expect(
      validateModelsAgainstProviders(withText, {
        agentModel: 'gpt-4.1-mini',
        agentModelProvider: 'openai',
        summaryModel: '',
        summaryModelProvider: '',
      }),
    ).toBeNull()
  })

  it('normalizeSupportedModelsText rewrites separators to newlines', () => {
    expect(normalizeSupportedModelsText('gpt-4.1，gpt-4.1-mini；o3-mini')).toBe(
      'gpt-4.1\ngpt-4.1-mini\no3-mini',
    )
  })
})
