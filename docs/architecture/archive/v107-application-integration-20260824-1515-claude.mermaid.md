# v107 application-integration — 镜像/技能 mention 存储 ID 化 + 名↔ID 映射持久化 (target)

```mermaid
graph TD;
  gw["taskGateway / APISIX"];
  taskFE["taskFE SPA 🟡 MODIFIED v107"];
  tts["taskTaskService 🟡 MODIFIED v107"];
  cloud["taskCloudService 🟡 MODIFIED v107"];
  django["saas-backend (Django legacy)"];
  dataTTS["task_tasks: image_skill_id / container_image_snapshot 🟡 MODIFIED v107"];
  dataCloud["cloud_tenant_installed_images: image_skills_json(技能 id) 🟡 MODIFIED v107"];
  plateauV106["Plateau v106"];
  plateauV107["Plateau v107 镜像/技能存储 ID 化"];
  gapId["Gap: 技能无 ID、任务仅存镜像 ID、名↔ID 映射未持久化"];
  wpId["WP-image-skill-id-mapping"];
  gw --> taskFE;
  gw --> tts;
  gw --> django;
  tts --> cloud;
  tts --> dataTTS;
  cloud --> dataCloud;
  plateauV106 --> gapId;
  wpId --|> gapId;
  wpId --|> plateauV107;
```
