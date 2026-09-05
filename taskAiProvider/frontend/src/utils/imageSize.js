export function formatFileSize(bytes) {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

export function parseSizeToBytes(sizeDisplay) {
  if (sizeDisplay === null || sizeDisplay === undefined) return "";
  const text = String(sizeDisplay).trim();
  if (!text) return "";
  const match = text.match(/^(\d+(?:\.\d+)?)\s*(B|BYTES|KB|MB|GB|TB)?$/i);
  if (!match) throw new Error("镜像大小格式不正确，例如 850 MB");
  const value = parseFloat(match[1]);
  const unit = (match[2] || "B").toUpperCase();
  const unitMap = { B: 1, BYTES: 1, KB: 1024, MB: 1024 ** 2, GB: 1024 ** 3, TB: 1024 ** 4 };
  const multiplier = unitMap[unit];
  if (!multiplier) throw new Error("单位仅支持 KB/MB/GB/TB");
  return String(Math.round(value * multiplier));
}
