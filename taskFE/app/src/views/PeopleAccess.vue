<template>
  <div v-if="!accessChecked" class="people-access-container p-6 text-text-light">加载中…</div>
  <div v-else-if="!canManage" class="people-access-container p-6">
    <div class="bg-white p-6 rounded-xl shadow text-center py-16">
      <p class="text-xl font-bold">无权限访问</p>
      <p class="text-text-light mt-3">访问管理仅对拥有访问配置权限的公司成员开放，如需访问请联系公司管理员。</p>
    </div>
  </div>
  <div v-else class="people-access-container p-6">
    <div class="flex flex-wrap items-start justify-between gap-3 mb-6">
      <div>
        <h2 class="text-2xl font-bold mb-2">访问管理</h2>
        <p class="text-text-light text-sm">
          为成员或小组分配一个或多个角色；权限为多角色并集。请先在
          <router-link class="text-primary underline" :to="`${tenantPath}/people/roles/`">角色管理</router-link>
          中配置可复用角色。
        </p>
      </div>
      <button
        v-if="orphanRoles.length"
        type="button"
        class="px-3 py-2 rounded-lg border border-border text-sm text-text-light hover:bg-primary/5 disabled:opacity-50"
        data-testid="cleanup-orphan-access-roles"
        :disabled="cleaningOrphans || cleanupOrphansGuard.isBusy()"
        @click="cleanupOrphanRoles"
      >
        {{ cleaningOrphans ? '清理中…' : `清理未绑定访问角色 (${orphanRoles.length})` }}
      </button>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div class="bg-white rounded-xl shadow-md p-4 lg:col-span-1" data-rg-key="people.access.subject_list">
        <div class="flex gap-2 mb-4">
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm transition-colors"
            :class="tab === 'member' ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light'"
            @click="tab = 'member'"
          >
            成员
          </button>
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-sm transition-colors"
            :class="tab === 'group' ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text-light'"
            @click="tab = 'group'"
          >
            小组
          </button>
        </div>
        <div v-if="listLoading" class="text-text-light text-sm py-8 text-center">加载列表…</div>
        <ul v-else class="space-y-1 max-h-[28rem] overflow-y-auto">
          <li v-for="row in subjectRows" :key="row.id">
            <button
              type="button"
              class="w-full text-left px-3 py-2 rounded-lg text-sm transition-colors"
              :class="selectedId === row.id ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-primary/5 text-text'"
              @click="selectSubject(row)"
            >
              <span class="block truncate">{{ row.label }}</span>
              <span class="block text-xs text-text-light truncate">
                角色：{{ row.roleNames.length ? row.roleNames.join('、') : '（未分配）' }}
              </span>
            </button>
          </li>
          <li v-if="!subjectRows.length" class="text-text-light text-sm py-8 text-center">暂无数据</li>
        </ul>
      </div>

      <div class="bg-white rounded-xl shadow-md p-6 lg:col-span-2">
        <div
          v-if="loadError"
          class="mb-4 text-red-600 text-sm"
          data-testid="people-access-catalog-error"
          :data-traceId="loadErrorTraceId || undefined"
        >
          {{ loadError }}
        </div>
        <div v-if="!selectedId" class="text-text-light text-center py-16">请选择左侧成员或小组</div>
        <div v-else>
          <div class="flex items-start justify-between gap-4 mb-4">
            <div>
              <h3 class="text-lg font-bold">{{ selectedLabel }}</h3>
              <p class="text-text-light text-sm mt-1">勾选要分配的角色（可多选，权限取并集）</p>
            </div>
            <button
              type="button"
              class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:opacity-90 disabled:opacity-50"
              data-rg-key="people.access.save_actions"
              :disabled="saving || !canSaveAccess || saveRolesGuard.isBusy()"
              data-testid="people-access-save"
              @click="onSave"
            >
              {{ saving ? '保存中…' : '保存' }}
            </button>
          </div>

          <div
            v-if="saveMessage"
            class="mb-4 text-sm"
            :class="saveOk ? 'text-green-700' : 'text-red-600'"
            :data-traceId="saveTraceId || undefined"
          >
            {{ saveMessage }}
          </div>

          <div class="space-y-2 mb-6" data-testid="people-access-role-picker">
            <label
              v-for="role in pickerRoles"
              :key="role.id || role.name"
              class="flex items-start gap-2 px-3 py-2 rounded-lg border border-border text-sm cursor-pointer hover:bg-primary/5"
            >
              <input
                type="checkbox"
                class="mt-0.5 rounded border-border"
                :checked="selectedRoleNames.includes(role.name)"
                @change="toggleRole(role.name, $event.target.checked)"
              />
              <span class="min-w-0">
                <span class="font-medium block truncate">{{ role.display_name || role.name }}</span>
                <span class="text-xs text-text-light">{{ role.is_system ? '系统' : '自定义' }} · {{ role.name }}</span>
              </span>
            </label>
            <p v-if="!pickerRoles.length" class="text-text-light text-sm py-6 text-center">
              暂无可分配角色，请先到角色管理创建
            </p>
          </div>

          <div class="border-t border-border pt-4">
            <h4 class="text-sm font-medium mb-2">有效权限预览（只读并集）</h4>
            <p v-if="!previewKeys.length" class="text-text-light text-sm">未选择角色或所选角色尚无 page/region 授权。</p>
            <ul v-else class="text-xs text-text-light space-y-1 max-h-40 overflow-y-auto">
              <li v-for="k in previewKeys" :key="k">{{ k }}</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { usePermissions } from '../composables/usePermissions.js'
import {
  assignableRoles,
  assignSubjectRoles,
  roleNamesForSubject,
} from '../domain/auth/assignSubjectRoles.js'
import { listOrphanCustomAccessRoles } from '../domain/auth/saveSubjectResourceAccess.js'
import { fetchRoleBoundResourceGroups, hasPeopleAccessWrite } from './peopleAccessCatalog.js'
import { loadPeopleAccessLists } from './peopleAccessLoad.js'

const route = useRoute()
const perms = usePermissions()
const tenantId = computed(() => String(route.params.tenant || ''))
const tenantPath = computed(() => `/tenant/${tenantId.value}`)

const accessChecked = ref(false)
const canManage = computed(() => {
  const cid = tenantId.value
  const view =
    typeof perms.hasRegionView === 'function' ? perms.hasRegionView.bind(perms) : perms.hasRegion
  return (
    view(cid, 'people.access.save_actions') ||
    view(cid, 'people.access.region_matrix') ||
    view(cid, 'people.access.subject_list') ||
    perms.hasPage(cid, 'people.access') ||
    perms.hasPerm(cid, 'member:manage')
  )
})

const canSaveAccess = computed(() => hasPeopleAccessWrite(perms, tenantId.value))

const tab = ref('member')
const listLoading = ref(false)
const members = ref([])
const groups = ref([])
const memberRoles = ref([])
const groupRoles = ref([])
const roles = ref([])
const catalogPages = ref([])

const selectedId = ref('')
const selectedLabel = ref('')
const selectedRoleNames = ref([])
const previewKeys = ref([])

const saving = ref(false)
const cleaningOrphans = ref(false)
// OPT-20260819-038: 角色分配/清理孤儿角色是写操作，防连点/超时重试双发
const saveRolesGuard = createClickGuard()
const cleanupOrphansGuard = createClickGuard()
const saveMessage = ref('')
const saveOk = ref(false)
const saveTraceId = ref('')
const loadError = ref('')
const loadErrorTraceId = ref('')

const orphanRoles = computed(() =>
  listOrphanCustomAccessRoles({
    roles: roles.value,
    memberRoles: memberRoles.value,
    groupRoles: groupRoles.value,
  }),
)

const pickerRoles = computed(() => assignableRoles(roles.value))

const subjectRows = computed(() => {
  if (tab.value === 'member') {
    return members.value.map((m) => {
      const id = String(m.id)
      const roleNames = roleNamesForSubject(memberRoles.value, 'member', id)
      return {
        id,
        label: m.member_name || m.name || m.user_name || m.id,
        roleNames,
      }
    })
  }
  return groups.value.map((g) => {
    const id = String(g.id)
    return {
      id,
      label: g.name || g.id,
      roleNames: roleNamesForSubject(groupRoles.value, 'group', id),
    }
  })
})

function toggleRole(name, checked) {
  const set = new Set(selectedRoleNames.value)
  if (checked) set.add(name)
  else set.delete(name)
  selectedRoleNames.value = [...set]
  refreshPreview()
}

async function refreshPreview() {
  const keys = new Set()
  for (const name of selectedRoleNames.value) {
    const role = roles.value.find((r) => r.name === name)
    if (!role) continue
    if (name === 'tenant_admin') {
      keys.add('(tenant_admin：全部系统 page/region)')
      continue
    }
    if (role.is_system) {
      for (const p of role.permissions || []) keys.add(p)
      continue
    }
    const bound = await fetchRoleBoundResourceGroups({
      apiFetch,
      role,
      companyId: tenantId.value,
    })
    for (const b of bound || []) {
      if (b?.group_key) keys.add(`${b.group_key}:${b.effect || 'operate'}`)
    }
  }
  previewKeys.value = [...keys].sort()
}

async function selectSubject(row) {
  selectedId.value = row.id
  selectedLabel.value = row.label
  selectedRoleNames.value = [...(row.roleNames || [])]
  saveMessage.value = ''
  await refreshPreview()
}

async function loadLists() {
  await loadPeopleAccessLists({
    tenantId: tenantId.value,
    state: {
      listLoading,
      loadError,
      loadErrorTraceId,
      members,
      groups,
      memberRoles,
      groupRoles,
      roles,
      catalogPages,
    },
  })
}

async function onSave() {
  // OPT-20260819-038: 角色分配是写操作，防连点/超时重试双发 PUT
  await saveRolesGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    saveMessage.value = ''
    saveTraceId.value = ''
    try {
      await assignSubjectRoles({
        apiFetch,
        companyId: tenantId.value,
        subjectType: tab.value,
        subjectId: selectedId.value,
        roleNames: selectedRoleNames.value,
        idempotencyKey,
      })
      saveOk.value = true
      saveMessage.value = '已保存角色分配。目标用户刷新后权限按多角色并集生效。'
      await loadLists()
      await perms.reload(apiFetch)
    } catch (e) {
      saveOk.value = false
      saveMessage.value = e?.message || '保存失败'
      saveTraceId.value = e?.traceId || e?.trace_id || ''
    } finally {
      saving.value = false
    }
  })
}

async function cleanupOrphanRoles() {
  const targets = orphanRoles.value
  if (!targets.length) return
  // OPT-20260819-038: 清理孤儿角色是批量删除，防连点双发
  await cleanupOrphansGuard.run(async ({ idempotencyKey }) => {
    cleaningOrphans.value = true
    saveMessage.value = ''
    let deleted = 0
    try {
      for (const role of targets) {
        const resp = await apiFetch(`/api/auth/roles/role_id/${role.id}/`, {
          method: 'DELETE',
          credentials: 'include',
          headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
        })
        if (resp.ok) deleted += 1
        else if (resp.status === 422) continue
        else {
          const err = await resp.json().catch(() => null)
          throw new Error(err?.detail || err?.message || `删除角色失败 (${resp.status})`)
        }
      }
      saveOk.value = true
      saveMessage.value = deleted ? `已清理 ${deleted} 个未绑定访问角色` : '无可删除的未绑定角色'
      await loadLists()
    } catch (e) {
      saveOk.value = false
      saveMessage.value = e?.message || '清理失败'
      saveTraceId.value = e?.traceId || ''
    } finally {
      cleaningOrphans.value = false
    }
  })
}

watch(tab, () => {
  selectedId.value = ''
  selectedLabel.value = ''
  selectedRoleNames.value = []
  previewKeys.value = []
  saveMessage.value = ''
})

onMounted(async () => {
  await perms.load(apiFetch)
  accessChecked.value = true
  if (canManage.value) await loadLists()
})
</script>
