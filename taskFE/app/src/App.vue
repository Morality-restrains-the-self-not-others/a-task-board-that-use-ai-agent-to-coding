<template>
  <div class="h-screen min-h-0 overflow-hidden bg-gray-50 font-sans">
    <!-- 整页 flex 布局：侧栏占满整屏高度，顶部导航栏与内容区位于侧栏右侧。
         收起/展开顶部导航栏时侧栏位置固定不动，「控制台导航」切换入口始终停留在同一位置。
         主内容区 min-h-0 + overflow-y-auto：长页面可滚；WorkPanel 用 h-full/overflow-hidden 贴底。 -->
    <div class="flex h-full min-h-0 min-w-0">
      <!-- 侧栏（租户控制台 / 系统管理；无侧栏页面渲染空位，右侧列自动占满） -->
      <router-view name="Sidebar"></router-view>
      <!-- 右侧列：顶部导航栏 + 页面内容；主区 min-w-0 避免挤压侧栏宽度，与侧栏 shrink-0 成对使用 -->
      <div class="flex-1 min-w-0 min-h-0 flex flex-col">
        <router-view name="Navbar"></router-view>
        <div v-if="isAuthPage" class="flex-1 min-h-0 overflow-y-auto">
          <!-- 登录/注册页面：不需要 flex 和侧边栏 -->
          <router-view class="w-full"></router-view>
        </div>
        <div v-else class="flex-1 min-w-0 min-h-0 overflow-y-auto flex flex-col">
          <router-view class="flex-1 min-h-0 min-w-0 w-full"></router-view>
        </div>
      </div>
    </div>

    <!-- 全局模态弹窗 -->
    <ModalUi 
      :show="modalService.state.show" 
      :title="modalService.state.title"
      :message="modalService.state.message"
      :type="modalService.state.type"
      :confirm-text="modalService.state.confirmText"
      :cancel-text="modalService.state.cancelText"
      :additional-actions="modalService.state.additionalActions"
      :trace-id="modalService.state.traceId"
      :show-close-button="modalService.state.showCloseButton !== false"
      @close="() => modalService.close()"
      @confirm="() => modalService.handleConfirm()"
      @cancel="() => modalService.handleCancel()"
    />
    
    <PrivacyReconsentGate />
    <PhoneVerifyAccessGate />

    <!-- 全局 Toast 通知；请求错误时根节点带 data-traceId -->
    <div 
      v-if="toastService.state.show"
      class="fixed top-4 right-4 z-50 max-w-sm rounded-lg shadow-lg transition-all duration-300 ease-in-out"
      v-bind="toastService.state.traceId ? { 'data-traceId': toastService.state.traceId } : {}"
      :class="{
        'bg-green-50 border border-green-200 text-green-800': toastService.state.type === 'success',
        'bg-red-50 border border-red-200 text-red-800': toastService.state.type === 'error',
        'bg-yellow-50 border border-yellow-200 text-yellow-800': toastService.state.type === 'warning',
        'bg-blue-50 border border-blue-200 text-blue-800': toastService.state.type === 'info'
      }"
    >
      <div class="p-4">
        <div class="flex items-start">
          <div class="flex-shrink-0 mt-0.5">
            <svg v-if="toastService.state.type === 'success'" class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
            </svg>
            <svg v-else-if="toastService.state.type === 'error'" class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"></path>
            </svg>
            <svg v-else-if="toastService.state.type === 'warning'" class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
            </svg>
            <svg v-else class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"></path>
            </svg>
          </div>
          <div class="ml-3 flex-1">
            <p
              class="text-sm font-medium"
              :data-traceId="toastService.state.traceId || undefined"
            >{{ toastService.state.message }}</p>
          </div>
          <div class="ml-4 flex-shrink-0">
            <button 
              @click="toastService.hide()"
              class="text-gray-400 hover:text-gray-600 focus:outline-none"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ModalUi from './components/Modal.ui.vue'
import PrivacyReconsentGate from './components/PrivacyReconsentGate.vue'
import PhoneVerifyAccessGate from './components/PhoneVerifyAccessGate.vue'
import modalService from './utils/modalService.js'
import toastService from './utils/toastService.js'

const route = useRoute()

// 移除全局挂载，改为通过 import 导入使用

const isAuthPage = computed(() => {
  return route.name === 'login' || route.name === 'auth_login' || route.name === 'register' || route.name === 'auth_register' || route.name === 'people_join'
})
</script>

<style scoped>
/* 组件内样式 */
</style>
