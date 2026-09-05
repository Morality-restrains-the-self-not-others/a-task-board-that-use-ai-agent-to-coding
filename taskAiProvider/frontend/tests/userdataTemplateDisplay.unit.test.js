import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import {
  normalizeUserdataTemplate,
  userdataTemplateDisplayLabel,
} from '../src/utils/userdataTemplateDisplay.js';

describe('normalizeUserdataTemplate', () => {
  it('keeps nested object with id/name/version', () => {
    assert.deepEqual(normalizeUserdataTemplate({ id: 501, name: 'boot-linux', version: '3' }), {
      id: '501',
      name: 'boot-linux',
      version: '3',
    });
  });

  it('accepts bare id string from legacy list API', () => {
    assert.deepEqual(normalizeUserdataTemplate('501'), { id: '501' });
  });

  it('returns null for empty', () => {
    assert.equal(normalizeUserdataTemplate(null), null);
    assert.equal(normalizeUserdataTemplate(''), null);
  });
});

describe('userdataTemplateDisplayLabel', () => {
  it('shows name version when API enriched the association', () => {
    assert.equal(
      userdataTemplateDisplayLabel({ id: '501', name: 'boot-linux', version: '3' }),
      'boot-linux v3',
    );
  });

  it('resolves bare id via options so list does not stay as em-dash', () => {
    assert.equal(
      userdataTemplateDisplayLabel({ id: '501' }, [{ id: '501', name: 'boot-linux', version: '3' }]),
      'boot-linux v3',
    );
  });

  it('returns em-dash when template missing', () => {
    assert.equal(userdataTemplateDisplayLabel(null), '—');
  });
});
