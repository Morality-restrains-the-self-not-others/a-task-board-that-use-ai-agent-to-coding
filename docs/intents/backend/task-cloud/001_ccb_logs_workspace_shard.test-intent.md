# 评论容器启动日志按工作空间分表 — 测试意图

## 测试目标

验证选片、隔离、list 还原、空 workspace 拒绝。

## 测试分层

- 单元：CRC32 选片、表名格式
- 集成（MySQL）：写入分片、list、跨 ws

## 用例矩阵

| ID | 用例 | 层 | 断言 |
|----|------|----|------|
| T-shard-empty | 空 workspace | 单元 | error，无表名 |
| T-shard-stable | 同 ws 两次 | 单元 | 同一 `%02d` |
| T-shard-format | 任意非空 ws | 单元 | 匹配 `^cloud_comment_container_binding_logs_[0-9]{2}$` |
| T-write-read | 写入后 list | 集成 | 消息可见 |
| T-isolation | wsA 写、wsB 读 | 集成 | B 看不到 A |
| T-timeline | pending→starting 文案 | 集成 | 顺序与文案保持 |

事件投递：本意图无新事件；不测新 publish。既有 SSE persist 回归仍跑。

## 通过标准

上述用例全绿；遗留 timeline / server scheduling persist 测试适配 workspace 后全绿。
