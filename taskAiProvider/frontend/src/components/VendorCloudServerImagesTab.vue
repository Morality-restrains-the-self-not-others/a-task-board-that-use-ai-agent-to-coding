<template>
  <div>
        <div v-if="vp.credentialHint" class="credential-alert">
          {{ vp.credentialHint }}
          <button class="btn-mini" type="button" @click="vp.tab = 'credentials'">去配置测试密钥</button>
        </div>
        <div class="row-between">
          <h2 class="h2">云平台服务器镜像</h2>
          <button class="btn btn-primary" type="button" @click="vp.openCsCreate">登记镜像</button>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>云平台</th>
                <th>名称</th>
                <th>镜像 ID</th>
                <th>区域</th>
                <th>架构</th>
                <th>操作系统版本</th>
                <th>基础硬件</th>
                <th>UserData 模板</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in vp.serverImages" :key="s.id">
                <td>{{ s.platform_type_display || platformLabel(s.platform_type) }}</td>
                <td>{{ s.image_name }}</td>
                <td class="mono">{{ s.image_id }}</td>
                <td>{{ s.region }}</td>
                <td>{{ s.architecture || "—" }}</td>
                <td>{{ s.os_version || "—" }}</td>
                <td class="hw-cell">{{ vp.hardwareSummary(s) }}</td>
                <td>{{ userdataTemplateDisplayLabel(s.userdata_template, vp.userdataTemplateOptions) }}</td>
                <td>
                  <button class="btn-mini" type="button" @click="vp.openCsEdit(s)">编辑</button>
                  <button class="btn-mini danger" type="button" @click="vp.removeServer(s)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
    <div v-if="vp.csOpen" class="modal-bg" @click.self="vp.closeCsModal">
      <div class="card modal wide">
        <h3>{{ vp.csEditingId ? "编辑云平台服务器镜像" : "登记云平台服务器镜像" }}</h3>
        <p v-if="!vp.csEditingId" class="hint">选择云平台和地域，从云平台获取镜像列表</p>
        <p v-else class="hint">可修改镜像名称、启用状态，以及该镜像默认使用的基础硬件规格（云平台/地域/镜像 ID 登记后不可改）。</p>

        <template v-if="!vp.csEditingId">
          <label class="lbl">云平台</label>
          <select v-model="vp.csForm.platform_type" class="inp" @change="vp.onPlatformChange">
            <option v-for="p in cloudPlatforms" :key="p.value" :value="p.value">{{ p.label }}</option>
          </select>

          <div v-if="vp.csLoadingRegions" class="hint">加载地域列表中…</div>
          <div v-else-if="vp.csRegions.length > 0">
            <label class="lbl">地域</label>
            <select v-model="vp.csForm.region" class="inp" @change="vp.onRegionChange">
              <option value="">请选择地域</option>
              <option v-for="r in vp.csRegions" :key="r.id" :value="r.id">{{ r.name }} ({{ r.id }})</option>
            </select>
          </div>

          <div v-if="vp.csLoadingImages" class="hint">加载镜像列表中…</div>
          <div v-else-if="vp.csImages.length > 0">
            <label class="lbl">镜像</label>
            <input
              v-model="vp.csImageSearchKeyword"
              class="inp"
              type="text"
              placeholder="输入镜像名称 / 镜像 ID / 系统关键词过滤"
            />
            <p v-if="vp.csImageSearchKeyword && vp.filteredCsImages.length === 0" class="hint">
              未匹配到镜像，请调整关键词
            </p>
            <select v-model="vp.csForm.selected_image_id" class="inp" @change="vp.onImageChange">
              <option value="">请选择镜像</option>
              <option v-for="img in vp.filteredCsImages" :key="img.id" :value="img.id">
                {{ img.name }} ({{ img.id }}) - {{ img.os_type || "" }} {{ img.architecture || "" }}
              </option>
            </select>
          </div>
        </template>

        <template v-else>
          <label class="lbl">云平台</label>
          <input class="inp" :value="platformLabel(vp.csForm.platform_type)" readonly />
          <label class="lbl">地域</label>
          <input v-model="vp.csForm.region" class="inp" readonly />
          <label class="lbl">镜像 ID</label>
          <input v-model="vp.csForm.image_id" class="inp" readonly />
          <label class="lbl">架构</label>
          <input v-model="vp.csForm.architecture" class="inp" readonly />
          <label class="lbl">操作系统类型</label>
          <input v-model="vp.csForm.os_type" class="inp" />
          <label class="lbl">操作系统版本</label>
          <input v-model="vp.csForm.os_version" class="inp" />
        </template>

        <div v-if="vp.csEditingId || vp.csForm.selected_image_id">
          <label class="lbl">镜像名称（可修改）</label>
          <input v-model="vp.csForm.image_name" class="inp" />
          <template v-if="!vp.csEditingId">
            <label class="lbl">镜像 ID</label>
            <input v-model="vp.csForm.image_id" class="inp" readonly />
            <label class="lbl">操作系统类型</label>
            <input v-model="vp.csForm.os_type" class="inp" readonly />
            <label class="lbl">操作系统版本</label>
            <input v-model="vp.csForm.os_version" class="inp" readonly />
            <label class="lbl">架构</label>
            <input v-model="vp.csForm.architecture" class="inp" readonly />
            <label class="lbl">镜像类型</label>
            <input v-model="vp.csForm.image_type" class="inp" readonly />
            <label class="lbl">系统盘大小 (GiB)</label>
            <input v-model="vp.csForm.image_size_gb" class="inp" readonly />
          </template>

          <label class="lbl flex-cb">
            <span>启用</span>
            <input v-model="vp.csForm.is_active" type="checkbox" class="chk" />
          </label>

          <h4 class="cs-hw-title">基础硬件环境</h4>
          <p class="hint">
            将按<strong>当前镜像可运行的规格</strong>与<strong>地域内可售资源（库存）</strong>过滤后再展示（阿里云：DescribeImageSupportInstanceTypes + DescribeAvailableResource）。亦可仅手填
            vCPU/内存。需已配置云平台凭证。
          </p>
          <button
            class="btn btn-ghost row-inline"
            type="button"
            :disabled="!vp.csForm.region || !vp.csForm.image_id || vp.csLoadingInstanceTypes"
            @click="vp.fetchCsInstanceTypes"
          >
            {{ vp.csLoadingInstanceTypes ? "加载中…" : "加载实例规格列表" }}
          </button>
          <label class="lbl">默认实例规格</label>
          <select v-model="vp.csForm.default_instance_type_id" class="inp" @change="vp.onCsInstanceTypeChange">
            <option value="">不指定</option>
            <option
              v-for="it in vp.csInstanceTypes"
              :key="it.instance_type_id"
              :value="it.instance_type_id"
            >
              {{ it.instance_type_id }} — {{ it.cpu_core_count }}vCPU / {{ it.memory_size }}GiB
            </option>
          </select>
          <label class="lbl">规格说明（可自动生成或手改）</label>
          <input v-model="vp.csForm.default_instance_type_label" class="inp" />
          <label class="lbl">vCPU 核数</label>
          <input v-model="vp.csForm.base_cpu_cores" class="inp" type="number" min="1" step="1" placeholder="可选" />
          <label class="lbl">内存 (GiB)</label>
          <input v-model="vp.csForm.base_memory_gib" class="inp" type="number" min="1" step="1" placeholder="可选" />
          <label class="lbl">UserData 模板版本</label>
          <select v-model="vp.csForm.userdata_template_id" class="inp">
            <option value="">不选用</option>
            <option v-for="t in vp.userdataTemplateOptions" :key="t.id" :value="String(t.id)">
              {{ t.name }} v{{ t.version }}（{{ t.os_type }}）
            </option>
          </select>
          <p class="hint">在平台审核 → UserData 模板中维护；此处仅选用已启用的模板。</p>
        </div>

        <p v-if="vp.csError" class="err" v-bind="vp.csErrorTraceId ? { 'data-traceId': vp.csErrorTraceId } : {}">{{ vp.csError }}</p>

        <div class="row">
          <button class="btn btn-ghost" type="button" @click="vp.closeCsModal">取消</button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="(!vp.csEditingId && !vp.csForm.selected_image_id) || vp.csSaving"
            @click="vp.saveCs"
          >
            {{ vp.csSaving ? "保存中…" : "保存" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { userdataTemplateDisplayLabel } from "../utils/userdataTemplateDisplay.js";
import { CLOUD_PLATFORMS as cloudPlatforms, platformLabel } from "../utils/platformLabel.js";

defineProps({
  vp: { type: Object, required: true },
});
</script>

<style scoped src="../views/VendorPortal.css"></style>
