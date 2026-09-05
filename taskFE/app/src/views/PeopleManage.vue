<template>
  <!-- OPT-20260809-025: 非管理员友好提示 — 先探权限，无权时展示「无权限」空态而非原始 403 -->
  <div v-if="!accessChecked" class="people-manage-container p-6 text-text-light">加载中…</div>
  <div v-else-if="!canManage" class="people-manage-container p-6">
    <div class="bg-white p-6 rounded-xl shadow text-center py-16">
      <p class="text-xl font-bold">无权限访问</p>
      <p class="text-text-light mt-3">人员管理仅对拥有成员管理权限的公司成员开放，如需访问请联系公司管理员。</p>
    </div>
  </div>
  <div v-else class="people-manage-container p-6">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-bold">管理人员</h2>
      <!-- 公司切换下拉框 -->
      <div v-if="companies.length >= 1" class="relative">
        <select
          v-model="selectedCompanyId"
          @change="switchCompany"
          class="px-4 py-2 border border-border rounded-lg bg-white focus:ring-2 focus:ring-primary focus:border-primary transition-colors text-sm"
        >
          <option
            v-for="c in companies"
            :key="c.company_id"
            :value="c.company_id"
          >
            🏢 {{ c.company_name }} {{ c.is_admin || c.is_creator ? '(管理员)' : '(成员)' }}
          </option>
        </select>
      </div>
    </div>

    <!-- 成员列表 -->
    <MemberList :key="tenantId" :tenant-id="tenantId" />

    <!-- 待处理邀请 -->
    <PendingInvitations :key="'invite-' + tenantId" :tenant-id="tenantId" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils.js'
import { usePermissions } from '../composables/usePermissions.js'
import MemberList from '../components/MemberList.vue'
import PendingInvitations from '../components/PendingInvitations.vue'

const route = useRoute()
const router = useRouter()

// tenantId 跟随路由变化，切换公司时自动更新
const tenantId = computed(() => route.params.tenant)
const selectedCompanyId = ref(route.params.tenant)
const companies = ref([])

const switchCompany = () => {
  if (selectedCompanyId.value && selectedCompanyId.value !== tenantId.value) {
    router.push(`/tenant/${selectedCompanyId.value}/people/manage/`)
  }
}

// 路由变化时同步下拉框选中值
watch(() => route.params.tenant, (newTenant) => {
  selectedCompanyId.value = newTenant
})

const fetchCompanies = async () => {
  try {
    const userId = getCookie('userId')
    if (!userId) return
    const resp = await apiFetch(`/api/accounts/users/me/`, {
      credentials: 'include',
      headers: { 'Accept': 'application/json' }
    })
    if (!resp.ok) return
    const data = await resp.json()
    // API 返回 companies: [{id, name, is_admin}, ...] — 不是 company_nicknames
    const rawCompanies = data.companies || []
    if (rawCompanies.length > 0) {
      companies.value = rawCompanies.map(c => ({
        company_id: c.id,
        company_name: c.name,
        // v63 RBAC: 管理员标签改用权限码判定（PDP 展开，自定义角色带 member:manage 亦为管理员）
        is_admin: perms.hasPerm(c.id, 'member:manage'),
        is_creator: data.current_company?.id === c.id && !!c.is_admin,
      }))
      // 确保当前 URL tenant 在列表中（URL 中的 tenant 可能来自直接链接）
      const current = companies.value.find(c => c.company_id === tenantId.value)
      if (!current) {
        companies.value.push({
          company_id: tenantId.value,
          company_name: '当前公司',
          is_admin: false,
          is_creator: false
        })
      }
    }
  } catch (e) {
    console.error('获取公司列表失败:', e)
  }
}

const perms = usePermissions()

// OPT-20260809-025: 权限已探测完成才渲染内容；canManage 按 v63 RBAC 权限码判定
//（member:manage 与后端 accounts 成员管理接口强制一致，自定义角色带权限码亦视为管理员）
const accessChecked = ref(false)
const canManage = computed(() => perms.hasPerm(tenantId.value, 'member:manage'))

onMounted(async () => {
  await perms.load(apiFetch)
  accessChecked.value = true
  if (canManage.value) {
    fetchCompanies()
  }
})
</script>

<style scoped>
.people-manage-container {
  min-height: 80vh;
}
</style>