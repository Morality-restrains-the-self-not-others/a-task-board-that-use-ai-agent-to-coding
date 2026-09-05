/**
 * 为 src 下模块 URL 附加内容 hash 查询参数（?h= / &h=），满足元规则 18_static_resource_cache_bust_query。
 * 开发态经 nginx 长缓存时避免命中过期 JS；与 configureServer 的 no-cache 头双保险。
 */
import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

const HASH_PARAM = 'h'
const HASH_LEN = 12

/** @param {string} filePath */
export function shortContentHash(filePath) {
  const buf = fs.readFileSync(filePath)
  return crypto.createHash('sha256').update(buf).digest('hex').slice(0, HASH_LEN)
}

/** @param {string} id */
export function stripHashQuery(id) {
  const qIdx = id.indexOf('?')
  if (qIdx === -1) return id
  const base = id.slice(0, qIdx)
  const rest = id
    .slice(qIdx + 1)
    .split('&')
    .filter((p) => !p.startsWith(`${HASH_PARAM}=`))
  return rest.length ? `${base}?${rest.join('&')}` : base
}

/** @param {string} id */
function hashableFilePath(id) {
  return stripHashQuery(id).split('?')[0]
}

/** @param {string} id */
function shouldAttachHash(id, srcRoot) {
  const filePath = hashableFilePath(id)
  if (!filePath.startsWith(srcRoot)) return false
  if (filePath.includes(`${path.sep}node_modules${path.sep}`)) return false
  return /\.(vue|[cm]?[jt]sx?|css)$/i.test(filePath)
}

/** @param {string} id @param {string} hash */
function withHashQuery(id, hash) {
  if (new RegExp(`[?&]${HASH_PARAM}=`).test(id)) return id
  return id.includes('?') ? `${id}&${HASH_PARAM}=${hash}` : `${id}?${HASH_PARAM}=${hash}`
}

/**
 * @param {{ srcDir?: string }} [options]
 */
export function assetCacheBustQueryPlugin(options = {}) {
  /** @type {string} */
  let srcRoot = ''

  return {
    name: 'task2app-asset-cache-bust-query',
    enforce: 'post',

    configResolved(config) {
      srcRoot = path.resolve(options.srcDir || path.join(config.root, 'src'))
    },

    async resolveId(source, importer, resolveOpts) {
      if (!source || source.startsWith('\0') || source.includes('virtual:')) return null
      if (source.includes(`${HASH_PARAM}=`)) return null

      const resolved = await this.resolve(source, importer, { ...resolveOpts, skipSelf: true })
      if (!resolved) return null

      const id = typeof resolved === 'string' ? resolved : resolved.id
      if (!id || !shouldAttachHash(id, srcRoot)) return resolved

      const filePath = hashableFilePath(id)
      let hash
      try {
        hash = shortContentHash(filePath)
      } catch {
        return resolved
      }

      const bustedId = withHashQuery(id, hash)
      if (typeof resolved === 'string') return bustedId
      return { ...resolved, id: bustedId }
    },

    load(id) {
      if (!new RegExp(`[?&]${HASH_PARAM}=`).test(id)) return null
      const filePath = hashableFilePath(id)
      if (filePath.endsWith('.vue')) return null
      if (!filePath.startsWith(srcRoot)) return null
      return fs.readFileSync(filePath, 'utf-8')
    },

    handleHotUpdate({ file, server }) {
      if (!file.startsWith(srcRoot)) return
      const mods = []
      for (const mod of server.moduleGraph.getModulesByFile(file) || []) {
        mods.push(mod)
      }
      return mods
    },

    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const raw = req.url || ''
        const pathname = raw.split('?')[0]
        if (
          pathname.startsWith('/@vite') ||
          pathname.startsWith('/@id/') ||
          pathname.startsWith('/@fs/')
        ) {
          next()
          return
        }
        const isSrcModule =
          pathname.startsWith('/src/') ||
          /^\/(utils|js|components|views|css)\//.test(pathname) ||
          /\.(vue|m?js|ts|css|jsx|tsx)$/i.test(pathname)
        if (isSrcModule) {
          res.setHeader('Cache-Control', 'no-cache, must-revalidate')
        }
        next()
      })
    },
  }
}
