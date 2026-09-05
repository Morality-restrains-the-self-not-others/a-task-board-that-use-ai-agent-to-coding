# 价值流：金丝雀平滑重启

- **日期:** 2026-09-03
- **设计:** `docs/superpowers/specs/2026-09-03-runall-smooth-canary-restart-design.md`

## 增量

单一垂直增量 **VS-1 滚动金丝雀重启**：运维在 9999 点击全部重启或精准编译重启 → 依赖序逐服务 overlap/drain → 栈保持可探活。

```
运维打开 Status
  → 确认「金丝雀平滑重启」
  → POST /api/restart-all | /api/precise-restart
  → 每服务：编译? → overlap start → 新 PID 健康 → SIGTERM 旧组
  → SSE 完成
```

测试点见 `docs/intents/platform/runall_smooth_canary_restart.test-intent.md`（T1–T7）。

无业务用户价值流；不写入 `conf/value-stream.yaml` 业务域（该文件为 SaaS 用户旅程）。平台测例在 runAll Go test。
