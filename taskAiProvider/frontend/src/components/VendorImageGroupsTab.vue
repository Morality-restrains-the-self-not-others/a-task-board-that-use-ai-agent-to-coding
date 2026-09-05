<template>
        <div class="row-between">
          <h2 class="h2">我的镜像组</h2>
          <button class="btn btn-primary" type="button" @click="vp.openGroupCreate">创建镜像组</button>
        </div>
        <div class="group-list">
          <div v-for="g in vp.imageGroups" :key="g.id" class="group-item">
            <div class="group-header" @click="vp.toggleGroup(g.id)">
              <div class="group-info">
                <img
                  v-if="g.icon_url"
                  class="group-icon"
                  :src="g.icon_url"
                  :alt="g.name + ' 图标'"
                />
                <span v-else class="group-icon group-icon-placeholder" aria-hidden="true"></span>
                <span class="group-name">{{ g.name }}</span>
                <span class="group-desc">{{ g.description || '—' }}</span>
                <span class="group-count">{{ g.versions_count }} 个版本</span>
                <span v-if="g.latest_version" class="group-latest">
                  最新：{{ g.latest_version.version }}
                  <span :class="vp.badgeClass(g.latest_version.status)">{{ g.latest_version.status_display || vp.statusDisplay(g.latest_version.status) }}</span>
                </span>
                <span v-else class="hint">无版本</span>
              </div>
              <div class="group-actions">
                <button class="btn-mini" type="button" @click.stop="vp.openGroupEdit(g)">编辑</button>
                <button class="btn-mini primary" type="button" @click.stop="vp.openAddVersion(g)">添加版本</button>
                <button class="btn-mini danger" type="button" @click.stop="vp.removeGroup(g)">删除</button>
                <span class="expand-icon">{{ vp.expandedGroups[g.id] ? '▼' : '▶' }}</span>
              </div>
            </div>
            <div v-if="vp.expandedGroups[g.id]" class="group-versions">
              <div v-if="vp.getGroupVersions(g.id).length === 0" class="no-version">
                <span class="hint">暂无版本，点击"添加版本"创建</span>
              </div>
              <div v-else class="version-list">
                <div v-for="im in vp.getGroupVersions(g.id)" :key="im.id" class="version-item">
                  <div class="version-row">
                    <span class="version-id" :title="String(im.id)">{{ im.id }}</span>
                    <span class="version-num" :title="im.version || ''">{{ im.version || "—" }}</span>
                    <span
                      class="version-num"
                      data-testid="saas-inbound-skill-version-cell"
                      :title="'容器→SaaS 接口 ' + (im.saas_inbound_skill_version ? ('v' + im.saas_inbound_skill_version) : '')"
                    >接口 v{{ im.saas_inbound_skill_version || "—" }}</span>
                    <span class="version-status">
                      <span :class="vp.badgeClass(im.status)">{{ im.status_display || vp.statusDisplay(im.status) }}</span>
                      <span
                        v-if="vp.isActiveVersion(im)"
                        class="badge badge-version-active"
                        title="当前激活版本：在公开镜像市场对外生效"
                      >当前激活</span>
                      <span
                        v-if="vp.groupActiveVersion(im)"
                        class="badge badge-group-active"
                        :title="`组内当前激活版本为「${vp.groupActiveVersion(im).version}」，可将其设为激活版本完成切换`"
                      >组内激活：{{ vp.groupActiveVersion(im).version }}</span>
                      <span
                        v-if="vp.isMarketplaceUnavailable(im)"
                        class="badge badge-unavailable"
                        :title="im.unavailable_reason || ''"
                      >{{ vp.formatUnavailableBadge(im.unavailable_reason) }}</span>
                      <span
                        v-if="vp.versionHasUnavailableRegion(im.id)"
                        class="badge badge-unavailable badge-region-nouserdata"
                        title="部分区域已绑定服务器镜像但未选择 UserData 模板"
                      >区域未配置模板</span>
                    </span>
                    <div class="version-actions" data-testid="version-row-actions">
                      <button
                        v-if="vp.versionHasAction(im, 'edit')"
                        class="btn-mini"
                        type="button"
                        :disabled="vp.versionActionBusy"
                        @click="vp.openEdit(im)"
                      >
                        编辑
                      </button>
                      <button class="btn-mini" type="button" @click="vp.toggleRuntimeEnv(im.id)">
                        {{ vp.expandedRuntimeEnvs[im.id] ? '收起区域明细' : '展开区域明细' }}
                      </button>
                      <button
                        v-if="vp.versionHasAction(im, 'delete')"
                        class="btn-mini danger"
                        type="button"
                        :disabled="vp.versionActionBusy"
                        :aria-busy="vp.versionActionBusy ? 'true' : 'false'"
                        @click="vp.openVersionDelete(im)"
                      >
                        删除
                      </button>
                      <button
                        v-if="vp.versionHasAction(im, 'submit')"
                        class="btn-mini primary"
                        type="button"
                        :disabled="vp.versionActionBusy"
                        :aria-busy="vp.versionActionBusy ? 'true' : 'false'"
                        @click="vp.submitReview(im)"
                      >
                        提交审核
                      </button>
                      <button
                        v-if="vp.versionHasAction(im, 'rejectNote')"
                        class="btn-mini"
                        type="button"
                        @click="vp.toggleRejectNote(im.id)"
                      >
                        {{ vp.expandedRejectNotes[im.id] ? '收起原因' : '查看原因' }}
                      </button>
                      <button
                        v-if="vp.versionHasAction(im, 'activate')"
                        class="btn-mini primary"
                        type="button"
                        :disabled="vp.versionActionBusy"
                        :aria-busy="vp.versionActionBusy ? 'true' : 'false'"
                        @click="vp.openVersionActivate(im)"
                      >
                        设为激活
                      </button>
                      <button
                        v-if="vp.versionHasAction(im, 'withdraw')"
                        class="btn-mini danger"
                        type="button"
                        :disabled="vp.versionActionBusy"
                        :aria-busy="vp.versionActionBusy ? 'true' : 'false'"
                        @click="vp.openVersionWithdraw(im)"
                      >
                        {{ vp.withdrawButtonLabel(im.status) }}
                      </button>
                    </div>
                    <span class="version-url mono" :title="im.image_url">{{ im.image_url }}</span>
                  </div>
                  <div class="version-resolve" data-testid="version-resolve-summary">
                    <span
                      v-if="vp.versionResolveSummaries[im.id]?.skillNames.length"
                      class="resolve-skills"
                      :title="vp.versionResolveSummaries[im.id].skillNames.join('、')"
                    >
                      技能：{{ vp.versionResolveSummaries[im.id].skillNames.join('、') }}
                    </span>
                    <span v-else class="resolve-hint">{{ vp.versionResolveSummaries[im.id]?.skillsHint }}</span>
                    <span
                      v-if="vp.versionResolveSummaries[im.id]?.autoRunPreview"
                      class="resolve-auto-run"
                      :title="vp.versionResolveSummaries[im.id].autoRunPreview"
                    >
                      自动运行：{{ vp.versionResolveSummaries[im.id].autoRunPreview }}
                    </span>
                    <span v-else class="resolve-hint">{{ vp.versionResolveSummaries[im.id]?.autoRunHint }}</span>
                  </div>
                  <div v-if="(im.status === 'rejected' || im.status === 'draft') && im.review_note && vp.expandedRejectNotes[im.id]" class="reject-note">
                    <strong>{{ im.status === 'rejected' ? '驳回原因：' : '撤销上架原因：' }}</strong>{{ im.review_note }}
                    <span v-if="im.reviewed_at" class="reject-time">（{{ vp.formatDate(im.reviewed_at) }}）</span>
                  </div>
                  <div v-if="vp.isMarketplaceUnavailable(im) && im.unavailable_reason" class="unavailable-note">
                    <strong>不可用原因：</strong>{{ im.unavailable_reason }}
                  </div>
                  <div v-if="vp.expandedRuntimeEnvs[im.id]" class="runtime-env-panel">
                    <div class="runtime-env-header">
                      <h4>各区域运行环境</h4>
                    </div>
                    <div class="runtime-env-list">
                      <div class="runtime-env-list-header">
                        <span class="col-platform">云平台</span>
                        <span class="col-region">区域</span>
                        <span class="col-image">服务器镜像</span>
                        <span class="col-userdata">UserData 模板</span>
                        <span class="col-status">状态</span>
                      </div>
                      <template v-for="platform in vp.cloudPlatforms" :key="platform.value">
                        <template v-if="vp.loadingRegions[platform.value]">
                          <div class="runtime-env-list-item loading">
                            <span class="col-platform">{{ platform.label }}</span>
                            <span class="col-region">加载中...</span>
                            <span class="col-image">-</span>
                            <span class="col-userdata">-</span>
                            <span class="col-status">-</span>
                          </div>
                        </template>
                        <template v-else-if="vp.getPlatformRegions(platform.value).length === 0">
                          <div class="runtime-env-list-item unset">
                            <span class="col-platform">{{ platform.label }}</span>
                            <span class="col-region">-</span>
                            <span class="col-image">-</span>
                            <span class="col-userdata">-</span>
                            <span class="col-status">
                              <span class="status-badge unset">暂无区域数据</span>
                            </span>
                          </div>
                        </template>
                        <template v-else>
                          <div
                            v-for="region in vp.getPlatformRegions(platform.value)"
                            :key="region.id"
                            :class="['runtime-env-list-item', vp.getRuntimeEnvRowClass(im.id, platform.value, region.id)]"
                          >
                            <span class="col-platform">{{ platform.label }}</span>
                            <span class="col-region" :title="`${region.name} (${region.id})`">{{ region.name }} ({{ region.id }})</span>
                            <template v-if="vp.getRuntimeEnvForRegion(im.id, platform.value, region.id)">
                              <span class="col-image">
                                <span class="image-name">{{ vp.getRuntimeEnvForRegion(im.id, platform.value, region.id).cloud_server_image?.image_name }}</span>
                                <span class="image-id mono">{{ vp.getRuntimeEnvForRegion(im.id, platform.value, region.id).cloud_server_image?.image_id }}</span>
                              </span>
                              <span
                                :class="[
                                  'col-userdata',
                                  'userdata-cell',
                                  vp.runtimeEnvHasUserdataTemplate(vp.getRuntimeEnvForRegion(im.id, platform.value, region.id)) ? '' : 'userdata-missing',
                                ]"
                              >{{
                                vp.runtimeEnvUserdataTemplateLabel(vp.getRuntimeEnvForRegion(im.id, platform.value, region.id))
                              }}</span>
                              <span class="col-status">
                                <span
                                  v-if="vp.getRuntimeEnvStatus(im.id, platform.value, region.id) === 'set'"
                                  class="status-badge set"
                                >已设置</span>
                                <span
                                  v-else-if="vp.getRuntimeEnvStatus(im.id, platform.value, region.id) === 'unavailable'"
                                  class="status-badge unavailable"
                                  title="已绑定服务器镜像但未选择 UserData 模板"
                                >不可用</span>
                                <button class="btn-mini settings-btn" type="button" @click.stop="vp.openRegionEnv(im, platform.value, region)">设置</button>
                              </span>
                            </template>
                            <template v-else>
                              <span class="col-image">-</span>
                              <span class="col-userdata">-</span>
                              <span class="col-status">
                                <span class="status-badge unset">未设置</span>
                                <button class="btn-mini settings-btn" type="button" @click.stop="vp.openRegionEnv(im, platform.value, region)">设置</button>
                              </span>
                            </template>
                          </div>
                        </template>
                      </template>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div v-if="vp.imageGroups.length === 0" class="empty-msg">
            <p class="hint">暂无镜像组，点击"创建镜像组"添加</p>
          </div>
        </div>
</template>

<script setup>
defineProps({
  vp: { type: Object, required: true },
});
</script>

<style scoped src="../views/VendorPortal.css"></style>
