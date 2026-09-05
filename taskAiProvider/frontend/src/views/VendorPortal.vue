<template>
  <div>
    <div class="header-row">
      <h1 class="h1">厂商门户</h1>
    </div>

    <div v-if="!vp.token" class="card" style="max-width: 480px">
      <p class="sub">
        厂商门户仅支持主站单点登录（主站邮箱须与厂商档案邮箱一致）。本地注册与密码登录已关闭；尚未认证的厂商请到主站租户「镜像市场」申请认证，审核通过后再使用 SSO。
      </p>
      <h2 class="h2">通过主站登录</h2>
      <p class="sub">请先在主站登录，再点击下方链接完成换票。</p>
      <div class="row">
        <a class="btn btn-primary row-btn" :href="vp.ssoVendorEntryHref">打开主站 SSO（厂商门户）</a>
      </div>
      <div class="row" style="margin-top: 0.5rem">
        <a class="btn btn-ghost row-btn" :href="vp.oidcVendorEntryHref">通过 OIDC 账号登录</a>
      </div>
      <p v-if="vp.msg" class="msg" v-bind="vp.msgTraceId ? { 'data-traceId': vp.msgTraceId } : {}">{{ vp.msg }}</p>
    </div>

    <template v-else>
      <div class="toolbar">
        <span>已登录：<strong>{{ vp.displayCompanyName }}</strong>（{{ vp.displayEmail }}） - ID: {{ vp.me?.id }}</span>
        <button class="btn btn-ghost" type="button" @click="vp.logout">退出</button>
      </div>

      <div class="vendor-summary">
        <div class="summary-card">
          <div class="summary-icon">🏢</div>
          <div class="summary-body">
            <div class="summary-label">公司名称</div>
            <div class="summary-value">{{ vp.displayCompanyName }}</div>
          </div>
        </div>
        <div class="summary-card">
          <div class="summary-icon">👤</div>
          <div class="summary-body">
            <div class="summary-label">联系人</div>
            <div class="summary-value">{{ vp.me?.contact_name || '—' }}</div>
          </div>
        </div>
        <div class="summary-card">
          <div class="summary-icon">📧</div>
          <div class="summary-body">
            <div class="summary-label">邮箱</div>
            <div class="summary-value">{{ vp.displayEmail }}</div>
          </div>
        </div>
      </div>
      <div class="vendor-stats">
        <div class="stat-item">
          <span class="stat-num">{{ vp.imageGroups.length }}</span>
          <span class="stat-label">镜像组</span>
        </div>
        <div class="stat-item">
          <span class="stat-num">{{ vp.containerImages.length }}</span>
          <span class="stat-label">容器镜像</span>
        </div>
        <div class="stat-item">
          <span class="stat-num">{{ vp.serverImages.length }}</span>
          <span class="stat-label">服务器镜像</span>
        </div>
        <div class="stat-item">
          <span class="stat-num">{{ vp.activeCredentialCount }}</span>
          <span class="stat-label">已启用密钥 / {{ vp.credentials.length }}  total</span>
        </div>
      </div>

      <div class="tabs">
        <button :class="{ on: vp.tab === 'groups' }" type="button" @click="vp.tab = 'groups'">镜像组</button>
        <button :class="{ on: vp.tab === 'cs' }" type="button" @click="vp.tab = 'cs'">云平台服务器镜像</button>
        <button :class="{ on: vp.tab === 'credentials' }" type="button" @click="vp.tab = 'credentials'">云平台测试密钥</button>
      </div>

      <VendorImageGroupsTab v-show="vp.tab === 'groups'" :vp="vp" />
      <VendorCloudServerImagesTab v-show="vp.tab === 'cs'" :vp="vp" />

      <div v-show="vp.tab === 'credentials'">
        <VendorCloudCredentialsTab
          :ref="(el) => { vp.credentialsTabRef = el }"
          :cloud-platforms="cloudPlatforms"
          :credential-hint="vp.credentialHint"
          @credentials-changed="vp.credentialHint = ''; vp.loadCredentials()"
        />
      </div>
    </template>

    <p v-if="vp.msg" class="msg" v-bind="vp.msgTraceId ? { 'data-traceId': vp.msgTraceId } : {}">{{ vp.msg }}</p>

    <VendorGroupFormModal :vp="vp" />

    <VendorVersionActionConfirmModal
      v-if="vp.actionConfirm"
      :title="vp.actionConfirmTitle()"
      :message="vp.actionConfirmMessage()"
      :confirm-label="vp.actionConfirmLabel()"
      :danger="vp.actionConfirmDanger()"
      :busy="vp.versionActionBusy"
      :error="vp.msg"
      :error-trace-id="vp.msgTraceId"
      @cancel="vp.actionConfirm = null"
      @confirm="vp.confirmPortalAction"
    />

    <VendorContainerImageVersionModals
      :ref="(el) => { vp.imageVersionModals = el }"
      @saved="vp.onImageVersionSaved"
      @error="vp.setMsgError"
    />

    <RegionEnvModal
      :visible="vp.regionEnvOpen"
      :server-images="vp.serverImages"
      :userdata-template-options="vp.userdataTemplateOptions"
      :cloud-platforms="cloudPlatforms"
      :container-image-id="vp.regionEnvImageId"
      :platform-type="vp.regionEnvPlatform"
      :region="vp.regionEnvRegion"
      :target-architectures="vp.envTargetArchs"
      :initial-cloud-server-image-id="vp.regionEnvInitialCsId"
      :initial-userdata-template-id="vp.regionEnvInitialUdId"
      :saving="vp.savingRegionEnv"
      :error="vp.regionEnvError"
      :error-trace-id="vp.regionEnvErrorTraceId"
      @close="vp.regionEnvOpen = false"
      @save="vp.handleRegionEnvSave"
      @open-cs-create="vp.handleRegionEnvOpenCsCreate"
    />
  </div>
</template>

<script setup>
import { useVendorPortal } from "../composables/useVendorPortal.js";
import VendorCloudCredentialsTab from "../components/VendorCloudCredentialsTab.vue";
import RegionEnvModal from "../components/RegionEnvModal.vue";
import VendorContainerImageVersionModals from "../components/VendorContainerImageVersionModals.vue";
import VendorImageGroupsTab from "../components/VendorImageGroupsTab.vue";
import VendorCloudServerImagesTab from "../components/VendorCloudServerImagesTab.vue";
import VendorGroupFormModal from "../components/VendorGroupFormModal.vue";
import VendorVersionActionConfirmModal from "../components/VendorVersionActionConfirmModal.vue";
import { CLOUD_PLATFORMS as cloudPlatforms } from "../utils/platformLabel.js";

const vp = useVendorPortal();
</script>

<style scoped src="./VendorPortal.css"></style>
