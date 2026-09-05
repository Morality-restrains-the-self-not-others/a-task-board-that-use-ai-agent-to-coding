import { createApp } from "vue";
import App from "./App.vue";
import router from "./router";

import "./style.css";

const app = createApp(App);
app.use(router);
// 等初始导航（含直接打开 /admin）完成后再挂载，避免首屏落在错误路由
router.isReady().then(() => {
  app.mount("#app");
});
