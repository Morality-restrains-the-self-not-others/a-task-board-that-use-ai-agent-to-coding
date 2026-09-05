<script setup>
import { useRoute } from "vue-router";
import { IMAGE_DEMO_HREF, IMAGE_DEMO_LABEL } from "./lib/imageDemoLink.js";
import { SAAS_MACHINE_CONTAINER_SKILL_LABEL } from "./lib/saasMachineContainerSkill.js";

const route = useRoute();
</script>

<template>
  <div class="shell">
    <header class="top">
      <div class="brand">SaaS AI Provider · 镜像市场</div>
      <nav class="nav">
        <router-link :to="{ name: 'vendor' }">厂商门户</router-link>
        <router-link :to="{ name: 'admin' }">平台审核</router-link>
        <router-link :to="{ name: 'catalog' }">公开目录</router-link>
        <router-link
          :to="{ name: 'saas-machine-container-skill' }"
          data-testid="nav-saas-machine-container-skill"
        >{{ SAAS_MACHINE_CONTAINER_SKILL_LABEL }}</router-link>
        <a
          :href="IMAGE_DEMO_HREF"
          target="_blank"
          rel="noopener noreferrer"
          data-testid="nav-image-demo"
        >{{ IMAGE_DEMO_LABEL }}</a>
      </nav>
    </header>
    <main class="main">
      <!-- 强制随路由切换重渲染，避免偶发视图不更新 -->
      <router-view :key="route.fullPath" />
    </main>
  </div>
</template>

<style scoped>
.shell {
  max-width: 1100px;
  margin: 0 auto;
  padding: 1.5rem 1.25rem 3rem;
}
.top {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border);
}
.brand {
  font-weight: 700;
  letter-spacing: 0.02em;
  font-size: 1.05rem;
}
.nav {
  display: flex;
  gap: 1.25rem;
  font-size: 0.9rem;
}
.nav a {
  color: var(--muted);
}
/* 仅用 exact-active：否则 path「/」会前缀匹配所有路由，/admin 时仍像留在「门户」 */
.nav a.router-link-exact-active {
  color: var(--text);
  font-weight: 600;
  text-decoration: none;
  border-bottom: 2px solid var(--accent);
  padding-bottom: 2px;
}
.main {
  min-height: 60vh;
}
</style>
