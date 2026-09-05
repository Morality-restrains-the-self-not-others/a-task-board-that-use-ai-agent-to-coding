import { defineConfig } from 'vite'
import { defaultExclude } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve, dirname } from 'path'
import fs from 'node:fs'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { assetCacheBustQueryPlugin } from './vite-plugin-asset-cache-bust.js'
import { buildTimeMetaPlugin } from './src/build/vite-plugin-build-time-meta.js'

const __dirname = dirname(fileURLToPath(import.meta.url))

const FB = {
  // 默认端口与 conf/frontend/vue/config.yaml 及 runall-lifecycle.sh stop 逻辑一致（4000）。
  // 此前默认 3000 与 ai-monitor/Grafana 冲突（OPT-20260806-060）：conf-read 崩溃回退
  // 默认值时 vite preview 会起 3000 导致端口冲突启动失败。
  vue: { host: 'localhost', port: 4000, allowedHost: 'http://daydaymoney.com' },
}

function hostnameFromAllowedHost(raw) {
  if (typeof raw !== 'string' || !raw.trim()) return ''
  try {
    const u = new URL(raw.trim())
    return u.hostname
  } catch {
    return ''
  }
}

function fbVueAllowedHostsList() {
  const h = hostnameFromAllowedHost(FB.vue.allowedHost)
  return h ? [h] : []
}

function normalizeAllowedHostEntry(raw) {
  if (typeof raw !== 'string') return ''
  const value = raw.trim()
  if (!value) return ''
  if (value === '*' || value.toLowerCase() === 'all') return '*'
  if (/^https?:\/\//i.test(value)) return hostnameFromAllowedHost(value)
  return value
}

function resolveVueAllowedHosts(vueCfg) {
  const baseHosts = []
  const single = normalizeAllowedHostEntry(vueCfg.allowedHost)
  if (single) baseHosts.push(single)
  const ext = Array.isArray(vueCfg.allowedExtendHosts) ? vueCfg.allowedExtendHosts : []
  const extHosts = ext.map(normalizeAllowedHostEntry).filter(Boolean)
  const entryOrigins = Array.isArray(vueCfg.publicEntryOrigins) ? vueCfg.publicEntryOrigins : []
  const entryHosts = entryOrigins.map(normalizeAllowedHostEntry).filter(Boolean)
  const merged = [...baseHosts, ...extHosts, ...entryHosts]
  if (merged.some((item) => item === '*')) return true
  return [...new Set(merged)]
}

/** 与 monorepo conf/（scripts/conf-read.py snapshot-json）一致 */
function loadMergedPortConfig() {
  try {
    const monorepoRoot = resolve(__dirname, '../../')
    const script = resolve(monorepoRoot, 'runAll/scripts/conf-read.py')
    const raw = execSync(`python3 "${script}" snapshot-json`, {
      cwd: monorepoRoot,
      encoding: 'utf-8',
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    const main = JSON.parse(raw)
    const v = main.vue || {}
    const vuePort = Number(v.port) || FB.vue.port
    const vueHost = typeof v.host === 'string' && v.host.trim() ? v.host.trim() : FB.vue.host
    const resolvedVueAllowedHosts = resolveVueAllowedHosts(v)
    const vueAllowedHosts =
      Array.isArray(resolvedVueAllowedHosts) && resolvedVueAllowedHosts.length === 0
        ? fbVueAllowedHostsList()
        : resolvedVueAllowedHosts
    const apiBaseUrl =
      typeof v.apiBaseUrl === 'string' && v.apiBaseUrl.trim() ? v.apiBaseUrl.trim().replace(/\/+$/, '') : ''
    const contactEmail = typeof v.contactEmail === 'string' ? v.contactEmail.trim() : ''
    const icpBeian = typeof v.icpBeian === 'string' ? v.icpBeian.trim() : ''
    const gitServicePublicUrl =
      typeof v.publicUrl === 'string' && v.publicUrl.trim()
        ? v.publicUrl.trim().replace(/\/+$/, '')
        : ''
    const relay = main.relayToTrae || {}
    const relayHost =
      typeof relay.host === 'string' && relay.host.trim() ? relay.host.trim() : '127.0.0.1'
    const relayPort = Number(relay.port) || 8797
    const relaySecret = typeof relay.secret === 'string' ? relay.secret : ''
    const taskSse = main.taskSSE || {}
    const taskSseHost =
      typeof taskSse.host === 'string' && taskSse.host.trim() ? taskSse.host.trim() : '127.0.0.1'
    const taskSsePort = Number(taskSse.port) || 8798
    const taskContainerGateway = main.taskContainerGateway || {}
    const taskContainerGatewayHost =
      typeof taskContainerGateway.host === 'string' && taskContainerGateway.host.trim()
        ? taskContainerGateway.host.trim()
        : '127.0.0.1'
    const taskContainerGatewayPort = Number(taskContainerGateway.port) || 8014
    // taskGateway 地址：优先 _addressing 解析值，其次 taskGateway.publicBase
    const addressing = main._addressing || {}
    const taskGatewayCfg = main.taskGateway || {}
    const gatewayOrigin = (addressing.addresses && addressing.addresses.gateway)
      || taskGatewayCfg.publicBase
      || `http://127.0.0.1:${taskGatewayCfg.httpPort || 18081}`
    const baseDomain =
      typeof addressing.base_domain === 'string' && addressing.base_domain.trim()
        ? addressing.base_domain.trim()
        : ''
    const addressingScheme =
      typeof addressing.scheme === 'string' && addressing.scheme.trim()
        ? addressing.scheme.trim()
        : 'https'
    // 公网页面/SSO 重定向使用 base 域名（${scheme}://${subdomains.base}），避免跳转到 api.* 网关域
    const basePublicOrigin = baseDomain
      ? `${addressingScheme}://${baseDomain}`
      : gatewayOrigin
    const ssoCookieDomain = baseDomain ? `.${baseDomain.replace(/^\./, '')}` : ''
    return {
      vue: {
        host: vueHost,
        port: vuePort,
        origin: `http://${vueHost}:${vuePort}`,
        allowedHosts: vueAllowedHosts,
        apiBaseUrl,
        contactEmail,
        icpBeian,
      },
      apiBaseUrl,
      contactEmail,
      icpBeian,
      gatewayOrigin,
      baseDomain,
      basePublicOrigin,
      ssoCookieDomain,
      relayToTrae: {
        host: relayHost,
        port: relayPort,
        origin: `http://${relayHost}:${relayPort}`,
        secret: relaySecret,
        businessApiOrigin:
          typeof relay.businessApiOrigin === 'string' && relay.businessApiOrigin.trim()
            ? relay.businessApiOrigin.trim().replace(/\/+$/, '')
            : 'http://127.0.0.1:8765',
      },
      taskSSE: {
        host: taskSseHost,
        port: taskSsePort,
        origin: `http://${taskSseHost}:${taskSsePort}`,
        enabled: taskSse.enabled !== false,
        transport: typeof taskSse.transport === 'string' ? taskSse.transport : 'redis',
      },
      taskContainerGateway: {
        host: taskContainerGatewayHost,
        port: taskContainerGatewayPort,
        origin: `http://${taskContainerGatewayHost}:${taskContainerGatewayPort}`,
        enabled: taskContainerGateway.enabled !== false,
      },
      gitService: {
        publicUrl: gitServicePublicUrl,
      },
    }
  } catch {
    return {
      vue: {
        ...FB.vue,
        origin: `http://${FB.vue.host}:${FB.vue.port}`,
        allowedHosts: fbVueAllowedHostsList(),
        contactEmail: '',
        icpBeian: '',
      },
      apiBaseUrl: '',
      contactEmail: '',
      icpBeian: '',
      gatewayOrigin: 'http://127.0.0.1:18081',
      baseDomain: '',
      basePublicOrigin: '',
      ssoCookieDomain: '',
      relayToTrae: {
        host: '127.0.0.1',
        port: 8797,
        origin: 'http://127.0.0.1:8797',
        secret: '',
        businessApiOrigin: 'http://127.0.0.1:8765',
      },
      taskSSE: {
        host: '127.0.0.1',
        port: 8798,
        origin: 'http://127.0.0.1:8798',
        enabled: true,
        transport: 'redis',
      },
      taskContainerGateway: {
        host: '127.0.0.1',
        port: 8014,
        origin: 'http://127.0.0.1:8014',
        enabled: true,
      },
    }
  }
}

/** 构建期 API 根：优先 VITE_API_BASE_URL；其次 vue.apiBaseUrl。
 *  空字符串 = 浏览器同源 `/api`（边缘 nginx 或下方 Vite proxy → gateway）。 */
function resolveViteApiBaseUrl(cfg, _isDev) {
  const env = typeof process.env.VITE_API_BASE_URL === 'string' ? process.env.VITE_API_BASE_URL.trim().replace(/\/+$/, '') : ''
  if (env !== '') return env
  if (typeof cfg.apiBaseUrl === 'string') {
    return cfg.apiBaseUrl.trim().replace(/\/+$/, '')
  }
  return ''
}

function resolveViteDefaultTaskApiEndpoint(cfg, _isDev) {
  return cfg.gatewayOrigin || cfg.apiBaseUrl
}

const portCfg = loadMergedPortConfig()
const vueSrv = portCfg.vue

/** 经 daydaymoney.com + SSH 隧道访问本机 Vite 时，应设 VITE_DEV_ORIGIN=http://daydaymoney.com，避免 dev 下绝对资源 URL 仍指向 localhost */
const viteDevPublicOrigin =
  (typeof process.env.VITE_DEV_ORIGIN === 'string' && process.env.VITE_DEV_ORIGIN.trim()) ||
  vueSrv.origin

function viteHmrOptions() {
  const hmr = { protocol: 'ws' }
  const host =
    typeof process.env.VITE_HMR_HOST === 'string' && process.env.VITE_HMR_HOST.trim()
      ? process.env.VITE_HMR_HOST.trim()
      : undefined
  const clientPortRaw =
    typeof process.env.VITE_HMR_CLIENT_PORT === 'string' && process.env.VITE_HMR_CLIENT_PORT.trim()
      ? process.env.VITE_HMR_CLIENT_PORT.trim()
      : ''
  const clientPort = clientPortRaw !== '' ? Number(clientPortRaw) : NaN
  if (host) Object.assign(hmr, { host })
  if (!Number.isNaN(clientPort) && clientPort > 0) Object.assign(hmr, { clientPort })
  return hmr
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const isDev = mode === 'development';
  const viteApiBaseUrl = resolveViteApiBaseUrl(portCfg, isDev)
  const viteDefaultTaskApiEndpoint = resolveViteDefaultTaskApiEndpoint(portCfg, isDev)
  // OIDC issuer / authorize 使用 base 域名（${scheme}://${subdomains.base}），
  // 避免浏览器跳转到 api.* 网关域，确保用户始终在 base 域名上浏览页面。
  const viteGatewayPublicBase =
    (typeof portCfg.basePublicOrigin === 'string' && portCfg.basePublicOrigin.trim()
      ? portCfg.basePublicOrigin.trim().replace(/\/+$/, '')
      : '') || viteApiBaseUrl
  const viteSsoCookieDomain =
    typeof portCfg.ssoCookieDomain === 'string' ? portCfg.ssoCookieDomain.trim() : ''

  return {
    define: {
      'import.meta.env.VITE_USERDATA_VERIFY_BASE_URL': JSON.stringify(
        portCfg.basePublicOrigin || ''
      ),
      // 模拟启动 TaskApiEndPointOrigin 默认值：使用 gateway origin 作为默认 API 端点
      'import.meta.env.VITE_DEFAULT_MOCK_TASK_API_ENDPOINT': JSON.stringify(
        viteDefaultTaskApiEndpoint
      ),
      'import.meta.env.VITE_ENABLE_APPLY_PATCH': JSON.stringify(process.env.VITE_ENABLE_APPLY_PATCH ?? 'false'),
      'import.meta.env.VITE_API_BASE_URL': JSON.stringify(viteApiBaseUrl),
      'import.meta.env.VITE_TASK_GATEWAY_PUBLIC_BASE': JSON.stringify(viteGatewayPublicBase),
      'import.meta.env.VITE_SSO_COOKIE_DOMAIN': JSON.stringify(viteSsoCookieDomain),
      'import.meta.env.VITE_DEFAULT_RELAY_BUSINESS_API_ORIGIN': JSON.stringify(
        portCfg.relayToTrae?.businessApiOrigin || 'http://127.0.0.1:8765'
      ),
      'import.meta.env.VITE_GIT_SERVICE_PUBLIC_URL': JSON.stringify(
        portCfg.gitService?.publicUrl || ''
      ),
      'import.meta.env.VITE_CONTACT_EMAIL': JSON.stringify(
        typeof portCfg.contactEmail === 'string' ? portCfg.contactEmail : '',
      ),
      'import.meta.env.VITE_ICP_BEIAN': JSON.stringify(
        typeof portCfg.icpBeian === 'string' ? portCfg.icpBeian : '',
      ),
    },
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
    },
    plugins: [
      vue(),
      assetCacheBustQueryPlugin({ srcDir: resolve(__dirname, 'src') }),
      // OPT-20260824-056: 产物 index.html 注入 build-time 供 E2E 检测产物/源码漂移
      buildTimeMetaPlugin(),
      {
        name: 'taskfe-health-and-spa-fallback',
        configureServer(server) {
          server.middlewares.use('/health', (req, res, next) => {
            if (req.method !== 'GET' && req.method !== 'HEAD') return next()
            res.statusCode = 200
            res.setHeader('Content-Type', 'application/json; charset=utf-8')
            res.end(JSON.stringify({ service: 'taskFE', ok: true, checks: {} }))
          })
        },
        configurePreviewServer(server) {
          const assetsDir = resolve(__dirname, 'dist')
          const indexPath = resolve(assetsDir, 'index.html')

          server.middlewares.use('/health', (req, res, next) => {
            if (req.method !== 'GET' && req.method !== 'HEAD') return next()
            let hasIndex = false
            try {
              hasIndex = fs.statSync(indexPath).isFile()
            } catch {}
            res.statusCode = hasIndex ? 200 : 503
            res.setHeader('Content-Type', 'application/json; charset=utf-8')
            res.end(
              JSON.stringify({
                service: 'taskFE',
                ok: hasIndex,
                checks: { dist_index: hasIndex },
              }),
            )
          })

          // Intercept Cache-Control for /static/assets/* — Vite preview internally
          // sets "no-cache" after our middleware runs, so we wrap setHeader to
          // force our cache header on every write.
          server.middlewares.use('/static/assets/', (req, res, next) => {
            const _set = res.setHeader.bind(res)
            res.setHeader = function (name, value) {
              if (name.toLowerCase() === 'cache-control') {
                value = 'public, max-age=31536000, immutable'
              }
              return _set.call(this, name, value)
            }
            next()
          })

          // SPA fallback: serve index.html for any path that is not a real file.
          server.middlewares.use((req, res, next) => {
            if (req.method !== 'GET' && req.method !== 'HEAD') return next()
            const urlPath = (req.url || '').split('?')[0]
            if (urlPath === '/health' || urlPath.startsWith('/api/')) return next()

            // Map URL path to disk path, stripping /static/assets/ prefix
            let rel = urlPath.slice(1)
            if (urlPath.startsWith('/static/assets/')) {
              rel = urlPath.slice('/static/assets/'.length)
            }

            if (rel) {
              const candidate = resolve(assetsDir, rel)
              try {
                if (fs.statSync(candidate).isFile()) return next()
              } catch {}
            }

            // Not a file → serve index.html
            try {
              const html = fs.readFileSync(indexPath)
              res.statusCode = 200
              res.setHeader('Content-Type', 'text/html; charset=utf-8')
              res.end(html)
            } catch {
              next()
            }
          })
        },
      },
    ],
    root: './src',
    base: isDev ? '/' : '/static/assets/',
    build: {
      outDir: '../dist',
      emptyOutDir: true,
      assetsDir: '',
      manifest: true,
      // Don't use compression in development mode
      // esbuild minify 峰值内存远低于 terser，避免本机内存紧张时 rendering chunks 阶段 OOM Kill（OPT-20260812-034）
      minify: isDev ? false : 'esbuild',
      sourcemap: isDev ? 'inline' : false,
      cssCodeSplit: false,
      rollupOptions: {
        output: {
          manualChunks: undefined
        }
      }
    },
    server: {
      allowedHosts: vueSrv.allowedHosts,
      port: vueSrv.port,
      strictPort: true,
      watch: {
        usePolling: true
      },
      host: true,
      origin: viteDevPublicOrigin,
      cors: true,
      hmr: viteHmrOptions(),
      // 同源 /api：开发直连 Vite 时转发到 task-gateway（与边缘 nginx 分流一致）。
      // OPT-20260808-022: 登录/会话接口（/api/auth/、/api/accounts/）优先分流到本地 dev
      // taskAuth（cd taskAuth && bash run.sh dev；独立端口 8005；生产实例不带 env 保持
      // .daydaymoney.com）。其余 /api 仍走网关 18081（forward-auth 用同一 sso cookie 校验，
      // 登录后全链路会话完整）。Vite proxy 按 key 注册顺序首个前缀匹配生效，会话路径须在前。
      // cookieDomainRewrite: '' — 剥离 dev taskAuth 的 Domain=localhost 属性为 host-only
      // cookie：Chrome 拒绝 localhost 域的 Set-Cookie（域匹配校验），host-only 无此限制；
      // 同源 /api 请求自动携带 → 网关 forward-auth 用同一 userId+token cookie 放行。
      proxy: {
        '/api/auth/': {
          target: 'http://127.0.0.1:8005',
          changeOrigin: true,
          secure: false,
          cookieDomainRewrite: '',
        },
        '/api/accounts/': {
          target: 'http://127.0.0.1:8005',
          changeOrigin: true,
          secure: false,
          cookieDomainRewrite: '',
        },
        '/api': {
          target: portCfg.gatewayOrigin || 'http://127.0.0.1:18081',
          changeOrigin: true,
          secure: false,
        },
      },
    },
    preview: {
      port: vueSrv.port,
      host: true,
      strictPort: true,
      allowedHosts: vueSrv.allowedHosts,
    },
    // OPT-20260807-049: 全量并行下重组件挂载测试偶发超时（TaskDetail.smoke/CreateTaskModal/
    // ProjectDetail/UserGitSiteOAuthSettings/TaskDetailLinkedProjectsPanel，隔离运行全过、
    // 失败集合随运行漂移、重跑 0 失败），提高默认 testTimeout/hookTimeout 吸收并行负载抖动，
    // 避免 CI 假红。
    // OPT-20260824-016: 纯 node:test 静态分析文件（*.bootstrapCloneLog / *.bootstrapFailureTrace /
    // *.zlog-released / Register.accessCode / TaskDetailCommentsSection）与 vitest 套件并存但无
    // 排除配置，全量跑批每轮误报「No test suite found」；按 node:test 命名约定排除。
    test: {
      testTimeout: 15000,
      hookTimeout: 15000,
      exclude: [
        ...defaultExclude,
        '**/*.bootstrapCloneLog.test.js',
        '**/*.bootstrapFailureTrace.test.js',
        '**/*.zlog-released.test.js',
        '**/Register.accessCode.test.js',
        '**/TaskDetailCommentsSection.test.js',
      ],
    },
  };
})
