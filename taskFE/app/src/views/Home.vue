<template>
  <div data-alias="view-home-page" class="min-h-screen bg-gray-50 font-sans">
    <!-- 英雄区域 -->
    <section class="bg-gradient-to-r from-primary to-primary-dark text-white py-24 pt-32">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-12 items-center">
          <div class="flex flex-col justify-center space-y-6">
            <h1 class="text-[clamp(2.5rem,5vw,4rem)] font-bold leading-tight">高效管理您的SaaS项目</h1>
            <p class="text-xl text-white/90 max-w-lg">支持用户并发执行任务。</p>
            <div class="flex flex-wrap gap-4">
              <a href="/auth/register/" class="btn-primary bg-white text-primary hover:bg-gray-100 px-8 py-3 text-base font-semibold">立即开始</a>
              <a href="#projects" class="btn-secondary bg-transparent border border-white text-white hover:bg-white/10 px-8 py-3 text-base font-semibold backdrop-blur-sm">了解更多</a>
            </div>
            <div class="flex items-center space-x-6 pt-4">
              <div class="flex -space-x-3">
                <img class="w-10 h-10 rounded-full border-2 border-white" src="https://picsum.photos/id/1005/40/40" alt="用户头像">
                <img class="w-10 h-10 rounded-full border-2 border-white" src="https://picsum.photos/id/1012/40/40" alt="用户头像">
                <img class="w-10 h-10 rounded-full border-2 border-white" src="https://picsum.photos/id/1027/40/40" alt="用户头像">
              </div>
            </div>
          </div>
          <div class="hidden md:flex justify-center items-center">
            <div class="bg-white/10 backdrop-blur-lg rounded-2xl p-8 shadow-2xl border border-white/20">
              <div class="grid grid-cols-2 gap-6">
                <div class="bg-white/20 rounded-xl p-5 transition-transform hover:scale-105 duration-300">
                  <div class="text-4xl font-bold">29+</div>
                  <div class="text-sm text-white/80 mt-1">测试用例</div>
                </div>
                <div class="bg-white/20 rounded-xl p-5 transition-transform hover:scale-105 duration-300">
                  <div class="text-4xl font-bold">4+</div>
                  <div class="text-sm text-white/80 mt-1">核心模块</div>
                </div>
                <div class="bg-white/20 rounded-xl p-5 transition-transform hover:scale-105 duration-300">
                  <div class="text-4xl font-bold">100%</div>
                  <div class="text-sm text-white/80 mt-1">测试通过率</div>
                </div>
                <div class="bg-white/20 rounded-xl p-5 transition-transform hover:scale-105 duration-300">
                  <div class="text-4xl font-bold">0</div>
                  <div class="text-sm text-white/80 mt-1">安全漏洞</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 功能特性 -->
    <section id="projects" class="py-24 bg-white">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-16">
          <div class="inline-block px-4 py-1 rounded-full bg-primary/10 text-primary font-medium text-sm mb-4">核心功能</div>
          <h2 class="text-[clamp(2rem,4vw,3rem)] font-bold text-gray-800 mb-4">强大的SaaS项目管理工具</h2>
          <p class="text-xl text-gray-600 max-w-3xl mx-auto">我们提供一系列专业功能，帮助您的团队更高效地协作和交付项目</p>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-10">
          <HomeFeatureCard
            v-for="card in featureCards"
            :key="card.title"
            :icon="card.icon"
            :title="card.title"
            :description="card.description"
            :href="card.href"
          />
        </div>
      </div>
    </section>

    <!-- 页脚 -->
    <footer class="bg-primary-dark text-white py-16">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-12">
          <div>
            <div data-testid="home-footer-brand" class="text-2xl font-bold mb-6 text-white">云端Coding 平台</div>
            <p class="text-primary-light/80 max-w-xs">强大的SaaS项目管理平台，帮助您更好地管理SaaS项目和团队协作</p>
          </div>
          <div>
            <h4 class="text-lg font-semibold mb-6 text-white">快速链接</h4>
            <ul class="space-y-3">
              <li><a href="/" data-testid="home-footer-home-link" class="text-primary-light/80 hover:text-white transition-colors duration-300">首页</a></li>
              <li>
                <router-link
                  :to="{ name: 'acknowledgments' }"
                  data-testid="home-footer-acknowledgments-link"
                  class="text-primary-light/80 hover:text-white transition-colors duration-300"
                >
                  致谢
                </router-link>
              </li>
            </ul>
          </div>
          <div>
            <h4 id="footer-contact" class="text-lg font-semibold mb-6 text-white">联系我们</h4>
            <ul class="space-y-3 text-primary-light/80">
              <li class="flex items-center">
                <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                </svg>
                <span data-testid="home-footer-contact-email">{{ contactEmail }}</span>
              </li>
              <li>
                <router-link
                  :to="{ name: 'faq' }"
                  data-testid="home-footer-faq-link"
                  class="flex items-center hover:text-white transition-colors duration-300"
                >
                  <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                  </svg>
                  常见问题
                </router-link>
              </li>
            </ul>
          </div>
        </div>
        <div class="border-t border-primary-light/20 mt-16 pt-8 text-center text-primary-light/80">
          <p data-testid="home-footer-copyright">&copy; 2023 SaaS Platform. All rights reserved.</p>
          <p v-if="icpBeian" class="mt-2">
            <a
              href="https://beian.miit.gov.cn/"
              target="_blank"
              rel="noopener noreferrer"
              class="text-primary-light/80 hover:text-white transition-colors duration-300"
              data-testid="home-footer-icp-link"
            >{{ icpBeian }}</a>
            <!-- Anti-Replay-OK: real external <a href> to MIIT beian portal; no write side-effect -->
          </p>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup>
/* @alias:view-home-page */
// Home.vue 首页视图组件
import HomeFeatureCard from '../components/HomeFeatureCard.vue'

// SSOT: conf/frontend/vue/config.yaml contactEmail → Vite VITE_CONTACT_EMAIL（与 FAQ 同源）
const contactEmail = import.meta.env.VITE_CONTACT_EMAIL
// SSOT: conf/frontend/vue/config.yaml icpBeian → Vite VITE_ICP_BEIAN
const icpBeian = String(import.meta.env.VITE_ICP_BEIAN || '').trim()

// 「核心功能」三张特性卡数据（OPT-20260811-071 抽为子组件；icon 为 SVG path d）
const featureCards = [
  {
    icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2',
    title: '项目管理',
    description: '创建和管理项目，分配任务，跟踪进度，确保项目按时交付。支持团队协作和实时沟通。',
    href: '/auth/register/',
  },
  {
    icon: 'M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z',
    title: '云平台管理',
    description: '管理已安装镜像，授权AI模型，监控云资源使用情况。提供灵活的部署和扩展方案。',
    href: '/auth/register/',
  },
  {
    icon: 'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z',
    title: '用户和权限管理',
    description: '管理用户账户，设置角色和权限，确保数据安全。支持多级权限和单点登录。',
    href: '/auth/register/',
  },
]
</script>

<style scoped>
/* 组件内样式 */
</style>
