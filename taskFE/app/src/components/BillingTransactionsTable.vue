<template>
  <div class="bg-white rounded-xl shadow-md p-6" data-alias="BillingTransactionsTable">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-lg font-semibold">交易记录</h2>
      <div class="flex items-center gap-3">
        <HeaderSelectFilter
          :model-value="paymentSource"
          :options="paymentSourceOptions"
          placeholder="全部"
          aria-label="支付来源"
          input-class="px-3 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          @update:model-value="onPaymentSourceChange"
        />
        <div class="text-sm text-gray-500">
          共 {{ totalCount }} 条记录
        </div>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b">
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500 align-top">
              <span class="block mb-1">交易时间</span>
              <HeaderDateFilter
                :start-date="startDate"
                :end-date="endDate"
                start-label="开始日期"
                end-label="结束日期"
                input-class="block w-40 px-2 py-1 border rounded-lg text-sm font-normal focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                @commit="onDateCommit"
              />
            </th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">
              <HeaderSelectFilter
                :model-value="transactionTypeFilter"
                :options="transactionTypeOptions"
                placeholder="全部类型"
                aria-label="交易类型"
                input-class="px-2 py-1 border rounded-lg text-sm font-normal focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                @update:model-value="onTransactionTypeChange"
              />
            </th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">
              <HeaderSelectFilter
                :model-value="billingUnitTypeFilter"
                :options="billingUnitTypeOptions"
                placeholder="全部分类"
                aria-label="消耗（元）分类"
                input-class="px-2 py-1 border rounded-lg text-sm font-normal focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                @update:model-value="onBillingUnitTypeChange"
              />
            </th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">变动明细</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">瞬时账目</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">有效期至</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500 align-top">
              <div class="relative project-filter">
                <span class="block mb-1">项目</span>
                <HeaderSearchFilter
                  :search="projectSearch"
                  :options="projectOptions"
                  :show-dropdown="showProjectDropdown"
                  aria-label="项目名称"
                  placeholder="输入项目名称搜索"
                  wrapper-class="project-filter"
                  dropdown-class="project-filter"
                  input-class="w-full px-2 py-1 border rounded-lg text-sm font-normal focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  @update:search="v => emit('update:projectSearch', v)"
                  @search="emit('search-projects')"
                  @focus="emit('focus-project')"
                  @select="o => emit('select-project', o)"
                />
              </div>
            </th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500 align-top">
              <div class="relative user-filter">
                <span class="block mb-1">成员</span>
                <HeaderSearchFilter
                  :search="userSearch"
                  :options="userOptions"
                  :show-dropdown="showUserDropdown"
                  aria-label="成员名称"
                  placeholder="输入成员名称搜索"
                  wrapper-class="user-filter"
                  dropdown-class="user-filter"
                  input-class="w-full px-2 py-1 border rounded-lg text-sm font-normal focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  @update:search="v => emit('update:userSearch', v)"
                  @search="emit('search-users')"
                  @focus="emit('focus-user')"
                  @select="o => emit('select-user', o)"
                />
              </div>
            </th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">描述</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="transaction in rowsWithLines"
            :key="transaction.id"
            class="border-b hover:bg-gray-50"
          >
            <td class="py-3 px-4 text-sm">{{ formatDate(transaction.created_at) }}</td>
            <td class="py-3 px-4 text-sm">
              <span
                :class="{
                  'text-green-600': transaction.transaction_type === 'recharge',
                  'text-red-600': transaction.transaction_type === 'consumption',
                  'text-blue-600': transaction.transaction_type === 'refund'
                }"
              >
                {{ getTransactionTypeText(transaction.transaction_type) }}
              </span>
            </td>
            <td class="py-3 px-4 text-sm">{{ transaction.billing_unit?.name || pointsSourceTypeDisplayOrDash(transaction) }}</td>
            <td class="py-3 px-4 text-sm font-medium">
              <span
                :class="{
                  'text-green-600': transaction.transaction_type === 'recharge',
                  'text-red-600': transaction.transaction_type === 'consumption',
                  'text-blue-600': transaction.transaction_type === 'refund'
                }"
              >
                {{ transactionChangeDisplay(transaction) }}
              </span>
            </td>
            <td class="py-3 px-4 text-sm text-gray-800">
              <div
                v-for="(line, idx) in transaction._lines"
                :key="idx"
                :class="{ 'mt-0.5 text-gray-600': idx > 0 }"
              >
                {{ line }}
              </div>
              <span v-if="transaction._lines.length === 0">—</span>
            </td>
            <td class="py-3 px-4 text-sm whitespace-nowrap">
              <template v-if="transaction.transaction_type === 'recharge' && transaction.expires_at">
                {{ formatDate(transaction.expires_at) }}
              </template>
              <template v-else>—</template>
            </td>
            <td class="py-3 px-4 text-sm">
              <router-link
                v-if="transaction.project_id"
                :to="`/tenant/${tenantId}/projects/${transaction.project_id}/`"
                class="text-primary hover:underline"
              >
                {{ transaction.project_name || transaction.project_id }}
              </router-link>
              <template v-else>-</template>
            </td>
            <td class="py-3 px-4 text-sm">{{ transaction.user_name || transaction.user_id || '-' }}</td>
            <td class="py-3 px-4 text-sm">{{ transaction.description || '-' }}</td>
          </tr>
          <tr v-if="rowsWithLines.length === 0">
            <td colspan="9" class="py-8 text-center text-gray-500">暂无交易记录</td>
          </tr>
        </tbody>
      </table>
    </div>

    <BillingPaginationNav
      :current-page="currentPage"
      :total-pages="totalPages"
      @change-page="$emit('change-page', $event)"
    />
  </div>
</template>

<script setup>
import BillingPaginationNav from './BillingPaginationNav.vue'
import HeaderDateFilter from './HeaderDateFilter.vue'
import HeaderSelectFilter from './HeaderSelectFilter.vue'
import HeaderSearchFilter from './HeaderSearchFilter.vue'
import { computed } from 'vue'
import { ledgerSnapshotLines, pointsSourceTypeDisplayOrDash, transactionChangeDisplay } from '../utils/transactionChangeDisplay.js'

const props = defineProps({
  tenantId: { type: [String, Number], required: true },
  transactions: { type: Array, default: () => [] },
  totalCount: { type: Number, default: 0 },
  currentPage: { type: Number, required: true },
  totalPages: { type: Number, required: true },
  formatDate: { type: Function, required: true },
  getTransactionTypeText: { type: Function, required: true },
  paymentSource: { type: String, default: '' },
  paymentSourceOptions: { type: Array, default: () => [] },
  transactionTypeFilter: { type: String, default: '' },
  billingUnitTypeFilter: { type: String, default: '' },
  billingUnitTypeOptions: { type: Array, default: () => [] },
  startDate: { type: String, default: '' },
  endDate: { type: String, default: '' },
  projectSearch: { type: String, default: '' },
  projectOptions: { type: Array, default: () => [] },
  showProjectDropdown: { type: Boolean, default: false },
  userSearch: { type: String, default: '' },
  userOptions: { type: Array, default: () => [] },
  showUserDropdown: { type: Boolean, default: false }
})

// OPT-20260819-016: 行级 ledgerSnapshotLines 只计算一次（模板 v-for 与空态 v-if
// 各调一次会造成重复解析），把结果展开到每行 _lines。
const rowsWithLines = computed(() =>
  (props.transactions || []).map((txn) => ({ ...txn, _lines: ledgerSnapshotLines(txn) })),
)

// 交易类型列头 select 选项（固定枚举，自 Django 迁移后保持不变）
const transactionTypeOptions = [
  { value: 'recharge', label: '入账' },
  { value: 'consumption', label: '消耗' },
  { value: 'refund', label: '退款' },
]

const emit = defineEmits([
  'update:filter',
  'update:projectSearch',
  'update:userSearch',
  'search-projects',
  'search-users',
  'focus-project',
  'focus-user',
  'select-project',
  'select-user',
  'apply',
  'change-page'
])

// 表头 select/日期变更即自动应用（OPT-20260807-045 前已有语义）
const onPaymentSourceChange = (value) => {
  emit('update:filter', 'pointsSourceType', value)
  emit('apply')
}

const onTransactionTypeChange = (value) => {
  emit('update:filter', 'transactionType', value)
  emit('apply')
}

const onBillingUnitTypeChange = (value) => {
  emit('update:filter', 'billingUnitType', value)
  emit('apply')
}

const onDateCommit = ({ key, value }) => {
  emit('update:filter', key, value)
  emit('apply')
}
</script>
