# v111 application-integration — 登录历史 (target)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  auth["taskAuth"];
  hist["auth_login_history"];
  evLogin["USER_LOGGED_IN"];
  plateauV110["Plateau v110"];
  plateauV111["Plateau v111 登录历史"];
  gapHist["Gap: 仅 last_login，无 IP/入口历史"];
  wpHist["WP-login-history"];
  taskFE --> gw;
  gw --> auth;
  auth --> hist;
  auth --> evLogin;
  plateauV110 --> gapHist;
  wpHist --|> gapHist;
  wpHist --|> plateauV111;
```
