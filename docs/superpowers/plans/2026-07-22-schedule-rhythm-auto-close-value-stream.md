# 价值流：排队调度自动关闭

- 日期：2026-07-22
- step id：`schedule-rhythm-auto-close`

## 增量流

```
配置 auto_close → 排队窗口运行 → T-5min 预告容器 → 窗口结束释放机器 → 槽位清空 → 下一窗口可再调度
```

## 测试点（对齐 flows）

| TP | 描述 |
|----|------|
| TP-AC-1 | 勾选保存回显 |
| TP-AC-2 | T-5min 通知一次 |
| TP-AC-3 | 窗外释放 + 清槽 |
| TP-AC-4 | 未勾选不释放 |

## CRG 社区对齐

稀疏图未给出社区；对齐既有 queued-schedule 应用集成社区（TTS–Cloud–Gateway–Agent）。
