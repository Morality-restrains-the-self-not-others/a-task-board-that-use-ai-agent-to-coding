<template>
  <div
    data-alias="view-faq-page"
    data-testid="view-faq-page"
    class="min-h-screen bg-gray-50 font-sans pt-24 pb-16 px-4 sm:px-6 lg:px-8"
  >
    <div class="max-w-5xl mx-auto">
      <h1 class="text-3xl font-bold text-gray-900 mb-2">常见问题</h1>
      <p class="text-gray-600 mb-8">
        常见问题与使用说明。
      </p>

      <div v-if="docs.length" class="space-y-8" data-testid="faq-doc-content">
        <div
          v-for="doc in docs"
          :key="doc.id"
          class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 sm:p-8"
          :data-testid="`faq-doc-panel-${doc.id}`"
        >
          <MarkdownContent :content="doc.content" empty-text="暂无内容" />
        </div>
        <p class="text-sm text-gray-500">
          <!-- Anti-Replay-OK: real router-link navigation, no write side-effect -->
          <router-link
            :to="{ name: 'home' }"
            class="text-primary hover:underline"
            data-testid="faq-back-home-link"
          >
            返回首页
          </router-link>
        </p>
      </div>

      <p v-else class="text-gray-500" data-testid="faq-empty">暂无常见问题文档</p>
    </div>
  </div>
</template>

<script setup>
import MarkdownContent from '../components/MarkdownContent.vue'
import { applyFaqPlaceholders } from '../faq/applyFaqPlaceholders.js'

/* @alias:view-faq-page */
// Faq.vue 常见问题页：展示 src/faq/*.md 文档（import.meta.glob ?raw 构建期打包，
// 新增 md 文件无需改代码自动被发现）。OPT-20260824-083 页脚 FAQ 链接落地页。
// 无侧边栏：全部文档纵向铺开；知识产权处理方法、账号与登录已下线。
// md 不放 src/public/：否则 Vite 拷到 /faq/ 静态目录，nginx try_files 命中目录会 403。

// key 形如 /src/faq/usage.md（相对本文件为 ../faq/*.md）
const FAQ_RAW_DOCS = import.meta.glob('../faq/*.md', {
  query: '?raw',
  import: 'default',
  eager: true,
})

const contactEmail = import.meta.env.VITE_CONTACT_EMAIL

const docs = Object.entries(FAQ_RAW_DOCS)
  .map(([key, content]) => {
    const id = key.split('/').pop().replace(/\.md$/, '')
    return { id, content: applyFaqPlaceholders(content, contactEmail) }
  })
  // 码点排序（非 locale）：英文字母文档在前，中文名文档在后，顺序确定可预期
  .sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
</script>
