import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import {
  extractCPUArchitecturesFromText,
  formatContainerImageArchitectures,
  formatContainerImageArchitectureSuffix,
  formatImageRequiredArch,
  formatInstanceSupportedArch,
  inferInstanceArchitecture,
  instanceCompatibleWithImageArchitectures,
  normalizeArchitecture,
  resolvePrimaryImageArchitecture,
} from './containerImageArchitecture.js'

// Shared CPU architecture case table — single source of truth also consumed by
// taskProjectService (Go) and taskCloudService (Go) tests. Keep all three in
// sync by editing shareLib/archfixture/testdata/instance_architecture_cases.json,
// never one side only (OPT-20260821-007).
const SHARED_ARCH_CASES_URL = new URL(
  '../../../../shareLib/archfixture/testdata/instance_architecture_cases.json',
  import.meta.url,
)

function loadSharedArchCases() {
  return JSON.parse(readFileSync(SHARED_ARCH_CASES_URL, 'utf8'))
}

describe('containerImageArchitecture', () => {
  it('normalizeArchitecture maps common aliases', () => {
    expect(normalizeArchitecture('amd64')).toBe('x86_64')
    expect(normalizeArchitecture('aarch64')).toBe('arm64')
    expect(normalizeArchitecture('')).toBe('')
  })

  it('resolvePrimaryImageArchitecture prefers selected installed image over task.container_image', () => {
    expect(
      resolvePrimaryImageArchitecture({
        task: { container_image: { target_architectures: ['arm64'] } },
        selectedImageId: 'img-1',
        installedImages: [{ id: 'img-1', target_architectures: ['x86_64'] }],
      }),
    ).toBe('x86_64')
  })

  it('inferInstanceArchitecture does not treat ecs.r6 as arm', () => {
    expect(inferInstanceArchitecture('ecs.r6.xlarge')).toBe('x86_64')
    expect(inferInstanceArchitecture('ecs.r6r.xlarge')).toBe('arm64')
    expect(instanceCompatibleWithImageArchitectures('ecs.g7.xlarge', ['x86_64'])).toBe(true)
    expect(instanceCompatibleWithImageArchitectures('ecs.g7.xlarge', ['arm64'])).toBe(false)
  })

  it('resolvePrimaryImageArchitecture falls back to installed image', () => {
    expect(
      resolvePrimaryImageArchitecture({
        selectedImageId: 'img-1',
        installedImages: [{ id: 'img-1', target_architectures: ['x86_64', 'arm64'] }],
      }),
    ).toBe('x86_64')
  })

  it('formatContainerImageArchitectures deduplicates aliases', () => {
    expect(formatContainerImageArchitectures({ target_architectures: ['amd64', 'x86_64'] })).toBe('x86_64')
    expect(formatContainerImageArchitectureSuffix({ target_architectures: ['arm64'] })).toBe(' [arm64]')
  })

  it('drops unknown tokens and infers ISA from version when declared list is empty', () => {
    expect(formatContainerImageArchitectures({
      target_architectures: ['x86_64', 'unknown'],
    })).toBe('x86_64')
    expect(formatContainerImageArchitectures({
      target_architectures: [],
      version: 'private_x86_64-latest',
      name: 'trae-agent',
    })).toBe('x86_64')
    expect(instanceCompatibleWithImageArchitectures('ecs.c6.large', {
      target_architectures: [],
      version: 'private_x86_64-latest',
    })).toBe(true)
  })

  it('formatImageRequiredArch and formatInstanceSupportedArch name both sides', () => {
    expect(formatImageRequiredArch({ target_architectures: [] })).toBe('未声明')
    expect(formatImageRequiredArch({ target_architectures: ['unknown'], version: 'latest' }))
      .toBe('unknown（无法识别为 x86_64/arm64）')
    expect(formatImageRequiredArch({ target_architectures: [], version: 'private_x86_64-latest' }))
      .toBe('x86_64')
    expect(formatInstanceSupportedArch('ecs.c6.large')).toBe('x86_64（实例规格 ecs.c6.large）')
  })

  it('extractCPUArchitecturesFromText matches shared case table (OPT-20260821-007)', () => {
    const fx = loadSharedArchCases()
    expect(fx.extract_from_text_cases.length).toBeGreaterThan(0)
    for (const tc of fx.extract_from_text_cases) {
      expect(extractCPUArchitecturesFromText(...tc.inputs), tc.name).toEqual(tc.want)
    }
  })

  it('inferInstanceArchitecture matches shared case table (OPT-20260821-007)', () => {
    const fx = loadSharedArchCases()
    expect(fx.infer_instance_architecture_cases.length).toBeGreaterThan(0)
    for (const tc of fx.infer_instance_architecture_cases) {
      expect(inferInstanceArchitecture(tc.instance_type), tc.instance_type).toBe(tc.want)
    }
  })
})
