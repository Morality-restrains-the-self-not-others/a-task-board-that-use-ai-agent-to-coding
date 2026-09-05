// 顶栏「容器→SaaS 接口」入口：指向站内渲染页（真实 href 由 router-link 输出）。
export const SAAS_MACHINE_CONTAINER_SKILL_HREF = "/saas-machine-container";
export const SAAS_MACHINE_CONTAINER_SKILL_LABEL = "容器→SaaS 接口";
// 站内页拉取同一 SSOT 原文的路径（由 vite 中间件 / 构建产物提供）。
export const SAAS_MACHINE_CONTAINER_SKILL_MD_HREF = "/saas-machine-container.md";

export function saasMachineContainerSkillMdHref(version) {
  const v = String(version || "").trim();
  if (!v) return SAAS_MACHINE_CONTAINER_SKILL_MD_HREF;
  return `${SAAS_MACHINE_CONTAINER_SKILL_MD_HREF}?version=${encodeURIComponent(v)}`;
}
