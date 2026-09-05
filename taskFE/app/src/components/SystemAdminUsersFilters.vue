<template>
  <td class="align-top px-2 py-2">
    <HeaderTextFilter
      :model-value="filters.id"
      aria-label="ID"
      placeholder="ID"
      data-alias="SystemAdminUsersFilterId"
      @update:model-value="v => $emit('update:filter', 'id', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderTextFilter
      :model-value="filters.email"
      aria-label="邮箱"
      placeholder="邮箱"
      data-alias="SystemAdminUsersFilterEmail"
      @update:model-value="v => $emit('update:filter', 'email', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderTextFilter
      :model-value="filters.tenant_company"
      aria-label="所属租户公司"
      placeholder="公司"
      data-alias="SystemAdminUsersFilterTenantCompany"
      @update:model-value="v => $emit('update:filter', 'tenant_company', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderTextFilter
      :model-value="filters.phone"
      aria-label="手机号"
      placeholder="手机号"
      data-alias="SystemAdminUsersFilterPhone"
      @update:model-value="v => $emit('update:filter', 'phone', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderSelectFilter
      :model-value="filters.login_method"
      :options="loginMethodOptions"
      placeholder="全部方式"
      aria-label="登录方式"
      data-alias="SystemAdminUsersFilterLoginMethod"
      @update:model-value="v => $emit('update:filter', 'login_method', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderTextFilter
      :model-value="filters.referrer"
      aria-label="推荐人"
      placeholder="推荐人"
      data-alias="SystemAdminUsersFilterReferrer"
      @update:model-value="v => $emit('update:filter', 'referrer', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderDateFilter
      :start-date="filters.date_joined_from"
      :end-date="filters.date_joined_to"
      start-alias="SystemAdminUsersFilterJoinedFrom"
      end-alias="SystemAdminUsersFilterJoinedTo"
      start-label="注册开始日期"
      end-label="注册结束日期"
      @update:start-date="v => $emit('update:filter', 'date_joined_from', v)"
      @update:end-date="v => $emit('update:filter', 'date_joined_to', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderDateFilter
      :start-date="filters.last_login_from"
      :end-date="filters.last_login_to"
      start-alias="SystemAdminUsersFilterLastLoginFrom"
      end-alias="SystemAdminUsersFilterLastLoginTo"
      start-label="最后活跃开始日期"
      end-label="最后活跃结束日期"
      @update:start-date="v => $emit('update:filter', 'last_login_from', v)"
      @update:end-date="v => $emit('update:filter', 'last_login_to', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderSelectFilter
      :model-value="filters.is_active"
      :options="statusOptions"
      placeholder="全部状态"
      aria-label="状态"
      data-alias="SystemAdminUsersFilterStatus"
      @update:model-value="v => $emit('update:filter', 'is_active', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderSelectFilter
      :model-value="filters.role"
      :options="roleOptions"
      placeholder="全部角色"
      aria-label="角色"
      data-alias="SystemAdminUsersFilterRole"
      @update:model-value="v => $emit('update:filter', 'role', v)"
    />
  </td>
  <td class="align-top px-2 py-2">
    <HeaderSelectFilter
      :model-value="filters.has_profit_sharing"
      :options="profitSharingOptions"
      placeholder="全部"
      aria-label="是否获得分账资格"
      data-alias="SystemAdminUsersFilterProfitSharing"
      @update:model-value="v => $emit('update:filter', 'has_profit_sharing', v)"
    />
  </td>
  <td class="align-top px-2 py-2 whitespace-nowrap">
    <!-- Anti-Replay-OK: 只读列表过滤重置，发 GET 不写资源 -->
    <button
      type="button"
      data-testid="user-list-column-filters-reset"
      class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm hover:bg-gray-50 transition-colors"
      @click="$emit('reset')"
    >
      重置
    </button>
  </td>
</template>

<script setup>
import HeaderTextFilter from './HeaderTextFilter.vue'
import HeaderSelectFilter from './HeaderSelectFilter.vue'
import HeaderDateFilter from './HeaderDateFilter.vue'

defineProps({
  filters: { type: Object, required: true },
})

defineEmits(['update:filter', 'reset'])

const loginMethodOptions = [
  { value: 'email', label: '邮箱' },
  { value: 'phone', label: '手机' },
  { value: 'username', label: '用户名' },
]
const statusOptions = [
  { value: 'true', label: '活跃' },
  { value: 'false', label: '已禁用' },
]
const roleOptions = [
  { value: 'superuser', label: '超管' },
  { value: 'staff', label: '员工' },
  { value: 'tester', label: '测试' },
  { value: 'tenant', label: '租户' },
  { value: 'user', label: '用户' },
]
const profitSharingOptions = [
  { value: 'true', label: '是' },
  { value: 'false', label: '否' },
]
</script>
