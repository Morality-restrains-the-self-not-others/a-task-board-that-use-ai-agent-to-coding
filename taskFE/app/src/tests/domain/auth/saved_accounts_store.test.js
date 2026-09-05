// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] saved_accounts_store.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const {
    ACTIVE_ACCOUNT_STORAGE_KEY,
    clearSavedAccounts,
    listSavedAccounts,
    removeSavedAccount,
    upsertSavedAccount,
    getSavedAccount,
    setActiveAccount,
    getActiveAccount,
    getActiveToken,
    getCurrentUserId,
    onAccountStateChanged,
  } = await import('../../../domain/auth/services/saved_accounts_store.js')

  describe('saved_accounts_store', () => {
    beforeEach(() => {
      localStorage.clear()
      clearSavedAccounts()
    })

    it('upsert 新账号并 list', () => {
      upsertSavedAccount({
        userId: '111',
        username: 'alice',
        token: 'tok-a',
        avatarUrl: null,
      })
      const list = listSavedAccounts()
      expect(list).toHaveLength(1)
      expect(list[0].userId).toBe('111')
      expect(list[0].username).toBe('alice')
      expect(list[0].token).toBe('tok-a')
    })

    it('同 userId 更新 username/token 不增加数量', () => {
      upsertSavedAccount({ userId: '111', username: 'a', token: 't1' })
      upsertSavedAccount({ userId: '111', username: 'a2', token: 't2' })
      const list = listSavedAccounts()
      expect(list).toHaveLength(1)
      expect(list[0].username).toBe('a2')
      expect(list[0].token).toBe('t2')
    })

    it('多账号写入收敛为单账号（新账号整体替换旧槽，OPT-20260807-015）', () => {
      upsertSavedAccount({ userId: '1', username: 'a', token: 't1' })
      upsertSavedAccount({ userId: '2', username: 'b', token: 't2' })
      upsertSavedAccount({ userId: '3', username: 'c', token: 't3' })
      const list = listSavedAccounts()
      expect(list).toHaveLength(1)
      expect(list[0].userId).toBe('3')
      expect(list[0].username).toBe('c')
      expect(getSavedAccount('1')).toBeNull()
      expect(getSavedAccount('2')).toBeNull()
    })

    it('remove 按 userId 删除（单账号下删除后清空）', () => {
      upsertSavedAccount({ userId: '1', username: 'a', token: 't1' })
      removeSavedAccount('1')
      expect(listSavedAccounts()).toHaveLength(0)
      expect(getSavedAccount('1')).toBeNull()
    })

    it('缺少 userId 或 token 抛错', () => {
      expect(() => upsertSavedAccount({ userId: '', token: 't' })).toThrow(/userId/)
      expect(() => upsertSavedAccount({ userId: '1', token: '' })).toThrow(/token/)
    })
  })

  describe('saved_accounts_store 活跃账号', () => {
    beforeEach(() => {
      localStorage.clear()
      clearSavedAccounts()
    })

    it('setActiveAccount 写入槽位并标记活跃', () => {
      setActiveAccount({ userId: '111', username: 'alice', token: 'tok-a' })
      expect(getActiveAccount().userId).toBe('111')
      expect(getActiveToken()).toBe('tok-a')
      expect(getCurrentUserId()).toBe('111')
      expect(localStorage.getItem(ACTIVE_ACCOUNT_STORAGE_KEY)).toBe('111')
    })

    it('无活跃标记时回退最近添加账号（旧数据迁移）', () => {
      upsertSavedAccount({ userId: '111', username: 'old', token: 't-old' })
      // 模拟旧版 localStorage 数据：有槽位但无活跃标记（写入 active 之前的历史数据）
      localStorage.removeItem(ACTIVE_ACCOUNT_STORAGE_KEY)
      expect(getActiveToken()).toBe('t-old')
      expect(getCurrentUserId()).toBe('111')
    })

    it('活跃槽被删除后无剩余账号返回 null（单账号收敛）', () => {
      setActiveAccount({ userId: '2', username: 'b', token: 't2' })
      removeSavedAccount('2')
      expect(getActiveToken()).toBeNull()
    })

    it('无任何槽位时返回 null', () => {
      expect(getActiveAccount()).toBeNull()
      expect(getActiveToken()).toBeNull()
      expect(getCurrentUserId()).toBeNull()
    })

    it('活跃标记指向已删除账号时回退最近剩余槽位（旧数据迁移）', () => {
      upsertSavedAccount({ userId: '1', username: 'a', token: 't1' })
      setActiveAccount({ userId: '1', username: 'a', token: 't1' })
      // 模拟登出：移除当前账号槽，活跃标记残留
      removeSavedAccount('1')
      upsertSavedAccount({ userId: '3', username: 'c', token: 't3' })
      expect(getActiveToken()).toBe('t3')
    })

    it('onAccountStateChanged 监听 storage 事件（跨 Tab 同步）', () => {
      const callback = vi.fn()
      const unsubscribe = onAccountStateChanged(callback)
      // 模拟其他 Tab 写入：storage 事件带 key 字段
      window.dispatchEvent(new StorageEvent('storage', { key: 'savedAccounts' }))
      expect(callback).toHaveBeenCalledWith({ event: 'authStateChanged' })

      window.dispatchEvent(new StorageEvent('storage', { key: ACTIVE_ACCOUNT_STORAGE_KEY }))
      expect(callback).toHaveBeenCalledTimes(2)

      // 无关 key 不触发
      window.dispatchEvent(new StorageEvent('storage', { key: 'unrelated' }))
      expect(callback).toHaveBeenCalledTimes(2)

      // 取消订阅后不再触发
      unsubscribe()
      window.dispatchEvent(new StorageEvent('storage', { key: 'savedAccounts' }))
      expect(callback).toHaveBeenCalledTimes(2)
    })
  })
}
