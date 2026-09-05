<template>
  <tr :class="user.is_archived ? 'bg-gray-50 opacity-75' : ''">
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 font-mono" :title="String(user.id)">{{ user.id }}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
      <template v-if="user.email">{{ user.email }}
        <span v-if="user.email_verified === false" class="ml-1 px-1.5 py-0.5 text-xs rounded bg-yellow-100 text-yellow-700">未验证</span>
      </template>
      <template v-else-if="user.phone">{{ user.phone }}
        <span class="ml-1 px-1.5 py-0.5 text-xs rounded bg-gray-100 text-gray-500">手机</span>
      </template>
      <template v-else-if="user.username">{{ user.username }}</template>
      <template v-else><span class="text-gray-400 font-mono text-xs">{{ user.id }}</span></template>
    </td>
    <td class="px-6 py-4 text-sm text-gray-700" data-testid="user-tenant-companies">{{ tenantCompanyLabel(user) }}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ user.phone || '—' }}</td>
    <td class="px-6 py-4 whitespace-nowrap">
      <span v-for="lm in (user.login_methods || [])" :key="lm.method_type"
        class="inline-block px-2 py-0.5 text-xs rounded-full mr-1"
        :class="lm.is_verified ? 'bg-blue-100 text-blue-700' : 'bg-yellow-100 text-yellow-700'">
        {{ { email: '邮箱', phone: '手机', username: '用户名' }[lm.method_type] || lm.method_type }}
      </span>
    </td>
    <td class="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-500">{{ user.referrer_code || '—' }}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ user.date_joined ? new Date(user.date_joined).toLocaleDateString('zh-CN') : '—' }}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" data-testid="user-last-login">{{ lastActiveLabel(user) }}</td>
    <td class="px-6 py-4 whitespace-nowrap">
      <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
        :class="user.is_archived ? 'bg-gray-100 text-gray-600' : (user.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800')">
        {{ user.is_archived ? '已归档' : (user.is_active ? '活跃' : '已禁用') }}
      </span>
    </td>
    <td class="px-6 py-4 whitespace-nowrap">
      <span
        class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
        data-testid="user-role-badge"
        :class="roleBadgeClass(user)"
      >
        {{ roleBadgeLabel(user) }}
      </span>
    </td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-700" data-testid="user-profit-sharing-qualification">{{ profitSharingQualificationLabel(user) }}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
      <button
        v-if="canImpersonate"
        data-testid="user-row-impersonate-btn"
        class="text-amber-700 hover:text-amber-900 mr-3"
        @click="handleImpersonate"
      >
        以该用户身份登录
      </button>
      <!-- Anti-Replay-OK: real href navigation to login history -->
      <a
        :href="`/system-admin/users/${user.id}/login-history/`"
        class="text-primary hover:text-primary/90 mr-3"
        data-testid="user-row-login-history"
      >登录历史</a>
      <button class="text-primary hover:text-primary/90 mr-3" @click="handleEdit">
        编辑
      </button>
      <button class="text-primary hover:text-primary/90 mr-3" @click="handleKyc">
        KYC
      </button>
      <button class="text-primary hover:text-primary/90 mr-3" @click="handleReferral">
        推荐绩效
      </button>
      <button class="text-primary hover:text-primary/90 mr-3" @click="handleRecharge">
        支付与签署
      </button>
      <template v-if="!user.is_archived">
        <button class="text-red-600 hover:text-red-800 mr-3" @click="handleDelete">
          禁用
        </button>
        <button class="text-orange-600 hover:text-orange-800" @click="handleArchive">
          归档
        </button>
      </template>
      <template v-else>
        <button class="text-green-600 hover:text-green-800" @click="handleUnarchive">
          取消归档
        </button>
      </template>
    </td>
  </tr>
</template>

<script setup>
const props = defineProps({
  user: {
    type: Object,
    required: true
  },
  canImpersonate: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['edit', 'delete', 'recharge', 'kyc', 'referral', 'archive', 'unarchive', 'impersonate'])

function tenantCompanyLabel(user) {
  const list = user?.tenant_companies
  if (!Array.isArray(list) || list.length === 0) return '—'
  const names = list
    .map((c) => String(c?.name || c?.id || '').trim())
    .filter(Boolean)
  return names.length ? names.join('、') : '—'
}

function lastActiveLabel(user) {
  const raw = String(user?.last_login || '').trim()
  if (!raw) return '—'
  const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T') + (raw.endsWith('Z') ? '' : 'Z')
  const d = new Date(normalized)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString('zh-CN')
}

function profitSharingQualificationLabel(user) {
  const v = user?.has_profit_sharing_qualification
  if (v === true) return '是'
  if (v === false) return '否'
  return '—'
}

function roleBadgeLabel(user) {
  if (user?.is_superuser) return '超管'
  if (user?.is_staff) return '员工'
  if (user?.is_tester) return '测试'
  if (user?.is_tenant) return '租户'
  return '用户'
}

function roleBadgeClass(user) {
  if (user?.is_superuser) return 'bg-purple-100 text-purple-800'
  if (user?.is_staff) return 'bg-blue-100 text-blue-800'
  if (user?.is_tester) return 'bg-amber-100 text-amber-800'
  if (user?.is_tenant) return 'bg-green-100 text-green-800'
  return 'bg-gray-100 text-gray-800'
}

const handleImpersonate = () => {
  emit('impersonate', props.user)
}

const handleEdit = () => {
  emit('edit', props.user)
}

const handleKyc = () => {
  emit('kyc', props.user)
}

const handleReferral = () => {
  emit('referral', props.user)
}

const handleRecharge = () => {
  emit('recharge', props.user)
}

const handleDelete = () => {
  emit('delete', props.user)
}

const handleArchive = () => {
  emit('archive', props.user)
}

const handleUnarchive = () => {
  emit('unarchive', props.user)
}
</script>
<style scoped>
/* 组件内样式可以在这里添加 */
</style>
