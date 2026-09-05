<template>
  <div class="p-6" data-alias="BillingTransactions">
    <h1 class="text-2xl font-bold mb-6">交易流水</h1>

    <BillingTransactionsFilters
      :filters="filters"
      :workspace-search="workspaceSearch"
      :task-search="taskSearch"
      :workspace-options="workspaceOptions"
      :task-options="taskOptions"
      :show-workspace-dropdown="showWorkspaceDropdown"
      :show-task-dropdown="showTaskDropdown"
      :selected-workspace-name="selectedWorkspaceName"
      :selected-task-name="selectedTaskName"
      @update:filter="onUpdateFilter"
      @update:workspace-search="workspaceSearch = $event"
      @update:task-search="taskSearch = $event"
      @search-workspaces="searchWorkspaces"
      @search-tasks="searchTasks"
      @focus-workspace="openWorkspaceDropdown"
      @focus-task="openTaskDropdown"
      @select-workspace="selectWorkspace"
      @select-task="selectTask"
      @apply="applyFilters"
      @reset="resetFilters"
    />

    <BillingTransactionsTable
      :tenant-id="tenantId"
      :transactions="transactions"
      :total-count="totalCount"
      :current-page="currentPage"
      :total-pages="totalPages"
      :format-date="formatDate"
      :get-transaction-type-text="getTransactionTypeText"
      :payment-source="filters.pointsSourceType"
      :payment-source-options="rechargeSourceOptions"
      :transaction-type-filter="filters.transactionType"
      :billing-unit-type-filter="filters.billingUnitType"
      :billing-unit-type-options="billingUnitTypeOptions"
      :start-date="filters.startDate"
      :end-date="filters.endDate"
      :project-search="projectSearch"
      :project-options="projectOptions"
      :show-project-dropdown="showProjectDropdown"
      :user-search="userSearch"
      :user-options="userOptions"
      :show-user-dropdown="showUserDropdown"
      @update:filter="onUpdateFilter"
      @update:project-search="projectSearch = $event"
      @update:user-search="userSearch = $event"
      @search-projects="searchProjects"
      @search-users="searchUsers"
      @focus-project="openProjectDropdown"
      @focus-user="openUserDropdown"
      @select-project="(option) => { selectProject(option); applyFilters() }"
      @select-user="(option) => { selectUser(option); applyFilters() }"
      @apply="applyFilters"
      @change-page="changePage"
    />
  </div>
</template>

<script setup>
import BillingTransactionsFilters from '../components/BillingTransactionsFilters.vue'
import BillingTransactionsTable from '../components/BillingTransactionsTable.vue'
import { useBillingTransactions } from '../composables/useBillingTransactions.js'

const {
  tenantId,
  filters,
  billingUnitTypeOptions,
  rechargeSourceOptions,
  transactions,
  currentPage,
  totalPages,
  totalCount,
  projectSearch,
  userSearch,
  workspaceSearch,
  taskSearch,
  projectOptions,
  userOptions,
  workspaceOptions,
  taskOptions,
  showProjectDropdown,
  showUserDropdown,
  showWorkspaceDropdown,
  showTaskDropdown,
  selectedWorkspaceName,
  selectedTaskName,
  formatDate,
  getTransactionTypeText,
  searchProjects,
  searchUsers,
  searchWorkspaces,
  searchTasks,
  openProjectDropdown,
  openUserDropdown,
  openWorkspaceDropdown,
  openTaskDropdown,
  selectProject,
  selectUser,
  selectWorkspace,
  selectTask,
  applyFilters,
  resetFilters,
  changePage
} = useBillingTransactions()

const onUpdateFilter = (key, value) => {
  filters.value[key] = value
}
</script>
