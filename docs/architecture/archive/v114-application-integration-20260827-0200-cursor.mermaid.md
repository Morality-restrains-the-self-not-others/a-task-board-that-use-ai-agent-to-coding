# v114 application-integration — 启动日志 COS 归档 (target)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  cloud["taskCloudService"];
  shards["ccb log shards"];
  ptr["cloud_comment_startup_log_object"];
  cos["Tencent COS"];
  evArch["CommentStartupLogArchived"];
  p113["Plateau v113 仅分片"];
  p114["Plateau v114 分片+COS"];
  gapLogs["Gap: 启动日志不在 COS"];
  wpLogs["WP-ccb-startup-logs-cos-archive"];
  taskFE --> gw;
  gw --> cloud;
  cloud --> shards;
  cloud --> ptr;
  cloud --> cos;
  cloud --> evArch;
  p113 --> gapLogs;
  wpLogs --|> gapLogs;
  wpLogs --|> p114;
```
