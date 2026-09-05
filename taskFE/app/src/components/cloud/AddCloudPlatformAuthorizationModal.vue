<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">添加云平台授权</h3>
        <button
          type="button"
          class="text-gray-500 hover:text-gray-700 transition-colors"
          @click="emit('close')"
        >
          &times;
        </button>
      </div>

      <form @submit.prevent="emit('submit')">
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">云平台类型</label>
            <select
              v-model="form.platform_type"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
              @change="emit('platform-type-change')"
            >
              <option value="">请选择云平台类型</option>
              <option
                v-for="platform in cloudPlatforms"
                :key="platform.value"
                :value="platform.value"
                :disabled="platform.value !== 'aliyun'"
              >
                {{ platform.label }}{{ platform.value !== 'aliyun' ? ' (开发中)' : '' }}
              </option>
            </select>
          </div>

          <div class="h-16 flex flex-col">
            <label class="block text-sm font-medium text-gray-700 mb-1">授权方式</label>
            <div class="grid grid-cols-2 gap-2 flex-grow">
              <div
                v-for="authType in availableAuthTypes"
                :key="authType.value"
                :class="[
                  'rounded-lg border-2 p-2 transition-all duration-200 h-full flex items-center',
                  authType.disabled
                    ? 'cursor-not-allowed border-gray-200 bg-gray-50'
                    : 'cursor-pointer',
                  !authType.disabled && form.authorization_type === authType.value
                    ? 'border-primary bg-primary/5'
                    : !authType.disabled && 'border-gray-200 hover:border-gray-300 hover:bg-gray-50',
                ]"
                @click="!authType.disabled && (form.authorization_type = authType.value)"
              >
                <div class="flex items-center space-x-2 w-full">
                  <div
                    :class="[
                      'w-4 h-4 rounded-full border-2 flex items-center justify-center flex-shrink-0',
                      !authType.disabled && form.authorization_type === authType.value
                        ? 'border-primary bg-primary'
                        : 'border-gray-300',
                    ]"
                  >
                    <div
                      v-if="!authType.disabled && form.authorization_type === authType.value"
                      class="w-2 h-2 bg-white rounded-full"
                    ></div>
                  </div>
                  <div class="flex flex-col justify-center flex-grow">
                    <p :class="['font-medium text-sm', authType.disabled ? 'text-gray-400' : 'text-gray-900']">
                      {{ authType.label }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="form.authorization_type === 'access_key'">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">Access Key</label>
              <input
                v-model="form.access_key"
                type="text"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="Access Key"
                required
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">Secret Key</label>
              <input
                v-model="form.secret_key"
                type="password"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="Secret Key"
                required
              />
            </div>
          </div>

          <div
            v-if="form.authorization_type === 'oauth' && form.platform_type === 'aliyun'"
            class="pt-4"
          >
            <p class="text-sm text-gray-600 mb-4">选择OAuth授权方式将跳转到阿里云官方页面进行授权</p>
            <button
              type="button"
              class="w-full px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all duration-300"
              @click="emit('oauth-login')"
            >
              跳转到阿里云授权
            </button>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">备注</label>
            <input
              v-model="form.remark"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="备注"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">是否启用</label>
            <div class="flex items-center gap-2">
              <input
                id="add-cloud-auth-is-active"
                v-model="form.is_active"
                type="checkbox"
                class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded"
              />
              <label for="add-cloud-auth-is-active" class="text-sm text-gray-700">启用</label>
            </div>
          </div>
        </div>

        <div v-if="form.authorization_type" class="flex justify-end space-x-3 mt-6">
          <button
            type="button"
            class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300"
            @click="emit('close')"
          >
            取消
          </button>
          <button
            type="submit"
            :disabled="addingAuthorization"
            class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300"
          >
            {{ addingAuthorization ? '添加中...' : '添加授权' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
/**
 * 添加云平台授权弹窗（从 WorkspaceSettingsCloudPlatform 拆分以降低单文件行数）。
 */
defineProps({
  visible: { type: Boolean, default: false },
  form: { type: Object, required: true },
  cloudPlatforms: { type: Array, required: true },
  availableAuthTypes: { type: Array, required: true },
  addingAuthorization: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'submit', 'oauth-login', 'platform-type-change'])
</script>
