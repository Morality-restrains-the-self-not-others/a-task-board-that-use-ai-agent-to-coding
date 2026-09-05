<script setup>
const props = defineProps({
  title: { type: String, required: true },
  confirmLabel: { type: String, required: true },
  confirmClass: { type: String, default: "btn-primary" },
  requireNote: { type: Boolean, default: false },
  placeholder: { type: String, default: "必填" },
  busy: { type: Boolean, default: false },
  note: { type: String, default: "" },
  error: { type: String, default: "" },
  errorTraceId: { type: String, default: "" },
});

const emit = defineEmits(["update:note", "cancel", "confirm"]);

function onConfirm() {
  emit("confirm");
}
</script>

<template>
  <div class="modal-bg" @click.self="emit('cancel')">
    <div class="card modal" role="dialog" aria-modal="true">
      <h3>{{ title }}</h3>
      <textarea
        v-if="requireNote"
        class="inp area"
        rows="4"
        :placeholder="placeholder"
        :value="note"
        :disabled="busy"
        @input="emit('update:note', $event.target.value)"
      />
      <p
        v-if="error"
        class="msg"
        v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
      >{{ error }}</p>
      <div class="row">
        <button class="btn btn-ghost" type="button" :disabled="busy" @click="emit('cancel')">取消</button>
        <button
          :class="['btn', confirmClass]"
          type="button"
          :disabled="busy"
          :aria-busy="busy ? 'true' : 'false'"
          @click="onConfirm"
        >{{ busy ? "处理中…" : confirmLabel }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped src="../views/AdminPortal.css"></style>
