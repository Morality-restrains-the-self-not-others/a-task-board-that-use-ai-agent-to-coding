<template>
  <div v-if="!accessChecked" class="people-roles-container p-6 text-text-light">加载中…</div>
  <div v-else-if="!canManage" class="people-roles-container p-6">
    <div class="bg-white p-6 rounded-xl shadow text-center py-16">
      <p class="text-xl font-bold">无权限访问</p>
      <p class="text-text-light mt-3">角色管理仅对拥有相应权限的公司成员开放。</p>
    </div>
  </div>
  <div v-else class="people-roles-container p-6" data-rg-key="people.roles.main">
    <div class="flex flex-wrap items-start justify-between gap-3 mb-6">
      <div>
        <h2 class="text-2xl font-bold mb-2">角色管理</h2>
        <p class="text-text-light text-sm">
          创建可复用角色并配置页面/区域权限；再到「访问管理」分配给成员或小组。
        </p>
      </div>
      <button
        type="button"
        class="px-3 py-2 rounded-lg bg-primary text-white text-sm hover:opacity-90 disabled:opacity-50"
        data-testid="people-roles-create"
        data-rg-key="people.roles.save_actions"
        :disabled="!canSave || creating"
        @click="onCreate"
      >
        {{ creating ? '创建中…' : '新建角色' }}
      </button>
    </div>

    <div
      v-if="banner"
      class="mb-4 text-sm"
      :class="bannerOk ? 'text-green-700' : 'text-red-600'"
      :data-traceId="bannerTrace || undefined"
    >
      {{ banner }}
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div class="bg-white rounded-xl shadow-md p-4 lg:col-span-1">
        <ul class="space-y-1 max-h-[32rem] overflow-y-auto">
          <li v-for="role in roleList" :key="role.id">
            <button
              type="button"
              class="w-full text-left px-3 py-2 rounded-lg text-sm transition-colors"
              :class="selectedId === role.id ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text'"
              @click="selectRole(role)"
            >
              <span class="block truncate">{{ role.display_name || role.name }}</span>
              <span class="block text-xs text-text-light">
                {{ role.is_system ? '系统角色' : '自定义' }} · {{ role.name }}
              </span>
            </button>
          </li>
          <li v-if="!roleList.length" class="text-text-light text-sm py-8 text-center">暂无角色</li>
        </ul>
      </div>

      <div class="bg-white rounded-xl shadow-md p-6 lg:col-span-2">
        <div v-if="!selectedId" class="text-text-light text-center py-16">请选择左侧角色</div>
        <div v-else>
          <div class="flex items-start justify-between gap-4 mb-4">
            <div class="flex-1 min-w-0">
              <label class="block text-sm font-medium mb-1" for="people-roles-display-name">显示名</label>
              <input
                id="people-roles-display-name"
                v-model="editDisplayName"
                type="text"
                class="w-full rounded-lg border border-border px-3 py-2 text-sm"
                :disabled="selectedIsSystem || !canSave"
                maxlength="64"
              />
              <p v-if="selectedIsSystem" class="text-amber-700 text-sm mt-2">
                系统角色不可修改授权；tenant_admin 默认拥有全部页面与区域。
              </p>
            </div>
            <div class="flex gap-2 shrink-0">
              <button
                v-if="!selectedIsSystem"
                type="button"
                class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:opacity-90 disabled:opacity-50"
                data-testid="people-roles-save"
                data-rg-key="people.roles.save_actions"
                :disabled="saving || !canSave"
                @click="onSave"
              >
                {{ saving ? '保存中…' : '保存' }}
              </button>
              <button
                v-if="!selectedIsSystem"
                type="button"
                class="px-4 py-2 rounded-lg border border-border text-sm text-red-600 hover:bg-red-50 disabled:opacity-50"
                data-testid="people-roles-delete"
                :disabled="deleting || !canSave || isRoleBound"
                :title="isRoleBound ? '请先在访问管理中解绑该角色' : ''"
                @click="onDelete"
              >
                {{ deleting ? '删除中…' : '删除' }}
              </button>
            </div>
          </div>

          <div v-if="!selectedIsSystem" data-rg-key="people.roles.main">
            <input
              v-model="catalogFilter"
              type="search"
              class="w-full rounded-lg border border-border px-3 py-2 text-sm mb-4"
              placeholder="筛选页面或区域…"
              data-testid="people-roles-catalog-filter"
            />
            <p v-if="catalogFilter && !filteredCatalogPages.length" class="text-sm text-text-light py-3 text-center">
              无匹配页面或区域
            </p>
            <ResourceGrantMatrix
              v-if="filteredCatalogPages.length || !catalogFilter"
              :pages="filteredCatalogPages"
              :model-value="selectedGrants"
              @update:model-value="onMatrixUpdate"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { usePermissions } from '../composables/usePermissions.js'
import { isSystemTenantRole } from '../domain/auth/tenantConsoleNav.js'
import { fetchRoleBoundResourceGroups, filterCatalogPages } from './peopleAccessCatalog.js'
import ResourceGrantMatrix from '../components/ResourceGrantMatrix.vue'
import { createTenantRole, saveRoleResourceGrants } from '../domain/auth/saveRoleResourceGrants.js'

const route = useRoute()
const perms = usePermissions()
const tenantId = computed(() => String(route.params.tenant || ''))

const accessChecked = ref(false)
const canManage = computed(() => {
  const cid = tenantId.value
  return (
    perms.hasPage(cid, 'people.roles') ||
    perms.hasRegion(cid, 'people.roles.main') ||
    perms.hasPerm(cid, 'member:manage')
  )
})
const canSave = computed(() => {
  const cid = tenantId.value
  if (typeof perms.hasRegionOperate === 'function') {
    return perms.hasRegionOperate(cid, 'people.roles.save_actions') || perms.hasPerm(cid, 'member:manage')
  }
  return perms.hasRegion(cid, 'people.roles.save_actions') || perms.hasPerm(cid, 'member:manage')
})

const roles = ref([])
const memberRoles = ref([])
const groupRoles = ref([])
const catalogPages = ref([])
const catalogFilter = ref('')
const filteredCatalogPages = computed(() => filterCatalogPages(catalogPages.value, catalogFilter.value))
const selectedId = ref('')
const editDisplayName = ref('')
/** @type {import('vue').Ref<Record<string, 'view'|'operate'>>} */
const selectedGrants = ref({})
const saving = ref(false)
const creating = ref(false)
const deleting = ref(false)
// OPT-20260819-038: 创建/保存/删除角色均为写操作，防连点/超时重试双发
const createRoleGuard = createClickGuard()
const saveRoleGuard = createClickGuard()
const deleteRoleGuard = createClickGuard()
const banner = ref('')
const bannerOk = ref(false)
const bannerTrace = ref('')

const roleList = computed(() =>
  [...roles.value].sort((a, b) => {
    if (Boolean(a.is_system) !== Boolean(b.is_system)) return a.is_system ? -1 : 1
    return String(a.display_name || a.name).localeCompare(String(b.display_name || b.name), 'zh')
  }),
)

const selectedRole = computed(() => roles.value.find((r) => r.id === selectedId.value) || null)
const selectedIsSystem = computed(
  () => Boolean(selectedRole.value?.is_system) || isSystemTenantRole(selectedRole.value?.name),
)

const isRoleBound = computed(() => {
  const name = selectedRole.value?.name
  if (!name) return false
  return (
    memberRoles.value.some((r) => r.role_name === name) ||
    groupRoles.value.some((r) => r.role_name === name)
  )
})

/** 共享矩阵更新（view/operate 效果映射） */
function onMatrixUpdate(next) {
  selectedGrants.value = next
}

async function loadAll() {
  const cid = tenantId.value
  const [rolesResp, mrResp, grResp, catResp] = await Promise.all([
    apiFetch(`/api/auth/roles/company_id/${cid}/`, { credentials: 'include', headers: { Accept: 'application/json' } }),
    apiFetch(`/api/tenant/member-role/company_id/${cid}/`, { credentials: 'include', headers: { Accept: 'application/json' } }),
    apiFetch(`/api/tenant/group-role/company_id/${cid}/`, { credentials: 'include', headers: { Accept: 'application/json' } }),
    apiFetch(`/api/auth/resource-groups/?company_id=${cid}`, { credentials: 'include', headers: { Accept: 'application/json' } }),
  ])
  if (rolesResp.ok) {
    const body = await rolesResp.json()
    roles.value = Array.isArray(body) ? body : []
  }
  if (mrResp.ok) {
    const body = await mrResp.json()
    memberRoles.value = Array.isArray(body) ? body : []
  }
  if (grResp.ok) {
    const body = await grResp.json()
    groupRoles.value = Array.isArray(body) ? body : []
  }
  if (catResp.ok) {
    const body = await catResp.json()
    catalogPages.value = Array.isArray(body.pages) ? body.pages : []
  }
}

async function selectRole(role) {
  selectedId.value = role.id
  editDisplayName.value = role.display_name || role.name
  banner.value = ''
  const isSys = Boolean(role.is_system) || isSystemTenantRole(role.name)
  if (isSys) {
    selectedGrants.value = {}
    return
  }
  selectedGrants.value = await fetchRoleBoundResourceGroups({
    apiFetch,
    role,
    companyId: tenantId.value,
  }).then((bound) => {
    const map = {}
    for (const b of bound || []) {
      if (b?.group_key) map[b.group_key] = b.effect === 'view' ? EFFECT_VIEW : EFFECT_OPERATE
    }
    return map
  })
}

async function onCreate() {
  const name = window.prompt('新角色显示名')
  if (!name || !String(name).trim()) return
  // OPT-20260819-038: 创建角色是写操作，防连点双发
  await createRoleGuard.run(async () => {
    creating.value = true
    banner.value = ''
    try {
      const created = await createTenantRole({
        apiFetch,
        companyId: tenantId.value,
        displayName: String(name).trim(),
      })
      await loadAll()
      const role = roles.value.find((r) => r.id === created.id) || created
      await selectRole(role)
      bannerOk.value = true
      banner.value = '已创建角色，请勾选页面/区域后保存'
    } catch (e) {
      bannerOk.value = false
      banner.value = e?.message || '创建失败'
      bannerTrace.value = e?.traceId || ''
    } finally {
      creating.value = false
    }
  })
}

async function onSave() {
  if (!selectedRole.value || selectedIsSystem.value) return
  // OPT-20260819-038: 保存角色授权是写操作，防连点双发
  await saveRoleGuard.run(async () => {
    saving.value = true
    banner.value = ''
    try {
      await saveRoleResourceGrants({
        apiFetch,
        companyId: tenantId.value,
        roleId: selectedRole.value.id,
        displayName: editDisplayName.value.trim() || selectedRole.value.display_name,
        selectedGrants: selectedGrants.value,
      })
      bannerOk.value = true
      banner.value = '角色授权已保存'
      await loadAll()
    } catch (e) {
      bannerOk.value = false
      banner.value = e?.message || '保存失败'
      bannerTrace.value = e?.traceId || ''
    } finally {
      saving.value = false
    }
  })
}

async function onDelete() {
  if (!selectedRole.value || selectedIsSystem.value || isRoleBound.value) return
  if (!window.confirm(`确认删除角色「${selectedRole.value.display_name || selectedRole.value.name}」？`)) return
  // OPT-20260819-038: 删除角色是写操作，防连点/超时重试双发 DELETE
  await deleteRoleGuard.run(async ({ idempotencyKey }) => {
    deleting.value = true
    banner.value = ''
    try {
      const resp = await apiFetch(`/api/auth/roles/role_id/${selectedRole.value.id}/`, {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => null)
        throw new Error(err?.detail || err?.message || `删除失败 (${resp.status})`)
      }
      selectedId.value = ''
      selectedGrants.value = {}
      bannerOk.value = true
      banner.value = '角色已删除'
      await loadAll()
    } catch (e) {
      bannerOk.value = false
      banner.value = e?.message || '删除失败'
      bannerTrace.value = e?.traceId || ''
    } finally {
      deleting.value = false
    }
  })
}

onMounted(async () => {
  await perms.load(apiFetch)
  accessChecked.value = true
  if (canManage.value) await loadAll()
})
</script>
