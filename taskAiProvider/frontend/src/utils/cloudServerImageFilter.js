export function filterCloudServerImagesByKeyword(images, keyword) {
  const normalizedKeyword = String(keyword || "").trim().toLowerCase();
  const source = Array.isArray(images) ? images : [];
  if (!normalizedKeyword) {
    return source;
  }
  return source.filter((image) => {
    const name = String(image?.name || "").toLowerCase();
    const id = String(image?.id || "").toLowerCase();
    const osType = String(image?.os_type || "").toLowerCase();
    const osVersion = String(image?.os_version || "").toLowerCase();
    const architecture = String(image?.architecture || "").toLowerCase();
    return [name, id, osType, osVersion, architecture].some((field) =>
      field.includes(normalizedKeyword),
    );
  });
}
