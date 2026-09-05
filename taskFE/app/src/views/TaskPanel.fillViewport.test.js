// @vitest-environment jsdom
/**
 * 看板「其他」区撑满视口剩余高度：结构类名与 CSS 契约回归。
 * 验收：#task-panel-board-other 所在高度链具备 flex/min-h-0，底边可贴浏览器底部。
 */
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

if (!process.env.VITEST) {
  console.log('[skip] TaskPanel.fillViewport.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')

  const taskPanelVue = readFileSync(join(__dirname, 'TaskPanel.vue'), 'utf8')
  const taskPanelCss = readFileSync(join(__dirname, 'TaskPanel.css'), 'utf8')
  const workPanelVue = readFileSync(join(__dirname, 'WorkPanel.vue'), 'utf8')
  const boardSectionVue = readFileSync(
    join(__dirname, '../components/DeliverableBoardSection.vue'),
    'utf8',
  )
  const kanbanVue = readFileSync(
    join(__dirname, '../components/DeliverableKanbanBoard.vue'),
    'utf8',
  )
  const appVue = readFileSync(join(__dirname, '../App.vue'), 'utf8')

  describe('WorkPanel 看板区撑满视口（滚动条贴底）', () => {
    it('App 壳层将主内容区约束在视口内并允许页面滚动', () => {
      expect(appVue).toMatch(/h-screen/)
      expect(appVue).toMatch(/min-h-0/)
      // 非认证页内容区可滚动；WorkPanel 自身 overflow-hidden 贴底
      expect(appVue).toMatch(/overflow-y-auto/)
    })

    it('WorkPanel 根节点填满路由区且禁止整页撑开', () => {
      expect(workPanelVue).toMatch(/data-alias="view-work-panel"[^>]*class="[^"]*\bh-full\b/)
      expect(workPanelVue).toMatch(/data-alias="view-work-panel"[^>]*class="[^"]*\bmin-h-0\b/)
      expect(workPanelVue).toMatch(/data-alias="view-work-panel"[^>]*class="[^"]*\boverflow-hidden\b/)
      expect(workPanelVue).not.toMatch(/data-alias="view-work-panel"[^>]*class="[^"]*\bmin-h-screen\b/)
    })

    it('TaskPanel 容器与 other section 形成纵向 flex 高度链', () => {
      expect(taskPanelVue).toMatch(/id="task-panel-container"[^>]*class="[^"]*\bmin-h-0\b/)
      expect(taskPanelVue).toMatch(/id="task-panel-container"[^>]*class="[^"]*\bflex\b/)
      // 底部 padding 去掉，避免横向滚动条悬空
      expect(taskPanelVue).toMatch(/pb-0/)
      expect(taskPanelVue).toMatch(/fill-remaining|fillRemaining/)
    })

    it('DeliverableBoardSection 支持 fillRemaining 撑满剩余高度', () => {
      expect(boardSectionVue).toMatch(/fillRemaining/)
      expect(boardSectionVue).toMatch(/deliverable-board-section--fill|flex-1/)
      expect(boardSectionVue).toMatch(/min-h-0/)
    })

    it('Kanban board 根节点具备 h-full/min-h-0 以便列高与底边贴齐', () => {
      expect(kanbanVue).toMatch(/data-alias="deliverable-kanban-board"/)
      expect(kanbanVue).toMatch(/\bh-full\b/)
      expect(kanbanVue).toMatch(/\bmin-h-0\b/)
    })

    it('TaskPanel.css 为 other section / board 保留高度传递规则', () => {
      expect(taskPanelCss).toMatch(/deliverable-section-other/)
      expect(taskPanelCss).toMatch(/deliverable-kanban-board/)
      expect(taskPanelCss).toMatch(/height:\s*100%|flex:\s*1/)
    })

    it('过滤块不再 max-height:40% 自滚，改由 #task-panel-container 纵向滚动完整展示过滤栏', () => {
      // OPT-20260810-049：拒绝过滤块 max-height:40% 自滚方案
      const filterBlockRule =
        taskPanelCss.match(/\.deliverable-filter-block\s*\{[^}]*\}/)?.[0] || ''
      expect(filterBlockRule).not.toMatch(/max-height:\s*40%/)
      expect(filterBlockRule).not.toMatch(/overflow-y:\s*auto/)
      // 容器改为可纵向滚动
      expect(taskPanelVue).toMatch(
        /id="task-panel-container"[^>]*class="[^"]*\boverflow-y-auto\b/,
      )
      // 「其他」看板 flex: 1 0 0% + min-height: 280px（有剩余空间贴底、空间不足不压缩）
      expect(taskPanelCss).toMatch(/flex:\s*1\s+0\s+0%/)
      expect(taskPanelCss).toMatch(/min-height:\s*280px/)
    })
  })
}
