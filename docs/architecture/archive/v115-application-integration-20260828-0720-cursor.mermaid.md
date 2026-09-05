# v115 application-integration — 排队调度历史 (current)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  tts["taskTaskService"];
  hist["task_queued_schedule_history"];
  rhythm["workspace_schedule_rhythms"];
  evIn["WorkspaceScheduleWindowEntered"];
  evOut["WorkspaceScheduleWindowExited"];
  p114["Plateau v114 仅快照"];
  p115["Plateau v115 可读历史"];
  gapHist["Gap: 状态栏无调度轨迹"];
  wpHist["WP-queue-schedule-history"];
  taskFE --> gw;
  gw --> tts;
  tts --> hist;
  tts --> rhythm;
  tts --> evIn;
  tts --> evOut;
  p114 --> gapHist;
  wpHist --|> gapHist;
  wpHist --|> p115;
```
