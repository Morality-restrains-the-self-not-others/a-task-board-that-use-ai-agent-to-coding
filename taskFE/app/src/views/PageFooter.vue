<template>
  <footer class="bg-gray-800 text-white py-12">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
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
      <div class="mt-12 pt-8 border-t border-gray-700 text-center text-gray-400">
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
        <p class="mt-2">
          <a
            :href="agplSourceUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-gray-400 hover:text-white underline underline-offset-2"
            data-testid="agpl-source-link"
            title="本服务源码（AGPL-3.0 第 13 条）"
          >{{ agplSourceLabel }}</a>
        </p>
      </div>
    </div>
  </footer>
</template>

<script setup>
import { AGPL_SOURCE_URL, AGPL_SOURCE_LABEL } from '../utils/agplSourceInfo.js'

const agplSourceUrl = AGPL_SOURCE_URL
const agplSourceLabel = AGPL_SOURCE_LABEL
// SSOT: conf/frontend/vue/config.yaml icpBeian → Vite VITE_ICP_BEIAN
const icpBeian = String(import.meta.env.VITE_ICP_BEIAN || '').trim()
</script>
