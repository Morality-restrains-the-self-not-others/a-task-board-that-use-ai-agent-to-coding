export const SAAS_INBOUND_SKILL_VERSIONS_HREF =
  "/api/ai-provider/saas-inbound-skill-versions/";

export const SKILL_VERSION_REQUIRED_MESSAGE =
  "请选择已发布的容器→SaaS 接口版本";

export function normalizeSkillVersion(raw) {
  return String(raw ?? "")
    .trim()
    .replace(/^[vV]/, "")
    .trim();
}

export function writableSkillVersions(catalog) {
  const versions = Array.isArray(catalog?.versions) ? catalog.versions : [];
  return versions.filter(
    (v) => v && (v.status === "current" || v.status === "deprecated"),
  );
}

export function skillVersionOptionLabel(entry) {
  const n = normalizeSkillVersion(entry?.version);
  const summary = String(entry?.summary || "").trim();
  let tag = entry?.status || "";
  if (entry?.status === "current") tag = "当前";
  else if (entry?.status === "deprecated") tag = "仍支持";
  const head = n ? `v${n}` : "";
  if (summary) return `${head}（${tag}）— ${summary}`;
  return `${head}（${tag}）`;
}

export function requireWritableSkillVersion(raw) {
  const n = normalizeSkillVersion(raw);
  if (!n) {
    throw new Error(SKILL_VERSION_REQUIRED_MESSAGE);
  }
  return n;
}

export function formatSkillVersionBadge(version) {
  const n = normalizeSkillVersion(version);
  return n ? `v${n}` : "—";
}
