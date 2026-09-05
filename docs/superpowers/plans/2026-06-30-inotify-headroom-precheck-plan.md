# 实施计划: inotify 余量不足预检与系统性修复

> 输入:
> - 设计文档: `docs/specs/inotify-headroom-precheck-设计文档.md`
> - 价值流: `docs/superpowers/plans/2026-06-30-inotify-headroom-precheck-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-30-inotify-headroom-precheck-nfr-clarification.md`
> - DDD: SKIP (无领域概念)

## 任务清单

### Increment 1: 释放 inotify 配额（根因修复）

#### Task 1.1: ramsync-daemon 改用轮询替代 inotify
- [ ] 文件: `/home/ljy/bin/ramsync-daemon.sh`
- [ ] 将 `monitor_loop()` 中的 `inotifywait -t 60 ... -r` 替换为 `sleep "${POLL_INTERVAL:-10}"`
- [ ] 保留 `do_sync` 调用逻辑和 `MIN_SYNC_INTERVAL` 防抖
- [ ] 语法验证: `bash -n /home/ljy/bin/ramsync-daemon.sh`
- [ ] 重启 daemon: `ramsync-daemon.sh stop && ramsync-daemon.sh start`
- [ ] 验证: 等待 30 秒后 `find /proc/$(pgrep -f ramsync-daemon)/fd -lname 'inotify*' | wc -l` 应为 0

### Increment 2: 提高系统上限（防御层）

#### Task 2.1: 持久化 sysctl 配置
- [ ] 执行: `sudo sysctl -w fs.inotify.max_user_watches=524288`
- [ ] 持久化: `echo 'fs.inotify.max_user_watches = 524288' | sudo tee /etc/sysctl.d/99-inotify.conf`
- [ ] 验证: `sysctl fs.inotify.max_user_watches` 输出 524288

### Increment 3: 启动可靠性增强（应用层防御）

#### Task 3.1: `start_vite_dev()` 存活二次确认
- [ ] 文件: `task2app/Saas_Ai_Provider/run.sh:170-173`
- [ ] 在 `if [ "$_ok" -eq 1 ]` 分支 `return 0` 之前插入:
```bash
sleep 2
if ! ps -p "$vid" > /dev/null 2>&1; then
    echo -e "${RED}Vite 进程在启动后退出（可能是 inotify 耗尽、端口冲突等），请查看：$LOG_DIR/vite.log${NC}"
    rm -f "$VITE_PID_FILE"
    return 1
fi
```

#### Task 3.2: `start_vite_dev()` inotify 余量预检
- [ ] 文件: `task2app/Saas_Ai_Provider/run.sh:149`（`echo "启动 Vite..."` 之后、`npm run dev` 之前）
- [ ] 插入 inotify 余量检查逻辑:
```bash
local _max_watches _used _avail
_max_watches=$(cat /proc/sys/fs/inotify/max_user_watches 2>/dev/null || echo 0)
if [ "$_max_watches" -gt 0 ] 2>/dev/null; then
    _used=$(find /proc/[0-9]*/fd -lname 'inotify*' 2>/dev/null | wc -l)
    _avail=$((_max_watches - _used))
    if [ "$_avail" -lt $((_max_watches / 5)) ]; then
        echo -e "${YELLOW}⚠ inotify 监听余量不足（已用 ${_used} / ${_max_watches}，可用 ${_avail}），建议：${NC}"
        echo -e "${YELLOW}  1. 提高系统上限: sudo sysctl -w fs.inotify.max_user_watches=524288${NC}"
        echo -e "${YELLOW}  2. 检查 ramsync-daemon 是否占用过多: ps aux | grep ramsync${NC}"
    fi
fi
```

#### Task 3.3: Vite 显式配置 watch.ignored
- [ ] 文件: `task2app/Saas_Ai_Provider/frontend/vite.config.js`
- [ ] 在 `server` 块中新增:
```js
watch: {
    ignored: [
        '**/node_modules/**',
        '**/.git/**',
        '**/dist/**',
        '**/__pycache__/**',
        '**/*.pyc',
        '**/logs/**',
    ],
},
```

### 验证

#### Task 4.1: 语法检查
- [ ] `bash -n task2app/Saas_Ai_Provider/run.sh` 无错误

#### Task 4.2: 功能验证
- [ ] 重启 aiProvider，确认 Vite 启动成功且无 ENOSPC 错误
- [ ] 检查 `vite.log` 确认无 ENOSPC 崩溃
- [ ] 验证 inotify 余量预检在正常余量时不误报（当前 ~2,195/65,536 → 应无警告）
- [ ] 验证 ramsync-daemon inotify 消耗为 0

---

## 依赖关系

```
Task 1.1 (ramsync) ──┐
                     ├──→ Task 4.2 (验证)
Task 2.1 (sysctl) ───┤
                     │
Task 3.1 (存活确认) ─┤
Task 3.2 (预检)    ──┤
Task 3.3 (vite cfg) ─┘

Task 4.1 (语法检查) ← 前置于 Task 4.2 (功能验证)
```

Tasks 1.1, 2.1, 3.1, 3.2, 3.3 可并行执行。

---

## 改动范围总结

| 文件 | 改动类型 | 估计行数 |
|------|---------|---------|
| `/home/ljy/bin/ramsync-daemon.sh` | 修改 monitor_loop | ~4 行 |
| `/etc/sysctl.d/99-inotify.conf` | 新增 | 1 行 |
| `task2app/Saas_Ai_Provider/run.sh` | 新增存活确认 | ~6 行 |
| `task2app/Saas_Ai_Provider/run.sh` | 新增预检逻辑 | ~10 行 |
| `task2app/Saas_Ai_Provider/frontend/vite.config.js` | 新增 watch.ignored | ~8 行 |

**总计**: ~29 行新增/修改，无新文件（除 sysctl conf），无删除。
