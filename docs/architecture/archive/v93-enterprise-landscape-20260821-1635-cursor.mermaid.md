# v93 enterprise-landscape

```mermaid
flowchart LR
  Vendor[厂商] --> Upload[登记镜像]
  Member[租户成员] --> Market[镜像市场]
  Member --> Task[创建任务/评论]
  Upload --> AP[taskAiProvider 抽取绑定]
  AP --> Market
  Market --> Task
```
