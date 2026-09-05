'use strict';

const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const dir = __dirname;

describe('logs panel reflow (javascript:S905)', () => {
  it('09.js 用 getBoundingClientRect 触发重排而非空表达式 offsetHeight', () => {
    const src = fs.readFileSync(path.join(dir, '09.js'), 'utf8');
    assert.match(src, /panel\.getBoundingClientRect\(\)/);
    assert.doesNotMatch(src, /panel\.offsetHeight\s*;/);
  });

  it('13.js 用 getBoundingClientRect 触发重排而非空表达式 offsetHeight', () => {
    const src = fs.readFileSync(path.join(dir, '13.js'), 'utf8');
    assert.match(src, /panel\.getBoundingClientRect\(\)/);
    assert.doesNotMatch(src, /panel\.offsetHeight\s*;/);
  });
});
