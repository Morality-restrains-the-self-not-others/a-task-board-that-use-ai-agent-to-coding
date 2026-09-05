<script setup>
import { computed } from "vue";
import { renderMarkdown } from "../lib/renderMarkdown.js";
import {
  autoRunStatusHint,
  skillsStatusHint,
} from "../lib/imageResolveInfo.js";

const props = defineProps({
  skills: { type: Array, default: () => [] },
  skillsStatus: { type: String, default: "" },
  skillsDetail: { type: String, default: "" },
  autoRunStepsMd: { type: String, default: "" },
  autoRunStepsStatus: { type: String, default: "" },
  autoRunStepsDetail: { type: String, default: "" },
});

const skillsOk = computed(() => props.skillsStatus === "ok");
const autoRunOk = computed(() => props.autoRunStepsStatus === "ok");
const autoRunHtml = computed(() => renderMarkdown(props.autoRunStepsMd));
const skillsHint = computed(() => skillsStatusHint(props.skillsStatus));
const autoRunHint = computed(() => autoRunStatusHint(props.autoRunStepsStatus));
</script>

<template>
  <div v-if="skillsStatus || autoRunStepsStatus" class="resolve-info" data-testid="image-resolve-info">
    <label class="lbl">技能列表</label>
    <ul v-if="skillsOk && skills.length" class="skill-list">
      <li v-for="s in skills" :key="s.name" class="skill-item">
        <span class="skill-name">{{ s.name }}</span>
        <span v-if="s.is_default" class="pill skill-default">默认</span>
        <span v-if="s.description" class="skill-desc">{{ s.description }}</span>
      </li>
    </ul>
    <p v-else-if="skillsOk && !skills.length" class="hint">镜像未声明技能</p>
    <p v-else-if="skillsHint" class="hint">
      {{ skillsHint }}<span v-if="skillsDetail" class="detail">：{{ skillsDetail }}</span>
    </p>

    <label class="lbl">自动运行说明</label>
    <article v-if="autoRunOk" class="markdown-body" v-html="autoRunHtml" />
    <p v-else-if="autoRunHint" class="hint">
      {{ autoRunHint }}<span v-if="autoRunStepsDetail" class="detail">：{{ autoRunStepsDetail }}</span>
    </p>
  </div>
</template>

<style scoped src="../views/VendorPortal.css"></style>

<style scoped>
.resolve-info {
  margin-top: 0.25rem;
}
.skill-list {
  list-style: none;
  margin: 0 0 0.5rem;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}
.skill-item {
  display: flex;
  align-items: baseline;
  gap: 0.4rem;
  font-size: 0.9rem;
}
.skill-name {
  font-weight: 600;
}
.skill-default {
  font-size: 0.72rem;
  padding: 0.05rem 0.45rem;
}
.skill-desc {
  color: var(--muted);
}
.detail {
  color: var(--muted);
}
.markdown-body {
  line-height: 1.65;
  margin-bottom: 0.5rem;
  max-height: 16rem;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0.75rem;
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
