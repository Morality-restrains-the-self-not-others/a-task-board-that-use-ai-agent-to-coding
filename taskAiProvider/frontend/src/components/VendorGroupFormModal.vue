<template>
    <div v-if="vp.groupOpen" class="modal-bg" @click.self="vp.groupOpen = false">
      <div class="card modal">
        <h3>{{ vp.groupEditingId ? '编辑镜像组' : '创建镜像组' }}</h3>
        <label class="lbl">名称 <span class="req">*</span></label>
        <input v-model="vp.groupForm.name" class="inp" />
        <label class="lbl">描述 <span class="req">*</span></label>
        <textarea v-model="vp.groupForm.description" class="inp area" rows="2"></textarea>
        <label class="lbl">图标 <span class="req">*</span></label>
        <input
          type="file"
          class="inp"
          accept="image/png,image/jpeg,image/webp"
          @change="vp.onGroupIconFile"
        />
        <p class="hint">PNG / JPEG / WEBP，不超过 512KB，必填</p>
        <div v-if="vp.groupIconPreview || vp.groupForm.icon_url" class="icon-preview-wrap">
          <img
            class="group-icon-preview"
            :src="vp.groupIconPreview || vp.groupForm.icon_url"
            alt="镜像组图标预览"
          />
        </div>
        <p
          v-if="vp.groupFormError"
          class="err"
          v-bind="vp.groupFormErrorTraceId ? { 'data-traceId': vp.groupFormErrorTraceId } : {}"
        >{{ vp.groupFormError }}</p>
        <div class="row">
          <button class="btn btn-ghost" type="button" @click="vp.groupOpen = false">取消</button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="vp.groupSaving"
            :aria-busy="vp.groupSaving ? 'true' : 'false'"
            @click="vp.saveGroup"
          >
            {{ vp.groupSaving ? "保存中…" : "保存" }}
          </button>
        </div>
      </div>
    </div>
</template>

<script setup>
defineProps({
  vp: { type: Object, required: true },
});
</script>

<style scoped src="../views/VendorPortal.css"></style>
