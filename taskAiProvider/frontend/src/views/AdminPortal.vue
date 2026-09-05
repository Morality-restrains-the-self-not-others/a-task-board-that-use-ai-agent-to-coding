<template>
  <div>
    <h1 class="h1">平台审核</h1>
    <p class="sub">平台管理员须通过主站单点登录进入；本服务已关闭本地密码登录。</p>

    <div v-if="!token" class="card" style="max-width: 480px">
      <h2 class="h2">通过主站登录</h2>
      <p class="sub">
        请先在主站登录超级管理员账号，再点击下方链接完成换票；也可从主站系统管理「容器镜像列表」页内「镜像市场管理（SSO）」进入。
      </p>
      <div class="row">
        <a class="btn btn-primary" :href="ssoAdminEntryHref">打开主站 SSO（管理端）</a>
      </div>
      <div class="row" style="margin-top: 0.5rem">
        <a class="btn btn-ghost" :href="oidcAdminEntryHref">通过 OIDC 账号登录</a>
      </div>
      <p v-if="msg" class="msg" v-bind="msgTraceId ? { 'data-traceId': msgTraceId } : {}">{{ msg }}</p>
    </div>

    <template v-else>
      <div class="toolbar">
        <span class="muted">已登录：{{ me?.display_name || me?.username }}</span>
        <button class="btn btn-ghost" type="button" @click="logout">退出</button>
      </div>

      <div class="tabs">
        <button :class="['tab-btn', { active: activeTab === 'review' }]" @click="activeTab = 'review'">容器镜像审核</button>
        <button :class="['tab-btn', { active: activeTab === 'vendors' }]" @click="activeTab = 'vendors'">厂商审核</button>
        <button :class="['tab-btn', { active: activeTab === 'docs' }]" @click="activeTab = 'docs'">证照存储</button>
        <button :class="['tab-btn', { active: activeTab === 'templates' }]" @click="activeTab = 'templates'">UserData 模板</button>
      </div>

      <AdminImageReviewTab v-if="activeTab === 'review'" />
      <AdminVendorsTab v-else-if="activeTab === 'vendors'" />
      <AdminVendorDocsStorage v-else-if="activeTab === 'docs'" />
      <AdminUserDataTemplates v-else-if="activeTab === 'templates'" />
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { api } from "../api";
import AdminUserDataTemplates from "../components/AdminUserDataTemplates.vue";
import AdminImageReviewTab from "../components/AdminImageReviewTab.vue";
import AdminVendorsTab from "../components/AdminVendorsTab.vue";
import AdminVendorDocsStorage from "../components/AdminVendorDocsStorage.vue";
import { mainSaasLogoutUrlWithNext, ssoAdminEntryUrl } from "../lib/mainSaasUrls.js";

const token = ref(localStorage.getItem("staff_token") || "");
const me = ref(null);
const msg = ref("");
const msgTraceId = ref("");
const activeTab = ref("review");
const ssoAdminEntryHref = computed(() => ssoAdminEntryUrl());
const oidcAdminEntryHref = computed(() => "/api/auth/oidc/authorize/?role=admin");

function clearMsg() {
  msg.value = "";
  msgTraceId.value = "";
}

function setMsgError(e) {
  msg.value = e?.message || String(e ?? "");
  msgTraceId.value = e?.traceId || "";
}

async function tryExchangeSsoFromHash() {
  const hash = window.location.hash || "";
  if (!hash.startsWith("#sso_bridge=")) {
    return false;
  }
  const encoded = hash.slice("#sso_bridge=".length);
  let bridge = encoded;
  try {
    bridge = decodeURIComponent(encoded);
  } catch {
    bridge = encoded;
  }
  history.replaceState(null, "", window.location.pathname + window.location.search);
  const data = await api("/api/auth/sso/exchange/", {
    method: "POST",
    body: JSON.stringify({ bridge }),
  });
  if (!data.access) {
    return false;
  }
  localStorage.removeItem("vendor_token");
  localStorage.setItem("staff_token", data.access);
  token.value = data.access;
  return true;
}

function tryExchangeOidcFromHash() {
  const hash = window.location.hash || "";
  if (!hash.startsWith("#token=")) {
    return false;
  }
  const raw = hash.slice("#token=".length);
  history.replaceState(null, "", window.location.pathname + window.location.search);
  localStorage.removeItem("vendor_token");
  localStorage.setItem("staff_token", raw);
  token.value = raw;
  return true;
}

function logout() {
  localStorage.removeItem("staff_token");
  localStorage.removeItem("vendor_token");
  token.value = "";
  me.value = null;
  const nextAbs = `${window.location.origin}${window.location.pathname || "/"}${window.location.search || ""}`;
  const url = mainSaasLogoutUrlWithNext(nextAbs);
  if (url) {
    window.location.assign(url);
  }
}

async function loadMe() {
  try {
    me.value = await api("/api/admin/auth/me/");
  } catch {
    me.value = null;
  }
}

onMounted(async () => {
  clearMsg();
  if (tryExchangeOidcFromHash()) {
    try {
      await loadMe();
      return;
    } catch (e) {
      localStorage.removeItem("staff_token");
      token.value = "";
      setMsgError(e);
      return;
    }
  }
  try {
    if (await tryExchangeSsoFromHash()) {
      await loadMe();
      return;
    }
  } catch (e) {
    setMsgError(e);
  }
  if (token.value) {
    await loadMe();
  }
});
</script>

<style scoped src="./AdminPortal.css"></style>
