import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, it } from 'node:test';

const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'config.mjs'), 'utf8');

describe('config.mjs yaml quote trim (javascript:S5850)', () => {
  it('groups leading/trailing quote alternatives', () => {
    assert.ok(src.includes('replace(/(?:^["\']|["\']$)/g'), src);
  });
  it('overlays conf-local instead of config.local.yaml (ADR-0054)', () => {
    assert.ok(src.includes("conf-local"), src);
    assert.equal(src.includes('config.local.yaml'), false, src);
    assert.ok(src.includes('taskSSE'), src);
  });
});
