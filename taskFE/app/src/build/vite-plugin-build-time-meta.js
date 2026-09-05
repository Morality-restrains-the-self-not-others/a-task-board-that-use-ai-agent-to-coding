/**
 * 在构建产物 index.html 注入 <meta name="build-time" content="<ISO-8601 UTC>">，
 * 供 E2E / 验收脚本对比产物与源码新鲜度（OPT-20260824-056）。
 *
 * 背景：:4000 产物服务常承载旧 public/html（release symlink 未切或未重新构建），
 * 而源码已推进；E2E UI 断言打旧产物导致「产物与源码漂移」类失败，诊断浪费多轮。
 * 注入 build-time 后，E2E 可读取该标记并与源码最新 mtime 比较，漂移时先提示重建。
 */

/** 生成 build-time meta 标签字符串（纯函数，便于单测）。 */
export function buildTimeMetaTag(buildTimeIso) {
  return `<meta name="build-time" content="${buildTimeIso}">`
}

/** 把 build-time meta 注入 <head>；无 </head> 时退化为插到 html 最前。 */
export function injectBuildTimeMeta(html, buildTimeIso) {
  const tag = buildTimeMetaTag(buildTimeIso)
  if (/<\/head>/i.test(html)) {
    return html.replace(/<\/head>/i, `${tag}</head>`)
  }
  return `${tag}\n${html}`
}

/**
 * Vite 插件：仅生产构建（apply: 'build'）向 index.html 注入 build-time。
 * @param {{ now?: () => Date }} [options] now 可注入便于单测固定时间。
 */
export function buildTimeMetaPlugin(options = {}) {
  const now = options.now || (() => new Date())
  return {
    name: 'task2app-build-time-meta',
    apply: 'build',
    transformIndexHtml(html) {
      return injectBuildTimeMeta(html, now().toISOString())
    },
  }
}
