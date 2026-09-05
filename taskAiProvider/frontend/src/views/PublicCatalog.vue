<template>
  <div>
    <h1 class="h1">已上架镜像（公开）</h1>
    <p class="sub">主 SaaS 可通过 <code>/api/public/catalog/</code> 拉取已审批镜像元数据（含各云平台宿主机镜像、硬件说明及选用的 UserData 模板版本）；通过 <code>/api/public/userdata-templates/</code> 查询当前启用的 UserData 模板版本列表（可选 <code>?os_type=centos</code> 等筛选）。</p>
    <a href="/api/public/catalog/" target="_blank" class="api-button">Catalog API</a>
    <a href="/api/public/userdata-templates/" target="_blank" class="api-button">UserData 版本</a>

    <!-- 未提交审核镜像查询区域 -->
    <div class="query-section">
      <h2 class="h2">未提交审核镜像查询</h2>
      <p class="sub">通过云厂商ID和容器ID查询未提交审核的镜像信息</p>
      <form @submit.prevent="queryUnsubmittedImage" class="query-form">
        <div class="form-group">
          <label for="vendorId">云厂商ID</label>
          <input type="text" id="vendorId" v-model="queryParams.vendorId" placeholder="请输入云厂商ID" required>
        </div>
        <div class="form-group">
          <label for="containerId">容器ID</label>
          <input type="text" id="containerId" v-model="queryParams.containerId" placeholder="请输入容器ID" required>
        </div>
        <button type="submit" class="query-button" :disabled="queryLoading">
          {{ queryLoading ? '查询中...' : '查询' }}
        </button>
      </form>
      
      <div v-if="queryError" class="err" v-bind="queryErrorTraceId ? { 'data-traceId': queryErrorTraceId } : {}">{{ queryError }}</div>
      <div v-else-if="queryResult" class="query-result">
        <h3 class="h3">查询结果</h3>
        <div class="result-card">
          <div class="result-item">
            <span class="result-label">图标：</span>
            <img
              v-if="imageGroupIconSrc(queryResult)"
              data-testid="image-group-icon"
              class="catalog-icon"
              :src="imageGroupIconSrc(queryResult)"
              :alt="queryResult.name"
            />
            <span v-else class="catalog-icon catalog-icon-placeholder" data-testid="image-group-icon-placeholder" aria-hidden="true"></span>
          </div>
          <div class="result-item">
            <span class="result-label">镜像名称：</span>
            <span>{{ queryResult.name }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">版本：</span>
            <span>{{ queryResult.version }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">容器→SaaS 接口版本：</span>
            <span data-testid="catalog-saas-inbound-skill-version">{{ queryResult.saas_inbound_skill_version || "—" }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">镜像地址：</span>
            <span class="mono">{{ queryResult.image_url }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">状态：</span>
            <span>{{ queryResult.status_display }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">厂商：</span>
            <span>{{ queryResult.vendor?.company_name }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">更新时间：</span>
            <span>{{ queryResult.updated_at }}</span>
          </div>
          <div class="result-item">
            <span class="result-label">运行环境：</span>
            <div v-if="queryResult.runtime_environments?.length">
              <div v-for="(env, index) in queryResult.runtime_environments" :key="index" class="env-item-small">
                {{ env.platform_type_display }} · {{ env.region }} · {{ env.image_name }}
              </div>
            </div>
            <span v-else class="muted">无</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <div v-else-if="error" class="err" v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}">{{ error }}</div>
    <div v-else class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>图标</th>
            <th>名称</th>
            <th>厂商</th>
            <th>镜像地址</th>
            <th>接口版本</th>
            <th>架构</th>
            <th>运行环境（宿主机镜像）</th>
            <th>硬件</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id">
            <td>
              <img
                v-if="imageGroupIconSrc(row)"
                data-testid="image-group-icon"
                class="catalog-icon"
                :src="imageGroupIconSrc(row)"
                :alt="row.name"
              />
              <span v-else class="catalog-icon catalog-icon-placeholder" data-testid="image-group-icon-placeholder" aria-hidden="true"></span>
            </td>
            <td>{{ row.name }}</td>
            <td>{{ row.vendor?.company_name || "—" }}</td>
            <td class="mono">{{ row.image_url }}</td>
            <td data-testid="public-catalog-saas-inbound-skill-version">{{ row.saas_inbound_skill_version ? ('v' + row.saas_inbound_skill_version) : "—" }}</td>
            <td>{{ (row.target_architectures || []).join(", ") }}</td>
            <td class="env-stack">
              <template v-if="row.runtime_environments?.length">
                <div
                  v-for="env in row.runtime_environments"
                  :key="envKey(env)"
                  class="env-item"
                >
                  <div class="env-head">
                    <span class="env-plat">{{ env.platform_type_display }}</span>
                    <span class="env-sep">·</span>
                    <span>{{ env.region }}</span>
                  </div>
                  <div class="mono env-name">{{ env.image_name }}</div>
                  <div class="env-meta muted">
                    <span class="mono">{{ env.image_id }}</span>
                    <template v-if="env.architecture">
                      <span class="env-sep">·</span>
                      <span>{{ env.architecture }}</span>
                    </template>
                    <template v-if="env.os_type">
                      <span class="env-sep">·</span>
                      <span>{{ env.os_type }}</span>
                    </template>
                  </div>
                </div>
              </template>
              <span v-else class="muted">—</span>
            </td>
            <td class="hw-stack">
              <template v-if="row.runtime_environments?.length">
                <div
                  v-for="env in row.runtime_environments"
                  :key="'hw-' + envKey(env)"
                  class="hw-item"
                >
                  <span v-if="env.hardware_summary">{{ env.hardware_summary }}</span>
                  <span v-else class="muted">未配置</span>
                </div>
              </template>
              <span v-else class="muted">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { api } from "../api";
import { imageGroupIconSrc } from "../utils/imageGroupIcon.js";

const rows = ref([]);
const loading = ref(true);
const error = ref("");
const errorTraceId = ref("");

// 查询未提交审核镜像的相关变量
const queryParams = ref({
  vendorId: "",
  containerId: ""
});
const queryLoading = ref(false);
const queryError = ref("");
const queryErrorTraceId = ref("");
const queryResult = ref(null);

function envKey(env) {
  return [env.platform_type, env.region, env.image_id].join("|");
}

// 查询未提交审核镜像的方法
async function queryUnsubmittedImage() {
  try {
    queryLoading.value = true;
    queryError.value = "";
    queryErrorTraceId.value = "";
    queryResult.value = null;

    const response = await api(`/api/public/unsubmitted-image/?vendor_id=${queryParams.value.vendorId}&container_id=${queryParams.value.containerId}`);
    queryResult.value = response;
  } catch (e) {
    queryError.value = e.message;
    queryErrorTraceId.value = e.traceId || "";
  } finally {
    queryLoading.value = false;
  }
}

onMounted(async () => {
  try {
    rows.value = await api("/api/public/catalog/");
  } catch (e) {
    error.value = e.message;
    errorTraceId.value = e.traceId || "";
  } finally {
    loading.value = false;
  }
});
</script>

<style scoped>
.h1 {
  font-size: 1.35rem;
  margin: 0 0 0.35rem;
}
.h2 {
  font-size: 1.15rem;
  margin: 2rem 0 0.35rem;
}
.h3 {
  font-size: 1rem;
  margin: 1rem 0 0.35rem;
}
.sub {
  color: var(--muted);
  font-size: 0.9rem;
  margin-bottom: 1rem;
}
.mono {
  font-size: 0.8rem;
  word-break: break-all;
}
.muted {
  color: var(--muted);
}
.err {
  color: var(--danger);
  margin: 1rem 0;
}
code {
  font-size: 0.85rem;
  background: #f1f5f9;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
}
.api-button {
  display: inline-block;
  background: #3b82f6;
  color: white;
  padding: 0.4rem 0.8rem;
  border-radius: 4px;
  text-decoration: none;
  font-size: 0.9rem;
  margin-bottom: 1rem;
  transition: background 0.2s;
}
.api-button:hover {
  background: #2563eb;
}
.api-button + .api-button {
  margin-left: 0.5rem;
}

/* 查询区域样式 */
.query-section {
  margin: 2rem 0;
  padding: 1.5rem;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid var(--border);
}
.query-form {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: end;
  margin-bottom: 1.5rem;
}
.form-group {
  flex: 1;
  min-width: 200px;
}
.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
  font-weight: 500;
}
.form-group input {
  width: 100%;
  padding: 0.6rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 0.9rem;
}
.query-button {
  background: #3b82f6;
  color: white;
  border: none;
  padding: 0.6rem 1.2rem;
  border-radius: 4px;
  font-size: 0.9rem;
  cursor: pointer;
  transition: background 0.2s;
}
.query-button:hover {
  background: #2563eb;
}
.query-button:disabled {
  background: #93c5fd;
  cursor: not-allowed;
}

/* 查询结果样式 */
.query-result {
  margin-top: 1.5rem;
}
.result-card {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  border: 1px solid var(--border);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}
.result-item {
  margin-bottom: 0.75rem;
  display: flex;
  align-items: flex-start;
}
.result-label {
  font-weight: 500;
  min-width: 100px;
  flex-shrink: 0;
}
.env-item-small {
  margin: 0.25rem 0;
  font-size: 0.85rem;
  padding-left: 100px;
}

.catalog-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  object-fit: cover;
  display: inline-block;
  border: 1px solid var(--border);
  background: var(--surface);
  vertical-align: middle;
}
.catalog-icon-placeholder {
  background: #e2e8f0;
}
.table-wrap {
  overflow-x: auto;
}
.env-stack,
.hw-stack {
  vertical-align: top;
  min-width: 12rem;
  max-width: 22rem;
  font-size: 0.85rem;
  line-height: 1.35;
}
.env-item {
  padding-bottom: 0.65rem;
  margin-bottom: 0.65rem;
  border-bottom: 1px solid var(--border);
}
.env-item:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}
.env-head {
  font-size: 0.88rem;
  margin-bottom: 0.2rem;
}
.env-plat {
  font-weight: 600;
  color: var(--text);
}
.env-sep {
  opacity: 0.5;
  margin: 0 0.2rem;
}
.env-name {
  font-size: 0.78rem;
}
.env-meta {
  font-size: 0.72rem;
  margin-top: 0.15rem;
}
.hw-item {
  padding-bottom: 0.65rem;
  margin-bottom: 0.65rem;
  border-bottom: 1px solid var(--border);
  min-height: 2.5rem;
}
.hw-item:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}
</style>
