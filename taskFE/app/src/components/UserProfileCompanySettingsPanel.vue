<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6">
    <h2 class="text-lg font-semibold text-text mb-4">在各公司的设置</h2>
    <div v-if="companyNicknames.length === 0" class="text-sm text-text-light">
      你当前还没有加入任何公司。
    </div>
    <div v-else class="space-y-6">
      <div
        v-for="item in companyNicknames"
        :key="item.company_id"
        class="border border-border rounded-lg p-4 space-y-4"
      >
        <p class="text-sm font-medium text-text">
          公司：{{ item.company_name || '未设置' }}
        </p>

        <div class="space-y-1">
          <button
            type="button"
            :disabled="isCompanyAvatarBusy(item.company_id) || isSaving"
            class="px-3 py-1.5 text-sm rounded-lg border border-primary/50 text-primary hover:bg-primary/5 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
            @click="copyPersonalToCompany(item.company_id)"
          >
            从个人昵称与头像复制
          </button>
          <p class="text-xs text-text-light max-w-xl">
            昵称将使用上方「个人昵称」输入框中的文字；头像将使用<strong class="font-medium text-text">已保存</strong>的全局头像。若刚修改过全局头像或昵称，请先点击页面底部「保存」再复制。
          </p>
        </div>

        <div class="space-y-2">
          <h3 class="text-sm font-medium text-text">昵称设置</h3>
          <input
            :value="item.member_name"
            type="text"
            maxlength="255"
            placeholder="在本公司显示的名称"
            class="w-full max-w-md px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none transition-colors"
            @input="onMemberNameInput(item.company_id, $event)"
          >
        </div>

        <div class="space-y-2">
          <h3 class="text-sm font-medium text-text">头像设置</h3>
          <p class="text-xs text-text-light">支持 JPEG、PNG、WebP、GIF，最大 2MB；仅在本公司相关展示中使用。</p>
          <div class="flex flex-wrap items-center gap-3">
            <img
              :src="companyMemberAvatarDisplayUrl(item)"
              alt=""
              class="w-16 h-16 rounded-full object-cover border border-border shrink-0"
              width="64"
              height="64"
            >
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                :disabled="isCompanyAvatarBusy(item.company_id)"
                class="px-3 py-1.5 text-sm rounded-lg border border-border text-text hover:bg-gray-50 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                @click="openCompanyAvatarPicker(item.company_id)"
              >
                {{ isCompanyAvatarBusy(item.company_id) ? '处理中...' : '上传图片' }}
              </button>
              <button
                v-if="item.member_avatar_url"
                type="button"
                :disabled="isCompanyAvatarBusy(item.company_id)"
                class="px-3 py-1.5 text-sm rounded-lg border border-red-300 text-danger hover:bg-red-50 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                @click="removeCompanyMemberAvatar(item.company_id)"
              >
                移除头像
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
    <input
      ref="companyAvatarFileInput"
      type="file"
      class="hidden"
      accept="image/jpeg,image/png,image/webp,image/gif"
      @change="onCompanyMemberAvatarFileChange"
    >
  </div>
</template>

<script setup>
import { ref, useTemplateRef } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { initialsAvatarDataUri } from '../utils/initialsAvatarDataUri.js'

const props = defineProps({
  companyNicknames: { type: Array, default: () => [] },
  personalNickname: { type: String, default: '' },
  isSaving: { type: Boolean, default: false },
})

const emit = defineEmits(['profile-updated', 'message', 'error', 'update:companyNicknames'])

const companyAvatarFileInput = useTemplateRef('companyAvatarFileInput')
const companyAvatarPickCompanyId = ref('')
const companyAvatarBusy = ref({})
// OPT-20260819-038: 公司头像 上传/移除/复制个人资料 均为写操作，防连点双发
const uploadAvatarGuard = createClickGuard()
const removeAvatarGuard = createClickGuard()
const copyPersonalGuard = createClickGuard()

const isCompanyAvatarBusy = (companyId) => Boolean(companyAvatarBusy.value[String(companyId)])

const setCompanyAvatarBusy = (companyId, busy) => {
  const id = String(companyId)
  const next = { ...companyAvatarBusy.value }
  if (busy) next[id] = true
  else delete next[id]
  companyAvatarBusy.value = next
}

const companyMemberAvatarDisplayUrl = (item) => {
  if (item.member_avatar_url) return item.member_avatar_url
  return initialsAvatarDataUri(item.member_name || item.company_name || 'member')
}

function onMemberNameInput(companyId, event) {
  const value = event?.target?.value ?? ''
  const next = props.companyNicknames.map((item) =>
    String(item.company_id) === String(companyId)
      ? { ...item, member_name: value }
      : item,
  )
  emit('update:companyNicknames', next)
}

const openCompanyAvatarPicker = (companyId) => {
  emit('error', '')
  emit('message', '')
  companyAvatarPickCompanyId.value = String(companyId)
  companyAvatarFileInput.value?.click()
}

const onCompanyMemberAvatarFileChange = async (event) => {
  const input = event.target
  const file = input.files && input.files[0]
  const cid = companyAvatarPickCompanyId.value
  input.value = ''
  companyAvatarPickCompanyId.value = ''
  if (!file || !cid) return
  // OPT-20260819-038: 上传公司头像是写操作，防连点双发
  await uploadAvatarGuard.run(async ({ idempotencyKey }) => {
    setCompanyAvatarBusy(cid, true)
    emit('error', '')
    emit('message', '')
    try {
      const body = new FormData()
      body.append('company_id', cid)
      body.append('avatar', file)
      const response = await apiFetch('/api/accounts/users/profile/company-avatar/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
        body,
      })
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || '上传失败')
      emit('profile-updated', data)
      emit('message', '该公司内的头像已更新')
    } catch (error) {
      console.error('上传公司头像失败:', error)
      emit('error', error.message || '上传失败')
    } finally {
      setCompanyAvatarBusy(cid, false)
    }
  })
}

const removeCompanyMemberAvatar = async (companyId) => {
  const cid = String(companyId)
  // OPT-20260819-038: 移除公司头像是写操作，防连点双发 DELETE
  await removeAvatarGuard.run(async ({ idempotencyKey }) => {
    setCompanyAvatarBusy(cid, true)
    emit('error', '')
    emit('message', '')
    try {
      const response = await apiFetch(
        `/api/accounts/users/profile/company-avatar/?company_id=${encodeURIComponent(cid)}`,
        {
          method: 'DELETE',
          headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
        },
      )
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || '移除失败')
      emit('profile-updated', data)
      emit('message', '已移除该公司的头像')
    } catch (error) {
      console.error('移除公司头像失败:', error)
      emit('error', error.message || '移除失败')
    } finally {
      setCompanyAvatarBusy(cid, false)
    }
  })
}

const copyPersonalToCompany = async (companyId) => {
  const cid = String(companyId)
  // OPT-20260819-038: 复制个人资料到公司是写操作，防连点双发 POST
  await copyPersonalGuard.run(async ({ idempotencyKey }) => {
    setCompanyAvatarBusy(cid, true)
    emit('error', '')
    emit('message', '')
    try {
      const response = await apiFetch('/api/accounts/users/profile/copy-personal-to-company/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({
          company_id: cid,
          personal_nickname: props.personalNickname,
        }),
      })
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || '复制失败')
      emit('profile-updated', data)
      emit('message', '已将该公司的昵称与头像与当前个人资料对齐（昵称取自输入框，头像取自已保存的全局头像）')
    } catch (error) {
      console.error('复制到公司失败:', error)
      emit('error', error.message || '复制失败')
    } finally {
      setCompanyAvatarBusy(cid, false)
    }
  })
}
</script>
