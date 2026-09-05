# v118 application-integration — Git OAuth 资源使用标记 (current)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  goauth["taskGitOauth"];
  proj["taskProjectService"];
  task["taskTaskService"];
  cred["taskCredentialService"];
  ev["taskEvents"];
  credRow["git_oauth_appusercredential"];
  pGrant["project_git_oauth_grant NEW"];
  cGrant["task_comments.oauth_grant"];
  evP["PROJECT_GIT_OAUTH_GRANTED"];
  evC["COMMENT_GIT_OAUTH_GRANTED"];
  p117["Plateau v117 有票即可用"];
  p118["Plateau v118 资源 L2"];
  gap["Gap: L1 冒充已授权"];
  wp["WP-git-oauth-resource-grant"];
  taskFE --> gw;
  gw --> goauth;
  goauth --> credRow;
  goauth --> proj;
  goauth --> task;
  proj --> pGrant;
  task --> cGrant;
  proj --> evP;
  task --> evC;
  proj --> ev;
  task --> ev;
  cred --> task;
  cred --> goauth;
  p117 --> gap;
  wp --|> gap;
  wp --|> p118;
```
