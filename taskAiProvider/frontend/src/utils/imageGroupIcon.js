/** Image-group icon upload + form validation (required PNG/JPEG/WEBP ≤512KB). */

import { uploadIssuedFile } from "./directUpload.js";

export {
  buildDirectUploadHeaders,
  isSameOriginUploadURL,
} from "./directUpload.js";

export const IMAGE_GROUP_ICON_ACCEPT = "image/png,image/jpeg,image/webp";
export const MAX_IMAGE_GROUP_ICON_BYTES = 512 * 1024;

export function validateImageGroupFields({
  name,
  description,
  iconFileKey,
  iconFile,
}) {
  if (!(name || "").trim()) {
    return "镜像组名称必填";
  }
  if (!(description || "").trim()) {
    return "镜像组描述必填";
  }
  if (!iconFile && !(iconFileKey || "").trim()) {
    return "镜像组图标必填";
  }
  if (iconFile) {
    const err = validateImageGroupIconFile(iconFile);
    if (err) return err;
  }
  return "";
}

export function validateImageGroupIconFile(file) {
  if (!file) {
    return "镜像组图标必填";
  }
  if (file.size <= 0 || file.size > MAX_IMAGE_GROUP_ICON_BYTES) {
    return "图标须为 PNG/JPEG/WEBP 且不超过 512KB";
  }
  const name = String(file.name || "").toLowerCase();
  const okExt =
    name.endsWith(".png") ||
    name.endsWith(".jpg") ||
    name.endsWith(".jpeg") ||
    name.endsWith(".webp");
  const type = String(file.type || "").toLowerCase();
  const okType =
    type === "image/png" || type === "image/jpeg" || type === "image/webp";
  if (!okExt && !okType) {
    return "图标须为 PNG/JPEG/WEBP 且不超过 512KB";
  }
  return "";
}

export function imageGroupIconSrc(group) {
  if (!group || typeof group !== "object") {
    return "";
  }
  const url = group.icon_url || group.image_group?.icon_url || "";
  return typeof url === "string" ? url.trim() : "";
}

export async function uploadImageGroupIcon(file, { api, fetchImpl } = {}) {
  const err = validateImageGroupIconFile(file);
  if (err) {
    throw new Error(err);
  }
  const post = api;
  const issued = await post("/api/vendor/image-groups/icon-upload-url/", {
    method: "POST",
    body: JSON.stringify({
      filename: file.name,
      content_type: file.type,
      size: file.size,
    }),
  });
  const token =
    typeof localStorage !== "undefined"
      ? localStorage.getItem("vendor_token")
      : "";
  await uploadIssuedFile(file, issued, {
    fetchImpl,
    bearerToken: token,
  });
  await post("/api/vendor/image-groups/icon-upload-complete/", {
    method: "POST",
    body: JSON.stringify({ file_key: issued.file_key }),
  });
  return issued.file_key || "";
}
