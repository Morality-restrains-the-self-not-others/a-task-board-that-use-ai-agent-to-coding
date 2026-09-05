<script setup>
const props = defineProps({
  title: { type: String, required: true },
  message: { type: String, default: "" },
  confirmLabel: { type: String, required: true },
  danger: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  error: { type: String, default: "" },
  errorTraceId: { type: String, default: "" },
});

const emit = defineEmits(["cancel", "confirm"]);
</script>

<template>
  <div class="modal-bg" @click.self="emit('cancel')">
    <div class="card modal" role="dialog" aria-modal="true" data-testid="version-action-confirm">
      <h3>{{ title }}</h3>
      <p class="sub">{{ message }}</p>
      <p
        v-if="error"
        class="msg"
        v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
      >{{ error }}</p>
      <div class="row">
        <button class="btn btn-ghost" type="button" :disabled="busy" @click="emit('cancel')">取消</button>
        <button
          :class="['btn', danger ? 'btn-danger' : 'btn-primary']"
          type="button"
          :disabled="busy"
          :aria-busy="busy ? 'true' : 'false'"
          @click="emit('confirm')"
        >{{ busy ? "处理中…" : confirmLabel }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped src="../views/VendorPortal.css"></style>
