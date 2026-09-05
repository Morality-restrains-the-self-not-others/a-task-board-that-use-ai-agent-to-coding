import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { buildRegionEnvAssociationPayload } from '../src/utils/regionEnvAssociationPayload.js';

describe('buildRegionEnvAssociationPayload', () => {
  it('uses cloud_server_image_id (not legacy cloud_server_image)', () => {
    const payload = buildRegionEnvAssociationPayload({
      platformType: 'aliyun',
      regionId: 'cn-hongkong',
      cloudServerImageId: '862586045524725760',
      userdataTemplateId: '1',
    });
    assert.equal(payload.cloud_server_image_id, '862586045524725760');
    assert.equal(payload.platform_type, 'aliyun');
    assert.equal(payload.region, 'cn-hongkong');
    assert.equal(payload.userdata_template_id, '1');
    assert.equal(Object.prototype.hasOwnProperty.call(payload, 'cloud_server_image'), false);
  });

  it('omits userdata_template_id when clearing server image', () => {
    const payload = buildRegionEnvAssociationPayload({
      platformType: 'aliyun',
      regionId: 'cn-hongkong',
      cloudServerImageId: '',
    });
    assert.equal(payload.cloud_server_image_id, null);
    assert.equal(Object.prototype.hasOwnProperty.call(payload, 'userdata_template_id'), false);
  });
});
