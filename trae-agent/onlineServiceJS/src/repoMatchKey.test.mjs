import assert from 'node:assert/strict';
import test from 'node:test';

import { canonicalRepoKey, repoMatchKeyFromUrl } from './repoMatchKey.mjs';
import { repoMatchKeyFromUrl as repoMatchKeyFromHelpers } from './layerGitRouteHelpers.mjs';

test('repoMatchKeyFromUrl strips scheme and .git', () => {
  assert.equal(
    repoMatchKeyFromUrl('https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git'),
    'gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
  );
});

test('repoMatchKeyFromUrl equates ssh:// and git@ without ssh port', () => {
  assert.equal(
    repoMatchKeyFromUrl('ssh://git@github.com/acme/demo.git'),
    'github.com/acme/demo',
  );
  assert.equal(
    repoMatchKeyFromUrl('ssh://git@gitlab.daydaymoney.com:2222/g/p.git'),
    repoMatchKeyFromUrl('git@gitlab.daydaymoney.com:g/p.git'),
  );
});


test('layerGitRouteHelpers re-exports repoMatchKeyFromUrl', () => {
  assert.equal(
    repoMatchKeyFromHelpers('ssh://git@gitlab.daydaymoney.com:2222/g/p.git'),
    repoMatchKeyFromUrl('git@gitlab.daydaymoney.com:g/p.git'),
  );
});

test('canonicalRepoKey keeps scheme and lowercases', () => {
  assert.equal(
    canonicalRepoKey('https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git'),
    'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
  );
});
