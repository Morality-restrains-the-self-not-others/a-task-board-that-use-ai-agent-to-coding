import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { formatDate } from '../src/utils/userDataTemplate.js';

describe('formatDate', () => {
  it('returns em-dash when created_at missing (admin list contract hole)', () => {
    assert.equal(formatDate(''), '—');
    assert.equal(formatDate(null), '—');
    assert.equal(formatDate(undefined), '—');
  });

  it('formats backend UTC timestamp with space separator', () => {
    const out = formatDate('2026-07-24 04:05:06.000000');
    assert.notEqual(out, '—');
    assert.notEqual(out, '');
    assert.match(out, /2026/);
  });

  it('returns em-dash for unparseable values', () => {
    assert.equal(formatDate('not-a-date'), '—');
  });
});
