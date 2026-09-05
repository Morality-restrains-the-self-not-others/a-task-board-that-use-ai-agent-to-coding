import { createRouter, createWebHistory } from "vue-router";
import VendorPortal from "../views/VendorPortal.vue";
import AdminPortal from "../views/AdminPortal.vue";
import PublicCatalog from "../views/PublicCatalog.vue";
import SaasMachineContainerSkill from "../views/SaasMachineContainerSkill.vue";

// 与 Vite base 一致；避免生产构建 base 与 history 不同步
const base = import.meta.env.BASE_URL || "/";

const router = createRouter({
  history: createWebHistory(base),
  // 静态子路径必须排在「/」之前，否则部分环境下 / 会先匹配导致 /admin 落到首页
  routes: [
    { path: "/saas-machine-container", name: "saas-machine-container-skill", component: SaasMachineContainerSkill },
    { path: "/admin", name: "admin", component: AdminPortal },
    { path: "/catalog", name: "catalog", component: PublicCatalog },
    { path: "/", name: "vendor", component: VendorPortal },
  ],
  scrollBehavior() {
    return { top: 0 };
  },
});

// 不在此拦截 /admin：未登录时由 AdminPortal 展示登录表单（见 views/AdminPortal.vue）
export default router;