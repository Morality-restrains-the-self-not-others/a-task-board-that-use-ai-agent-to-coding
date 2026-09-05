/**
 * 根据文件路径推断展示用类型分类（图标）。
 * @param {string} filePath
 * @returns {'code'|'markup'|'data'|'image'|'archive'|'binary'|'text'|'unknown'}
 */
export function fileTypeFromPath(filePath) {
  const raw = String(filePath || '').trim().replace(/\\/g, '/')
  const base = raw.includes('/') ? raw.slice(raw.lastIndexOf('/') + 1) : raw
  const dot = base.lastIndexOf('.')
  const ext = dot > 0 ? base.slice(dot + 1).toLowerCase() : ''

  if (!ext) {
    // 常见无扩展名二进制（如 jre/lib/modules）
    if (/^(modules|vmlinux|vmlinuz)$/i.test(base)) return 'binary'
    return 'unknown'
  }

  if (
    [
      'js',
      'jsx',
      'mjs',
      'cjs',
      'ts',
      'tsx',
      'vue',
      'py',
      'go',
      'java',
      'kt',
      'rs',
      'c',
      'cc',
      'cpp',
      'h',
      'hpp',
      'cs',
      'rb',
      'php',
      'swift',
      'scala',
      'sh',
      'bash',
      'zsh',
      'ps1',
    ].includes(ext)
  ) {
    return 'code'
  }
  if (['html', 'htm', 'xml', 'xhtml', 'svg'].includes(ext)) return 'markup'
  if (['json', 'yaml', 'yml', 'toml', 'ini', 'csv', 'tsv'].includes(ext)) return 'data'
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'ico', 'bmp', 'avif'].includes(ext)) return 'image'
  if (['zip', 'tar', 'gz', 'tgz', 'bz2', 'xz', '7z', 'rar', 'jar', 'war'].includes(ext)) {
    return 'archive'
  }
  if (
    [
      'bin',
      'exe',
      'dll',
      'so',
      'dylib',
      'o',
      'a',
      'class',
      'pyc',
      'wasm',
      'pdf',
      'woff',
      'woff2',
      'ttf',
      'otf',
      'mp3',
      'mp4',
      'webm',
      'ogg',
      'wav',
    ].includes(ext)
  ) {
    return 'binary'
  }
  if (['md', 'txt', 'log', 'rst', 'adoc', 'css', 'scss', 'less', 'map'].includes(ext)) {
    return 'text'
  }
  return 'unknown'
}

/**
 * @param {string} filePath
 * @returns {string} 简短中文标签
 */
export function fileTypeLabel(filePath) {
  const cat = fileTypeFromPath(filePath)
  const map = {
    code: '代码',
    markup: '标记',
    data: '数据',
    image: '图片',
    archive: '压缩包',
    binary: '二进制',
    text: '文本',
    unknown: '文件',
  }
  return map[cat] || '文件'
}
