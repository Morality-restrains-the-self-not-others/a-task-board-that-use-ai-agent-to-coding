# v115 enterprise-landscape — 排队调度历史 (current)

```mermaid
graph TD;
  user["工作空间成员"];
  viewNow["查看当前时段与槽位"];
  viewHist["查看调度历史"];
  fe["taskFE 工作台"];
  tts["taskTaskService"];
  p114["Plateau v114"];
  p115["Plateau v115"];
  g["Gap: 无调度历史"];
  wp["WP-queue-schedule-history"];
  user --> viewNow;
  user --> viewHist;
  user --> fe;
  fe --> tts;
  tts --> viewHist;
  p114 --> g;
  wp --|> g;
  wp --|> p115;
```
