# v114 enterprise-landscape — 启动日志 COS (target)

```mermaid
graph TD;
  user["工作台用户"];
  fe["taskFE 工作台"];
  cloud["taskCloudService"];
  archive["归档启动日志到 COS"];
  cos["Tencent COS"];
  p113["Plateau v113"];
  p114["Plateau v114"];
  g["Gap: 启动日志未进 COS"];
  wp["WP-ccb-startup-logs-cos-archive"];
  user --> fe;
  fe --> cloud;
  cloud --> archive;
  archive --> cos;
  p113 --> g;
  wp --|> g;
  wp --|> p114;
```
