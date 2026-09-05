export function normalizeArchitecture(value) {
  const normalized = String(value || "").trim().toLowerCase().replace(/-/g, "_");
  if (!normalized) return "";
  if (["x86", "x64", "x86_64", "amd64"].includes(normalized)) return "x86_64";
  if (["arm", "arm64", "aarch64"].includes(normalized)) return "arm64";
  return normalized;
}

export function serverImageMatchesRegion(serverImage, region) {
  const serverRegion = String(serverImage?.region ?? "").trim().toLowerCase();
  const regionId = String(region?.id ?? "").trim().toLowerCase();
  const regionName = String(region?.name ?? "").trim().toLowerCase();
  if (!serverRegion) return false;
  return serverRegion === regionId || serverRegion === regionName;
}

export function serverImageMatchesArch(serverImage, targetArchs) {
  const archs = (targetArchs || []).map((a) => normalizeArchitecture(a)).filter(Boolean);
  if (archs.length === 0) return true;
  const sArch = normalizeArchitecture(serverImage?.architecture);
  return !sArch || archs.includes(sArch);
}

/** 区域运行环境弹窗：列出同云平台已登记的全部镜像，匹配项优先排序。 */
export function buildRegionEnvServerOptions(serverImages, { platform, region, targetArchs, selectedId }) {
  const source = Array.isArray(serverImages) ? serverImages : [];
  if (!platform || !region) return [];

  const filtered = source.filter((s) => {
    if (s.platform_type !== platform) return false;
    if (s.is_active === false) return false;
    return true;
  });

  const sorted = [...filtered].sort((a, b) => {
    const aRegion = serverImageMatchesRegion(a, region) ? 0 : 1;
    const bRegion = serverImageMatchesRegion(b, region) ? 0 : 1;
    if (aRegion !== bRegion) return aRegion - bRegion;
    const aArch = serverImageMatchesArch(a, targetArchs) ? 0 : 1;
    const bArch = serverImageMatchesArch(b, targetArchs) ? 0 : 1;
    if (aArch !== bArch) return aArch - bArch;
    return String(a.image_name || "").localeCompare(String(b.image_name || ""), "zh-CN");
  });

  if (!selectedId) return sorted;
  const selected = source.find((x) => String(x.id) === String(selectedId));
  if (!selected || sorted.some((x) => String(x.id) === String(selectedId))) {
    return sorted;
  }
  return [...sorted, selected];
}
