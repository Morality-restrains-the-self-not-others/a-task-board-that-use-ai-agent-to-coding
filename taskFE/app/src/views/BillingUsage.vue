<template>
  <div class="p-6" data-alias="BillingUsage">
    <h1 class="text-2xl font-bold mb-6">资源使用记录</h1>

    <BillingUsageTable
      :tenant-id="tenantId"
      :usage-records="usageRecords"
      :total-count="totalCount"
      :current-page="currentPage"
      :total-pages="totalPages"
      :format-date="formatDate"
      @change-page="changePage"
    >
      <template #filters>
        <BillingUsageFilters
          :filters="filters"
          :billing-unit-type-options="billingUnitTypeOptions"
          :project-search="projectSearch"
          :project-options="projectOptions"
          :show-project-dropdown="showProjectDropdown"
          :selected-project-name="selectedProjectName"
          :workspace-search="workspaceSearch"
          :workspace-options="workspaceOptions"
          :show-workspace-dropdown="showWorkspaceDropdown"
          :selected-workspace-name="selectedWorkspaceName"
          :user-search="userSearch"
          :user-options="userOptions"
          :show-user-dropdown="showUserDropdown"
          :selected-user-name="selectedUserName"
          :task-search="taskSearch"
          :task-options="taskOptions"
          :show-task-dropdown="showTaskDropdown"
          :selected-task-name="selectedTaskName"
          @update:filter="onUpdateFilter"
          @update:project-search="projectSearch = $event"
          @search-projects="searchProjects"
          @open-project-dropdown="openProjectDropdown"
          @select-project="selectProject"
          @update:workspace-search="workspaceSearch = $event"
          @search-workspaces="searchWorkspaces"
          @open-workspace-dropdown="openWorkspaceDropdown"
          @select-workspace="selectWorkspace"
          @update:user-search="userSearch = $event"
          @search-users="searchUsers"
          @open-user-dropdown="openUserDropdown"
          @select-user="selectUser"
          @update:task-search="taskSearch = $event"
          @search-tasks="searchTasks"
          @open-task-dropdown="openTaskDropdown"
          @select-task="selectTask"
          @apply="applyFilters"
          @reset="resetFilters"
        />
      </template>
    </BillingUsageTable>
  </div>
</template>

<script setup>
import BillingUsageFilters from '../components/BillingUsageFilters.vue'
import BillingUsageTable from '../components/BillingUsageTable.vue'
import { useBillingUsage } from '../composables/useBillingUsage.js'

const {
  tenantId,
  filters,
  billingUnitTypeOptions,
  usageRecords,
  currentPage,
  totalPages,
  totalCount,
  projectSearch,
  selectedProjectName,
  showProjectDropdown,
  projectOptions,
  workspaceSearch,
  selectedWorkspaceName,
  showWorkspaceDropdown,
  workspaceOptions,
  userSearch,
  selectedUserName,
  showUserDropdown,
  userOptions,
  taskSearch,
  selectedTaskName,
  showTaskDropdown,
  taskOptions,
  formatDate,
  searchProjects,
  openProjectDropdown,
  selectProject,
  searchWorkspaces,
  openWorkspaceDropdown,
  selectWorkspace,
  searchUsers,
  openUserDropdown,
  selectUser,
  searchTasks,
  openTaskDropdown,
  selectTask,
  applyFilters,
  resetFilters,
  changePage
} = useBillingUsage()

const onUpdateFilter = (key, value) => {
  filters.value[key] = value
}
</script>
