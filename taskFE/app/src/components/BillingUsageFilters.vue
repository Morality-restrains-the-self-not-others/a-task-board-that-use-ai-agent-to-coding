<template>
  <!-- 时间范围 → 使用时间列 -->
  <td class="align-top py-2 px-4">
    <HeaderDateFilter
      :start-date="filters.startDate"
      :end-date="filters.endDate"
      start-alias="BillingUsageStartDate"
      end-alias="BillingUsageEndDate"
      start-label="开始日期"
      end-label="结束日期"
      @update:start-date="v => $emit('update:filter', 'startDate', v)"
      @update:end-date="v => $emit('update:filter', 'endDate', v)"
    />
  </td>

  <!-- 计费单元类型 → 计费单元列 -->
  <td class="align-top py-2 px-4">
    <HeaderSelectFilter
      :model-value="filters.billingUnitType"
      :options="billingUnitTypeOptions"
      placeholder="全部类型"
      aria-label="计费单元类型"
      data-alias="BillingUsageUnitType"
      @update:model-value="v => $emit('update:filter', 'billingUnitType', v)"
    />
  </td>

  <!-- 使用量列无过滤条件 -->
  <td class="align-top py-2 px-4"></td>

  <!-- 项目名称 → 项目列 -->
  <td class="relative align-top py-2 px-4 project-filter">
    <HeaderSearchFilter
      :search="projectSearch"
      :options="projectOptions"
      :show-dropdown="showProjectDropdown"
      :selected-name="selectedProjectName"
      aria-label="项目名称"
      placeholder="项目名称或缩略词"
      data-alias="BillingUsageProjectSearch"
      wrapper-class="project-filter"
      :show-id="true"
      @update:search="v => $emit('update:projectSearch', v)"
      @search="$emit('search-projects')"
      @focus="$emit('open-project-dropdown')"
      @select="o => $emit('select-project', o)"
    />
  </td>

  <!-- 成员名称 → 用户列 -->
  <td class="relative align-top py-2 px-4 user-filter">
    <HeaderSearchFilter
      :search="userSearch"
      :options="userOptions"
      :show-dropdown="showUserDropdown"
      :selected-name="selectedUserName"
      aria-label="成员名称"
      placeholder="成员名称或缩略词"
      data-alias="BillingUsageUserSearch"
      wrapper-class="user-filter"
      :show-id="true"
      @update:search="v => $emit('update:userSearch', v)"
      @search="$emit('search-users')"
      @focus="$emit('open-user-dropdown')"
      @select="o => $emit('select-user', o)"
    />
  </td>

  <!-- 工作空间名称 → 工作空间列 -->
  <td class="relative align-top py-2 px-4 workspace-filter">
    <HeaderSearchFilter
      :search="workspaceSearch"
      :options="workspaceOptions"
      :show-dropdown="showWorkspaceDropdown"
      :selected-name="selectedWorkspaceName"
      aria-label="工作空间名称"
      placeholder="工作空间名称或缩略词"
      data-alias="BillingUsageWorkspaceSearch"
      wrapper-class="workspace-filter"
      :show-id="true"
      @update:search="v => $emit('update:workspaceSearch', v)"
      @search="$emit('search-workspaces')"
      @focus="$emit('open-workspace-dropdown')"
      @select="o => $emit('select-workspace', o)"
    />
  </td>

  <!-- 任务名称 → 任务列 -->
  <td class="relative align-top py-2 px-4 task-filter">
    <HeaderSearchFilter
      :search="taskSearch"
      :options="taskOptions"
      :show-dropdown="showTaskDropdown"
      :selected-name="selectedTaskName"
      aria-label="任务名称"
      placeholder="任务名称或缩略词"
      data-alias="BillingUsageTaskSearch"
      wrapper-class="task-filter"
      :show-id="true"
      @update:search="v => $emit('update:taskSearch', v)"
      @search="$emit('search-tasks')"
      @focus="$emit('open-task-dropdown')"
      @select="o => $emit('select-task', o)"
    />
  </td>

  <!-- 应用过滤/重置 → 描述列 -->
  <td class="align-top py-2 px-4 whitespace-nowrap">
    <div class="flex gap-1.5">
      <button
        data-alias="BillingUsageApply"
        class="px-3 py-1.5 bg-primary text-white rounded-lg text-sm hover:bg-primary/90 transition-colors"
        @click="$emit('apply')"
      >
        应用过滤
      </button>
      <button
        data-alias="BillingUsageReset"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm hover:bg-gray-50 transition-colors"
        @click="$emit('reset')"
      >
        重置
      </button>
    </div>
  </td>
</template>

<script setup>
import HeaderDateFilter from './HeaderDateFilter.vue'
import HeaderSelectFilter from './HeaderSelectFilter.vue'
import HeaderSearchFilter from './HeaderSearchFilter.vue'

defineProps({
  filters: { type: Object, required: true },
  billingUnitTypeOptions: { type: Array, default: () => [] },
  projectSearch: { type: String, default: '' },
  projectOptions: { type: Array, default: () => [] },
  showProjectDropdown: { type: Boolean, default: false },
  selectedProjectName: { type: String, default: '' },
  workspaceSearch: { type: String, default: '' },
  workspaceOptions: { type: Array, default: () => [] },
  showWorkspaceDropdown: { type: Boolean, default: false },
  selectedWorkspaceName: { type: String, default: '' },
  userSearch: { type: String, default: '' },
  userOptions: { type: Array, default: () => [] },
  showUserDropdown: { type: Boolean, default: false },
  selectedUserName: { type: String, default: '' },
  taskSearch: { type: String, default: '' },
  taskOptions: { type: Array, default: () => [] },
  showTaskDropdown: { type: Boolean, default: false },
  selectedTaskName: { type: String, default: '' }
})

defineEmits([
  'update:filter',
  'update:projectSearch',
  'search-projects',
  'open-project-dropdown',
  'select-project',
  'update:workspaceSearch',
  'search-workspaces',
  'open-workspace-dropdown',
  'select-workspace',
  'update:userSearch',
  'search-users',
  'open-user-dropdown',
  'select-user',
  'update:taskSearch',
  'search-tasks',
  'open-task-dropdown',
  'select-task',
  'apply',
  'reset'
])
</script>
