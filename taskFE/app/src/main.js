import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './css/tailwind.css'
import './css/styles.css'
import modalService from './utils/modalService'
import toastService from './utils/toastService'
import './utils/cookieUtils'
import { config, getApiUrl, applyRuntimeApiBaseFromWindow, applyRuntimeGatewayBaseFromWindow } from './utils/config.js'
import * as apiUtils from './utils/apiUtils.js'
import { initReferralCode } from './utils/referralUtils.js'
import { normalizeUrlAccessCodeToOwn } from './utils/referralAccessCodeUtils.js'
import { getStoredUserId } from './utils/sessionUserIdUtils.js'
import { setupOAuthCallbackToastGuard } from './utils/gitSiteOAuthCallbackUtils.js'
import { installRouterChunkGuard } from './utils/chunkLoadGuard.js'
import { handleRgDeepLink, initRgDeepLink } from './utils/rgDeepLink.js'

// 定义工具函数（getCookie 已在 cookieUtils.js 中定义并自动挂载）
// 保留已有的 window.utils 对象，不要覆盖它

// 移除全局挂载，改为通过 import 导入使用

// 密码已改为服务端 bcrypt 存储，前端不再做密码哈希
// PasswordHasher.js 已保留供向后兼容（已不再导入使用）

const app = createApp(App)
app.use(router)
setupOAuthCallbackToastGuard(router, toastService)
installRouterChunkGuard(router)
// #rg= 深链定位（访问管理「打开页面/定位」）：hashchange 监听 + 路由切换后重扫
initRgDeepLink()
router.afterEach(() => handleRgDeepLink())

// OPT-20260824-005: 登录后地址栏 accessCode 归一化为自己的默认推荐码。
// 地址栏 ?accessCode= 是分享链接溯源参数（他人推荐码），登录后继续残留会让用户复制地址栏
// 分享时分账归属错误；同时同步旧机制 localStorage（referralUtils 点击处理器按它给同源链接
// 附加 accessCode，不同步会让他人码在下次点击时卷土重来）。未登录不处理，保留邀请溯源参数。
router.afterEach(() => {
  if (!getStoredUserId()) return
  normalizeUrlAccessCodeToOwn().catch(() => {})
})

applyRuntimeApiBaseFromWindow()
applyRuntimeGatewayBaseFromWindow()

// 须在 mount 之前挂载：路由首屏可能在 mount 同步阶段即调用 apiFetch，晚赋值会导致「全局配置未加载」及错误 API 前缀
window.config = config
window.getApiUrl = getApiUrl
window.apiFetch = apiUtils.apiFetch
window.api = apiUtils

app.mount('#app')

// 检查Vue应用状态
console.log('Vue app created:', app)
console.log('Vue devtools hook:', window.__VUE_DEVTOOLS_GLOBAL_HOOK__)
console.log('Vue version:', app.version)

window.vueApp = app

// 初始化推荐码处理
initReferralCode()

console.log('全局配置已设置:', config)
console.log('API 工具函数已暴露到全局')
console.log('推荐码处理已初始化')
