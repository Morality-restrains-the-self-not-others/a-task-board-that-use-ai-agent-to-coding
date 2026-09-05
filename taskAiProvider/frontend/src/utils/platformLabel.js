/**
 * Cloud platform type → display name mapping, shared across VendorPortal and AdminPortal.
 * Single source of truth; keep in sync with Go backend infrastructure.cloudPlatformDisplay.
 */
export const PLATFORM_DISPLAY_NAMES = {
  aliyun: "阿里云",
  tencentcloud: "腾讯云",
  huaweicloud: "华为云",
  ctyun: "天翼云",
  cmcc: "移动云",
  cucloud: "联通云",
  baiducloud: "百度智能云",
  aws: "AWS",
};

/** Array form for v-for / select options in VendorPortal. */
export const CLOUD_PLATFORMS = Object.entries(PLATFORM_DISPLAY_NAMES).map(
  ([value, label]) => ({ value, label }),
);

/**
 * Resolve a platform_type value (e.g. "aliyun") to its Chinese display name.
 * Returns the raw input when the platform is unrecognized.
 * @param {string} platformType
 * @returns {string}
 */
export function platformLabel(platformType) {
  if (!platformType) return "—";
  return PLATFORM_DISPLAY_NAMES[platformType] || platformType;
}

/**
 * Convenience alias matching AdminPortal's naming convention.
 * @param {string} platformType
 * @returns {string}
 */
export function getPlatformDisplayName(platformType) {
  return platformLabel(platformType);
}
