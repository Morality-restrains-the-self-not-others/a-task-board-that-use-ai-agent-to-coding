/**
 * Build POST body for vendor container ↔ cloud-server-image association.
 * Canonical key is cloud_server_image_id (not nested cloud_server_image).
 */
export function buildRegionEnvAssociationPayload({ platformType, regionId, cloudServerImageId, userdataTemplateId }) {
  const payload = {
    platform_type: platformType,
    region: regionId,
    cloud_server_image_id: cloudServerImageId || null,
  };
  if (cloudServerImageId) {
    payload.userdata_template_id = userdataTemplateId || null;
  }
  return payload;
}
