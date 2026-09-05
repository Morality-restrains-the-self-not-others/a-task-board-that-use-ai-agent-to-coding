<template>
  <footer class="bg-gray-800 text-white py-10 sm:py-12">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <!-- 移动端折叠信息，减少首屏后的滚动负担 -->
      <div v-if="isMobileViewport" class="space-y-3">
        <div>
          <h3 class="text-2xl font-bold mb-3">SaaS平台</h3>
          <p class="text-gray-400">提供全方位的项目管理解决方案，帮助您的团队更高效地协作和交付。</p>
        </div>

        <details class="group border border-gray-700 rounded-lg px-4 py-3">
          <summary class="list-none flex items-center justify-between cursor-pointer font-semibold">
            产品
            <span class="text-gray-400 group-open:rotate-180 transition-transform duration-200">⌄</span>
          </summary>
          <ul class="mt-3 space-y-2 text-gray-400">
            <li><a href="#" class="hover:text-white">项目管理</a></li>
            <li><a href="#" class="hover:text-white">云平台</a></li>
            <li><a href="#" class="hover:text-white">API文档</a></li>
            <li><a href="#" class="hover:text-white">定价</a></li>
          </ul>
        </details>

        <details class="group border border-gray-700 rounded-lg px-4 py-3">
          <summary class="list-none flex items-center justify-between cursor-pointer font-semibold">
            资源
            <span class="text-gray-400 group-open:rotate-180 transition-transform duration-200">⌄</span>
          </summary>
          <ul class="mt-3 space-y-2 text-gray-400">
            <li><a href="#" class="hover:text-white">文档中心</a></li>
            <li><a href="#" class="hover:text-white">教程</a></li>
            <li><a href="#" class="hover:text-white">支持</a></li>
            <li><a href="#" class="hover:text-white">博客</a></li>
          </ul>
        </details>

        <details class="group border border-gray-700 rounded-lg px-4 py-3">
          <summary class="list-none flex items-center justify-between cursor-pointer font-semibold">
            联系我们
            <span class="text-gray-400 group-open:rotate-180 transition-transform duration-200">⌄</span>
          </summary>
          <ul class="mt-3 space-y-2 text-gray-400">
            <li>邮箱：contact@daydaymoney.com</li>
            <li>电话：400-123-4567</li>
            <li>地址：北京市朝阳区</li>
          </ul>
        </details>
      </div>

      <!-- 桌面端保持完整四列 -->
      <div v-else class="grid grid-cols-2 lg:grid-cols-4 gap-8">
        <div>
          <h3 class="text-xl font-bold mb-4">SaaS平台</h3>
          <p class="text-gray-400">提供全方位的项目管理解决方案，帮助您的团队更高效地协作和交付。</p>
        </div>
        <div>
          <h3 class="text-lg font-bold mb-4">产品</h3>
          <ul class="space-y-2">
            <li><a href="#" class="text-gray-400 hover:text-white">项目管理</a></li>
            <li><a href="#" class="text-gray-400 hover:text-white">云平台</a></li>
            <li><a href="#" class="text-gray-400 hover:text-white">API文档</a></li>
            <li><a href="#" class="text-gray-400 hover:text-white">定价</a></li>
          </ul>
        </div>
        <div>
          <h3 class="text-lg font-bold mb-4">资源</h3>
          <ul class="space-y-2">
            <li><a href="#" class="text-gray-400 hover:text-white">文档中心</a></li>
            <li><a href="#" class="text-gray-400 hover:text-white">教程</a></li>
            <li><a href="#" class="text-gray-400 hover:text-white">支持</a></li>
            <li><a href="#" class="text-gray-400 hover:text-white">博客</a></li>
          </ul>
        </div>
        <div>
          <h3 class="text-lg font-bold mb-4">联系我们</h3>
          <ul class="space-y-2">
            <li class="text-gray-400">邮箱：contact@daydaymoney.com</li>
            <li class="text-gray-400">电话：400-123-4567</li>
            <li class="text-gray-400">地址：北京市朝阳区</li>
          </ul>
        </div>
      </div>
      <div class="mt-8 sm:mt-12 pt-6 sm:pt-8 border-t border-gray-700 text-center text-gray-400">
        <p>&copy; 2025 SaaS平台. 保留所有权利.</p>
        <p v-if="icpBeian" class="mt-2">
          <a
            href="https://beian.miit.gov.cn/"
            target="_blank"
            rel="noopener noreferrer"
            class="text-gray-400 hover:text-white"
            data-testid="icp-link"
          >{{ icpBeian }}</a>
        </p>
      </div>
    </div>
  </footer>
</template>

<script setup>
defineProps({
  isMobileViewport: { type: Boolean, default: false },
})

// SSOT: conf/frontend/vue/config.yaml icpBeian → Vite VITE_ICP_BEIAN
const icpBeian = String(import.meta.env.VITE_ICP_BEIAN || '').trim()
</script>
