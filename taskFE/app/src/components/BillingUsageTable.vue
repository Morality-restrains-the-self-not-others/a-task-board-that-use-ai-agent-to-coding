<template>
  <div class="bg-white rounded-xl shadow-md p-6" data-alias="BillingUsageTable">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-lg font-semibold">使用记录</h2>
      <div class="text-sm text-gray-500">
        共 {{ totalCount }} 条记录
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b">
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">使用时间</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">计费单元</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">使用量</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">项目</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">用户</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">工作空间</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">任务</th>
            <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">描述</th>
          </tr>
          <tr
            v-if="$slots.filters"
            data-alias="BillingUsageFiltersRow"
            class="border-b bg-gray-50/60"
          >
            <slot name="filters" />
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="record in usageRecords"
            :key="record.id"
            class="border-b hover:bg-gray-50"
          >
            <td class="py-3 px-4 text-sm">{{ formatDate(record.usage_time) }}</td>
            <td class="py-3 px-4 text-sm">{{ record.billing_unit?.name || '-' }}</td>
            <td class="py-3 px-4 text-sm font-medium">
              {{ record.amount }} {{ record.billing_unit?.unit || '-' }}
            </td>
            <td class="py-3 px-4 text-sm">
              <router-link
                v-if="record.project_id"
                :to="`/tenant/${tenantId}/projects/${record.project_id}/`"
                class="text-primary hover:underline"
              >
                {{ record.project_name || record.project_id }}
              </router-link>
              <template v-else>-</template>
            </td>
            <td class="py-3 px-4 text-sm">{{ record.user_name || record.user_id || '-' }}</td>
            <td class="py-3 px-4 text-sm">{{ record.workspace_name || record.workspace_id || '-' }}</td>
            <td class="py-3 px-4 text-sm">
              <router-link
                v-if="record.task_id && record.workspace_id"
                :to="`/tenant/${tenantId}/workspace/${record.workspace_id}/task-detail/${record.task_id}/`"
                class="text-primary hover:underline"
              >
                {{ record.task_name || record.task_id }}
              </router-link>
              <template v-else>
                {{ record.task_name || record.task_id || '-' }}
              </template>
            </td>
            <td class="py-3 px-4 text-sm">{{ record.description || '-' }}</td>
          </tr>
          <tr v-if="usageRecords.length === 0">
            <td colspan="8" class="py-8 text-center text-gray-500">暂无使用记录</td>
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

defineProps({
  tenantId: { type: [String, Number], required: true },
  usageRecords: { type: Array, default: () => [] },
  totalCount: { type: Number, default: 0 },
  currentPage: { type: Number, required: true },
  totalPages: { type: Number, required: true },
  formatDate: { type: Function, required: true }
})

defineEmits(['change-page'])
</script>
