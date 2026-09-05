# v107 enterprise-landscape — 镜像/技能 mention 存储 ID 化 + 名↔ID 映射持久化 (target)

```mermaid
graph TD;
  dev["开发者 / 外部用户"];
  taskSvc["创建/编辑任务并绑定镜像技能 🟡 MODIFIED v107"];
  taskFE["taskFE SPA 🟡 MODIFIED v107"];
  tts["taskTaskService 🟡 MODIFIED v107"];
  cloud["taskCloudService 🟡 MODIFIED v107"];
  mysql["MySQL (task/cloud 库)"];
  plateauV106["Plateau v106"];
  plateauV107["Plateau v107 镜像/技能存储 ID 化"];
  gapId["Gap: 技能无 ID、任务仅存镜像 ID、名↔ID 映射未持久化"];
  wpId["WP-image-skill-id-mapping"];
  dev --> taskSvc;
  tts --> taskSvc;
  taskFE --> tts;
  tts --> cloud;
  mysql --> tts;
  mysql --> cloud;
  plateauV106 --> gapId;
  wpId --|> gapId;
  wpId --|> plateauV107;
```
