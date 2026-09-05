<script setup>
import { onMounted, ref } from "vue";
import { api } from "../api.js";
import { renderMarkdown } from "../lib/renderMarkdown.js";
import {
  formatSkillVersionBadge,
  skillVersionOptionLabel,
} from "../lib/saasInboundSkillVersion.js";
import {
  SAAS_MACHINE_CONTAINER_SKILL_LABEL,
  saasMachineContainerSkillMdHref,
} from "../lib/saasMachineContainerSkill.js";

const loading = ref(true);
const error = ref("");
const errorTraceId = ref("");
const html = ref("");
const currentVersion = ref("");
const selectedVersion = ref("");
const catalogVersions = ref([]);

async function loadMarkdown(version) {
  const resp = await fetch(saasMachineContainerSkillMdHref(version));
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}`);
  }
  const text = await resp.text();
  html.value = renderMarkdown(text);
}

onMounted(async () => {
  try {
    const catalog = await api("/api/ai-provider/saas-inbound-skill-versions/");
    currentVersion.value = String(catalog?.current || "");
    catalogVersions.value = Array.isArray(catalog?.versions) ? catalog.versions : [];
    selectedVersion.value = currentVersion.value;
    await loadMarkdown(selectedVersion.value);
  } catch (e) {
    error.value = `加载「容器→SaaS 接口」文档失败：${e?.message || e}`;
    errorTraceId.value = e?.traceId || "";
  } finally {
    loading.value = false;
  }
});

async function onDocVersionChange() {
  try {
    error.value = "";
    errorTraceId.value = "";
    await loadMarkdown(selectedVersion.value);
  } catch (e) {
    error.value = `加载接口版本文档失败：${e?.message || e}`;
    errorTraceId.value = e?.traceId || "";
  }
}
</script>

<template>
  <section class="skill-doc" data-testid="saas-machine-container-skill-page">
    <h2 class="page-title">{{ SAAS_MACHINE_CONTAINER_SKILL_LABEL }}</h2>
    <div class="skill-meta">
      <span
        class="skill-version-badge"
        data-testid="saas-inbound-skill-version-badge"
      >
        当前接口 {{ formatSkillVersionBadge(currentVersion) }}
      </span>
      <label class="skill-version-picker">
        查看已发布版本
        <select
          v-model="selectedVersion"
          class="inp"
          data-testid="saas-inbound-skill-version-doc-select"
          @change="onDocVersionChange"
        >
          <option v-for="v in catalogVersions" :key="v.version" :value="String(v.version)">
            {{ skillVersionOptionLabel(v) }}
          </option>
        </select>
      </label>
    </div>
    <p v-if="loading" class="muted">加载中…</p>
    <p
      v-else-if="error"
      class="error"
      data-testid="saas-machine-container-skill-error"
      v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
    >
      {{ error }}
    </p>
    <article v-else class="markdown-body" data-testid="saas-machine-container-skill-body" v-html="html" />
  </section>
</template>

<style scoped>
.skill-doc {
  line-height: 1.65;
}
.page-title {
  margin-bottom: 1rem;
}
.skill-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.25rem;
  align-items: center;
  margin-bottom: 1rem;
}
.skill-version-badge {
  display: inline-block;
  padding: 0.15rem 0.55rem;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.12);
  color: #1d4ed8;
  font-weight: 600;
  font-size: 0.9rem;
}
.skill-version-picker {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  font-size: 0.9rem;
}
.skill-version-picker .inp {
  min-width: 16rem;
  padding: 0.35rem 0.5rem;
  border: 1px solid var(--border);
  border-radius: 4px;
}
.muted {
  color: var(--muted);
}
.error {
  color: #c0392b;
}
.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.75rem 0;
  font-size: 0.9rem;
}
.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid var(--border);
  padding: 0.4rem 0.6rem;
  text-align: left;
  vertical-align: top;
}
.markdown-body :deep(th) {
  background: rgba(127, 127, 127, 0.08);
  font-weight: 600;
}
.markdown-body :deep(code) {
  background: rgba(127, 127, 127, 0.12);
  padding: 0.1em 0.3em;
  border-radius: 3px;
  font-size: 0.88em;
}
.markdown-body :deep(pre) {
  background: rgba(127, 127, 127, 0.08);
  padding: 0.75rem;
  overflow-x: auto;
  border-radius: 4px;
}
.markdown-body :deep(blockquote) {
  margin: 0.75rem 0;
  padding-left: 0.75rem;
  border-left: 3px solid var(--border);
  color: var(--muted);
}
</style>
