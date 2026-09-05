import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import {
  CLOUD_PLATFORMS,
  PLATFORM_DISPLAY_NAMES,
  platformLabel,
  getPlatformDisplayName,
} from '../src/utils/platformLabel.js';

describe('PLATFORM_DISPLAY_NAMES', () => {
  it('covers all 8 cloud platforms', () => {
    const keys = Object.keys(PLATFORM_DISPLAY_NAMES);
    assert.equal(keys.length, 8);
  });

  it('maps aliyun to 阿里云', () => {
    assert.equal(PLATFORM_DISPLAY_NAMES['aliyun'], '阿里云');
  });

  it('maps tencentcloud to 腾讯云', () => {
    assert.equal(PLATFORM_DISPLAY_NAMES['tencentcloud'], '腾讯云');
  });

  it('maps huaweicloud to 华为云', () => {
    assert.equal(PLATFORM_DISPLAY_NAMES['huaweicloud'], '华为云');
  });

  it('maps aws to AWS', () => {
    assert.equal(PLATFORM_DISPLAY_NAMES['aws'], 'AWS');
  });
});

describe('CLOUD_PLATFORMS', () => {
  it('returns array of {value, label} matching PLATFORM_DISPLAY_NAMES', () => {
    assert.equal(CLOUD_PLATFORMS.length, 8);
    for (const item of CLOUD_PLATFORMS) {
      assert.ok(typeof item.value === 'string');
      assert.ok(typeof item.label === 'string');
      assert.equal(item.label, PLATFORM_DISPLAY_NAMES[item.value]);
    }
  });
});

describe('platformLabel', () => {
  it('returns Chinese display name for aliyun', () => {
    assert.equal(platformLabel('aliyun'), '阿里云');
  });

  it('returns Chinese display name for tencentcloud', () => {
    assert.equal(platformLabel('tencentcloud'), '腾讯云');
  });

  it('returns Chinese display name for huaweicloud', () => {
    assert.equal(platformLabel('huaweicloud'), '华为云');
  });

  it('returns Chinese display name for ctyun', () => {
    assert.equal(platformLabel('ctyun'), '天翼云');
  });

  it('returns Chinese display name for cmcc', () => {
    assert.equal(platformLabel('cmcc'), '移动云');
  });

  it('returns Chinese display name for cucloud', () => {
    assert.equal(platformLabel('cucloud'), '联通云');
  });

  it('returns Chinese display name for baiducloud', () => {
    assert.equal(platformLabel('baiducloud'), '百度智能云');
  });

  it('returns AWS for aws', () => {
    assert.equal(platformLabel('aws'), 'AWS');
  });

  it('returns raw platform type for unrecognized value', () => {
    assert.equal(platformLabel('unknown-cloud'), 'unknown-cloud');
  });

  it('returns em-dash for empty string', () => {
    assert.equal(platformLabel(''), '—');
  });

  it('returns em-dash for null/undefined', () => {
    assert.equal(platformLabel(null), '—');
    assert.equal(platformLabel(undefined), '—');
  });
});

describe('getPlatformDisplayName', () => {
  it('is an alias for platformLabel', () => {
    assert.equal(getPlatformDisplayName('aliyun'), '阿里云');
    assert.equal(getPlatformDisplayName('tencentcloud'), '腾讯云');
    assert.equal(getPlatformDisplayName(''), '—');
  });
});
