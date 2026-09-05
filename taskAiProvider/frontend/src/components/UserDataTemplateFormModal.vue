<template>
  <div v-if="visible" class="udt-modal-overlay" @click.self="$emit('close')">
    <div class="udt-modal-panel" @click.stop>
      <div class="udt-modal-header">
        <h3 class="udt-modal-title">{{ title }}</h3>
        <button type="button" class="udt-modal-close" @click="$emit('close')" aria-label="关闭">
          &times;
        </button>
      </div>

      <form :id="formId || undefined" @submit.prevent="$emit('submit')">
        <div class="udt-form-stack">
          <div>
            <label class="udt-label">模板名称</label>
            <input type="text" v-model="form.name" class="udt-input" placeholder="模板名称" required>
          </div>

          <div>
            <label class="udt-label">版本号</label>
            <input type="text" v-model="form.version" class="udt-input" placeholder="版本号，例如 1.0.0" required>
          </div>

          <div>
            <label class="udt-label">支持的操作系统版本</label>
            <select v-model="form.os_type" class="udt-input" required>
              <option value="">请选择操作系统版本</option>
              <template v-for="grp in osGroups" :key="grp.label">
                <optgroup :label="grp.label">
                  <option v-for="opt in grp.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </optgroup>
              </template>
            </select>
          </div>

          <div>
            <label class="udt-label">UserData 变量</label>
            <p class="udt-hint">
              运行时 ID/令牌/镜像<strong>禁止</strong>写进模板真实值：「生成容器脚本」固定写入
              <code class="udt-code">__TASK2APP_ACCESS_TOKEN__</code>、
              <code class="udt-code">__TASK2APP_TASK_CLOUD_PREFIX__</code>、
              <code class="udt-code">__TASK2APP_CONTAINER_IMAGE__</code>、
              <code class="udt-code">__TASK2APP_CONTAINER_NAME__</code>、
              <code class="udt-code">__TASK2APP_COMMENT_ID__</code>、
              <code class="udt-code">__TASK2APP_TRACE_ID__</code>；
              由 start-vm → RunInstances 前动态替换。
              <code class="udt-code">__TASK2APP_TASK_API_ENDPOINT__</code> 仅为 SaaS API 根。
              「生成容器脚本」会写入逐步 <span class="udt-mono">boot-progress</span>（含 trace_id，经 SSE 显示到任务详情）。
            </p>
            <div v-for="(variable, index) in form.userdata_variables" :key="index" class="udt-var-row">
              <input type="text" v-model="variable.name" class="udt-input udt-input-sm udt-flex-1" placeholder="变量名">
              <button type="button" class="udt-btn udt-btn-danger-sm" @click="$emit('removeVar', 'userdata_variables', index)">
                移除
              </button>
            </div>
            <button type="button" class="udt-btn udt-btn-info-sm" @click="$emit('addVar', 'userdata_variables')">
              添加变量
            </button>
          </div>

          <div>
            <label class="udt-label">容器变量</label>
            <div v-for="(variable, index) in form.container_variables" :key="index" class="udt-var-row">
              <input type="text" v-model="variable.name" class="udt-input udt-input-sm udt-flex-1" placeholder="变量名">
              <button type="button" class="udt-btn udt-btn-danger-sm" @click="$emit('removeVar', 'container_variables', index)">
                移除
              </button>
            </div>
            <button type="button" class="udt-btn udt-btn-info-sm" @click="$emit('addVar', 'container_variables')">
              添加变量
            </button>
          </div>

          <div>
            <label class="udt-label">模板内容</label>
            <textarea
              :id="contentFieldId || undefined"
              v-model="form.content"
              class="udt-input udt-textarea"
              placeholder="模板内容"
              rows="6"
              required
            ></textarea>
          </div>

          <div>
            <label class="udt-label">自动验证脚本</label>
            <textarea
              v-model="form.auto_verify_script"
              class="udt-input udt-textarea"
              placeholder="自动验证脚本"
              rows="4"
            ></textarea>
          </div>

          <div class="udt-checkbox-row">
            <input type="checkbox" v-model="form.is_active" class="udt-checkbox" id="udt-is-active">
            <label for="udt-is-active" class="udt-checkbox-label">启用</label>
          </div>

          <div class="udt-actions">
            <button type="button" class="udt-btn udt-btn-secondary" @click="$emit('close')">
              取消
            </button>
            <button type="button" class="udt-btn udt-btn-success" @click="$emit('generateScript')">
              生成容器脚本
            </button>
            <button type="submit" class="udt-btn udt-btn-primary" :disabled="loading">
              <span v-if="loading">保存中...</span>
              <span v-else>保存</span>
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, default: '' },
  form: { type: Object, required: true },
  loading: { type: Boolean, default: false },
  osGroups: { type: Array, default: () => [] },
  formId: { type: String, default: '' },
  contentFieldId: { type: String, default: '' },
})

defineEmits(['close', 'submit', 'generateScript', 'addVar', 'removeVar'])
</script>

<style scoped>
.udt-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 1.25rem 0.75rem 2rem;
  background: rgba(0, 0, 0, 0.55);
  -webkit-overflow-scrolling: touch;
}

.udt-modal-panel {
  flex: 0 1 auto;
  width: 100%;
  max-width: 42rem;
  max-height: min(92vh, calc(100vh - 2.5rem));
  overflow-x: hidden;
  overflow-y: auto;
  margin-top: 0;
  margin-bottom: auto;
  background: #fff;
  border-radius: 0.5rem;
  box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.12);
  padding: 1.5rem;
}

.udt-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.udt-modal-title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
}

.udt-modal-close {
  border: none;
  background: transparent;
  color: #6b7280;
  font-size: 1.5rem;
  line-height: 1;
  padding: 0;
  cursor: pointer;
  transition: color 0.15s;
}

.udt-modal-close:hover {
  color: #374151;
}

.udt-form-stack > * + * {
  margin-top: 1rem;
}

.udt-label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.udt-hint {
  margin: 0 0 0.5rem;
  font-size: 0.75rem;
  line-height: 1.625;
  color: #6b7280;
}

.udt-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: #374151;
}

.udt-code {
  font-size: 0.75rem;
  background: #f3f4f6;
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
}

.udt-input {
  width: 100%;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  border: 1px solid #d1d5db;
  background: #fff;
  color: #111827;
  transition: box-shadow 0.3s, border-color 0.3s;
}

.udt-input:focus {
  outline: none;
  border-color: transparent;
  box-shadow: 0 0 0 2px var(--accent);
}

.udt-input-sm {
  padding: 0.5rem 1rem;
}

.udt-textarea {
  resize: vertical;
  min-height: 6rem;
}

.udt-var-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.udt-flex-1 {
  flex: 1 1 0%;
}

.udt-btn {
  border: none;
  border-radius: 0.5rem;
  padding: 0.75rem 1rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.15s, filter 0.15s;
}

.udt-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.udt-btn-sm,
.udt-btn-danger-sm,
.udt-btn-info-sm {
  padding: 0.5rem 0.75rem;
  font-weight: 500;
}

.udt-btn-danger-sm {
  background: #fee2e2;
  color: #dc2626;
}

.udt-btn-danger-sm:hover {
  background: #fecaca;
}

.udt-btn-info-sm {
  background: #dbeafe;
  color: #2563eb;
}

.udt-btn-info-sm:hover {
  background: #bfdbfe;
}

.udt-checkbox-row {
  display: flex;
  align-items: center;
}

.udt-checkbox {
  width: 1rem;
  height: 1rem;
  accent-color: var(--accent);
}

.udt-checkbox-label {
  margin-left: 0.5rem;
  font-size: 0.875rem;
  color: #374151;
}

.udt-actions {
  display: flex;
  gap: 1rem;
}

.udt-actions .udt-btn {
  flex: 1 1 0%;
}

.udt-btn-secondary {
  background: #e5e7eb;
  color: #1f2937;
}

.udt-btn-secondary:hover {
  background: #d1d5db;
}

.udt-btn-success {
  background: #16a34a;
  color: #fff;
}

.udt-btn-success:hover {
  background: #15803d;
}

.udt-btn-primary {
  background: linear-gradient(135deg, var(--accent), var(--accent-dim));
  color: #fff;
}

.udt-btn-primary:hover:not(:disabled) {
  filter: brightness(0.92);
}
</style>
