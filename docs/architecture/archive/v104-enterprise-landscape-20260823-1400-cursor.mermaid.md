# v104 enterprise-landscape mermaid

开发者关容器后仍能复查执行日志；平台管理员配置 COS 桶与路径规则。

```mermaid
flowchart LR
  Dev[开发者] --> FE[taskFE 任务详情]
  Admin[平台管理员] --> Page[系统管理 COS]
  FE --> Cloud[taskCloudService]
  Page --> Cloud
  Cloud --> COS[Tencent COS]
```
