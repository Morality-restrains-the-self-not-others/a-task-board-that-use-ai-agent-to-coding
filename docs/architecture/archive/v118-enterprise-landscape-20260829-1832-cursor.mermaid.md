# v118 enterprise-landscape — Git OAuth 资源使用标记 (current)

```mermaid
graph TD;
  enabler["启用自动运行的人 / 发评人"];
  projUser["项目页使用者"];
  markP["项目页 OAuth 回流打标"];
  markC["评论 / 自动运行 OAuth 回流打标"];
  fe["taskFE"];
  goauth["taskGitOauth"];
  proj["taskProjectService"];
  task["taskTaskService"];
  p117["Plateau v117"];
  p118["Plateau v118"];
  gap["Gap: 有 L1 即可任意换票"];
  wp["WP-git-oauth-resource-grant"];
  projUser --> markP;
  enabler --> markC;
  projUser --> fe;
  enabler --> fe;
  fe --> goauth;
  goauth --> proj;
  goauth --> task;
  p117 --> gap;
  wp --|> gap;
  wp --|> p118;
```
