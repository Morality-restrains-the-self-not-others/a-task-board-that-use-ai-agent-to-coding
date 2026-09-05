import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import {
  PRIVATE_REGISTRY_HINT,
  isCloudVendorPrivateRegistryText,
  isPrivateRegistryHost,
  suggestPublicAliyunAcr,
  enrichResolveArchitectureError,
} from '../src/utils/privateRegistryHint.js';

// Shared host classification table — single source of truth also consumed by the
// Go test (shareLib/registryhost/registryhost_test.go). Keep both in sync by
// editing testdata/registry_cases.json, never one side only.
const SHARED_CASES_URL = new URL(
  '../../../shareLib/registryhost/testdata/registry_cases.json',
  import.meta.url,
);

function loadSharedCases() {
  const raw = JSON.parse(readFileSync(SHARED_CASES_URL, 'utf8'));
  return raw.cases;
}

describe('shared registry case table (Go shareLib/registryhost <-> JS privateRegistryHint.js)', () => {
  const cases = loadSharedCases();

  it('host-level classification matches the shared table', () => {
    assert.ok(cases.length > 0, 'shared case table must not be empty');
    for (const tc of cases) {
      assert.equal(
        isPrivateRegistryHost(tc.registry),
        tc.want_private,
        `isPrivateRegistryHost(${JSON.stringify(tc.registry)}) should be ${tc.want_private}`,
      );
    }
  });

  it('Aliyun public suggestion matches the shared table', () => {
    for (const tc of cases) {
      assert.equal(
        suggestPublicAliyunAcr(tc.registry),
        tc.want_public_suggestion || '',
        `suggestPublicAliyunAcr(${JSON.stringify(tc.registry)}) should be ${JSON.stringify(
          tc.want_public_suggestion || '',
        )}`,
      );
    }
  });
});

describe('isCloudVendorPrivateRegistryText', () => {
  it('detects Aliyun ACR VPC hostname from the live timeout error', () => {
    const err =
      'Get "https://registry-vpc.cn-qingdao.aliyuncs.com/v2/ruandao/task2app-trae/manifests/x86_64-latest": context deadline exceeded';
    assert.equal(isCloudVendorPrivateRegistryText(err), true);
  });

  it('detects Aliyun ACR VPC image URL', () => {
    assert.equal(
      isCloudVendorPrivateRegistryText(
        'registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest',
      ),
      true,
    );
  });

  it('detects RFC1918 registry hosts', () => {
    assert.equal(isCloudVendorPrivateRegistryText('10.0.1.8:5000/app:latest'), true);
    assert.equal(isCloudVendorPrivateRegistryText('http://192.168.1.10/v2/'), true);
  });

  it('does not flag public Aliyun ACR or Docker Hub', () => {
    assert.equal(
      isCloudVendorPrivateRegistryText('registry.cn-qingdao.aliyuncs.com/ruandao/app:latest'),
      false,
    );
    assert.equal(isCloudVendorPrivateRegistryText('docker.io/library/nginx:latest'), false);
    assert.equal(isCloudVendorPrivateRegistryText('ghcr.io/org/app:1'), false);
  });
});

describe('enrichResolveArchitectureError', () => {
  it('appends the intranet hint for the Aliyun VPC timeout without duplicating it', () => {
    const raw =
      'Get "https://registry-vpc.cn-qingdao.aliyuncs.com/v2/ruandao/task2app-trae/manifests/x86_64-latest": context deadline exceeded';
    const once = enrichResolveArchitectureError(raw, {
      image_url: 'registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest',
    });
    assert.ok(once.includes(PRIVATE_REGISTRY_HINT));
    assert.ok(once.includes('registry.cn-qingdao.aliyuncs.com'));
    const twice = enrichResolveArchitectureError(once, {
      image_url: 'registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest',
    });
    assert.equal(twice, once);
  });

  it('api.js enriches resolve-target-architectures failures', () => {
    const src = readFileSync(new URL('../src/api.js', import.meta.url), 'utf8');
    assert.ok(src.includes('enrichResolveArchitectureError'));
    assert.ok(src.includes('resolve-target-architectures'));
  });

  it('leaves public-registry errors unchanged', () => {
    const raw = 'registry status 401';
    assert.equal(
      enrichResolveArchitectureError(raw, { image_url: 'docker.io/library/nginx:latest' }),
      raw,
    );
  });
});
