<script setup>
import { onMounted, ref } from "vue";
import { api } from "../api.js";
import {
  skillVersionOptionLabel,
  writableSkillVersions,
} from "../lib/saasInboundSkillVersion.js";

const model = defineModel({ type: String, default: "" });

const loading = ref(true);
const error = ref("");
const errorTraceId = ref("");
const options = ref([]);

onMounted(async () => {
  try {
    const data = await api("/api/ai-provider/saas-inbound-skill-versions/");
    options.value = writableSkillVersions(data);
  } catch (e) {
    error.value = e?.message || String(e ?? "加载接口版本失败");
    errorTraceId.value = e?.traceId || "";
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="skill-version-field">
    <label class="lbl" for="saas-inbound-skill-version-select">容器→SaaS 接口版本</label>
    <select
      id="saas-inbound-skill-version-select"
      v-model="model"
      class="inp"
      required
      data-testid="saas-inbound-skill-version-select"
      :disabled="loading || Boolean(error)"
    >
      <option value="">请选择已发布版本</option>
      <option v-for="v in options" :key="v.version" :value="String(v.version)">
        {{ skillVersionOptionLabel(v) }}
      </option>
    </select>
    <p v-if="loading" class="hint">加载已发布接口版本…</p>
    <p
      v-else-if="error"
      class="err"
      role="alert"
      v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
    >
      {{ error }}
    </p>
    <p v-else class="hint">须选择已发布契约；与镜像标签（版本号）相互独立。</p>
  </div>
</template>
