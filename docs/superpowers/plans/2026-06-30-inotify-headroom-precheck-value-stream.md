# Value Stream: 修复 inotify 余量不足 — Vite ENOSPC 崩溃

> Derived from design: `docs/specs/inotify-headroom-precheck-设计文档.md`
> Updates prior analysis: `task2app/docs/superpowers/plans/2026-06-30-vite-enospc-fix-value-stream.md`

## Value Summary

开发者在 monorepo 中启动 aiProvider 时，Vite 不再因 inotify 监听耗尽而静默崩溃，HMR 热重载稳定可用。本次分析新增了 ramsync-daemon 作为主要 inotify 消费者的识别和修复。

## Related Value Streams

- **`2026-06-30-vite-enospc-fix`** (prior analysis): modification — 新增 ramsync-daemon 根因修复 + 细化增量拆分
- **`2026-05-25-runall-stability-first`**: extension — 同为 runAll 启动可靠性改进

## End-to-End Flow

```
系统级修复 (ramsync-daemon + sysctl)
  → runAll 启动 aiProvider → run.sh start_vite_dev()
    → [预检] inotify 余量检查 → 不足时告警+建议
    → npm run dev (vite)
    → 端口 LISTEN → 延迟 2s → PID 存活确认 ✅
    → Vite HMR 可用
```

## Value Increments

### Increment 1: 释放 inotify 配额（根因修复）
**Value to user:** 系统 inotify 余量从 ~50% 恢复到 ~90%+，所有文件监听服务受益
**Scope:**
- ramsync-daemon.sh: `inotifywait -r` → `sleep + rsync` 轮询，释放 ~34,000 watches
**Depends on:** nothing

### Increment 2: 提高系统上限（防御层）
**Value to user:** inotify 总容量从 65,536 提升至 524,288，多服务并发不再耗尽
**Scope:**
- sysctl: `fs.inotify.max_user_watches=524288` 持久化
**Depends on:** Increment 1（先释放配额，再扩容）

### Increment 3: 启动可靠性增强（应用层防御）
**Value to user:** Vite 启动失败时明确报错（不再静默崩溃），低余量时提前告警
**Scope:**
- `start_vite_dev()`: 存活二次确认（sleep 2 + PID 检查）
- `start_vite_dev()`: inotify 余量预检（<20% 时黄色告警+修复建议）
- vite.config.js: `watch.ignored` 显式排除 node_modules/.git/dist
**Depends on:** Increment 2（有扩容后才做预检才有意义）

## YAML Impact

No new YAML entries needed — infrastructure fix, no new fields, no new test files, no domain model changes.
