// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] installedImageLabel.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    formatInstalledImageRunLabel,
    formatInstalledImageUpdateTime,
    formatTaskDefaultImageDisplay,
    resolveTaskDefaultImageLabel,
  } = await import('./installedImageLabel.js')

  describe('formatInstalledImageRunLabel', () => {
    it('joins name and version', () => {
      expect(formatInstalledImageRunLabel('trae-agent', 'x86_64-latest')).toBe(
        'trae-agent:x86_64-latest',
      )
    })

    it('does not duplicate version already in name', () => {
      expect(formatInstalledImageRunLabel('trae-agent:v1', 'v1')).toBe('trae-agent:v1')
    })

    it('returns name when version empty', () => {
      expect(formatInstalledImageRunLabel('plain', '')).toBe('plain')
    })
  })

  describe('formatInstalledImageUpdateTime', () => {
    // 本地时区渲染对照（测试机 +08 / CI UTC 均可，避免硬编码时区偏移）
    const localRender = (iso) => {
      const d = new Date(iso)
      const pad = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
    }

    it('优先 updated_at（目录快照），格式为 YYYY-MM-DD HH:mm', () => {
      const iso = '2026-08-20T10:11:12Z'
      expect(
        formatInstalledImageUpdateTime({ updated_at: iso, installed_at: '2026-08-01T00:00:00Z' }),
      ).toBe(localRender(iso))
    })

    it('updated_at 缺失时回退 installed_at', () => {
      const iso = '2026-08-01T02:03:00Z'
      expect(formatInstalledImageUpdateTime({ installed_at: iso })).toBe(localRender(iso))
    })

    it('两者皆空或解析失败返回空串', () => {
      expect(formatInstalledImageUpdateTime({})).toBe('')
      expect(formatInstalledImageUpdateTime({ updated_at: 'not-a-date' })).toBe('')
      expect(formatInstalledImageUpdateTime(null)).toBe('')
    })
  })

  describe('resolveTaskDefaultImageLabel', () => {
    const catalog = [
      { id: '878236719185424384', name: 'trae-agent', version: 'x86_64-latest' },
    ]

    it('U1 目录命中时返回 name:version', () => {
      expect(
        resolveTaskDefaultImageLabel({
          imageId: '878236719185424384',
          installedImages: catalog,
        }),
      ).toBe('trae-agent:x86_64-latest')
    })

    it('U3 目录未命中时回退评论 container_image_label', () => {
      expect(
        resolveTaskDefaultImageLabel({
          imageId: '877435294134071296',
          installedImages: catalog,
          comments: [
            { container_image_id: '877435294134071296', container_image_label: 'trae-agent' },
          ],
        }),
      ).toBe('trae-agent')
    })

    it('不把目录里同名不同 id 的镜像当成已绑定镜像', () => {
      expect(
        resolveTaskDefaultImageLabel({
          imageId: '877435294134071296',
          installedImages: catalog,
        }),
      ).toBe('')
    })

    it('回退任务嵌套 container_image 对象', () => {
      expect(
        resolveTaskDefaultImageLabel({
          imageId: 'img-1',
          nestedImage: { id: 'img-1', name: 'nested', version: '9' },
        }),
      ).toBe('nested:9')
    })
  })

  describe('formatTaskDefaultImageDisplay', () => {
    it('U4 仅有 id 时展示已绑定', () => {
      expect(formatTaskDefaultImageDisplay('', '877435294134071296')).toBe('已绑定')
    })

    it('U5 无 id 时展示未设置', () => {
      expect(formatTaskDefaultImageDisplay('', '')).toBe('未设置')
    })

    it('有解析结果时原样返回', () => {
      expect(formatTaskDefaultImageDisplay('trae-agent', 'id-1')).toBe('trae-agent')
    })

    // OPT-20260820-045: 镜像已卸载（container_image_removed）时展示「已卸载」而非「已绑定」
    it('removed 标记时展示已卸载（不再笼统已绑定）', () => {
      expect(formatTaskDefaultImageDisplay('', '877435294134071296', true)).toBe('已卸载')
    })

    it('有 label 时优先 label（即使 removed 标记在场）', () => {
      expect(formatTaskDefaultImageDisplay('trae-agent:1', 'id-1', true)).toBe('trae-agent:1')
    })
  })

  describe('resolveTaskDefaultImageLabel 水合快照优先', () => {
    // OPT-20260820-045: 后端 taskToJSON 单任务路径水合 container_image 快照，
    // 目录未命中时应优先用该快照（name:version）而非评论 mention 兜底。
    it('目录未命中时优先嵌套 container_image 快照（含 version）', () => {
      expect(
        resolveTaskDefaultImageLabel({
          imageId: '877435294134071296',
          installedImages: [],
          nestedImage: { id: '877435294134071296', name: 'trae-agent', version: 'x86_64-latest' },
          comments: [
            { container_image_id: '877435294134071296', container_image_label: 'trae-agent' },
          ],
        }),
      ).toBe('trae-agent:x86_64-latest')
    })
  })
}
