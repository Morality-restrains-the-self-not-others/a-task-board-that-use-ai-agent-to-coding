# v111 enterprise-landscape — 登录历史 (target)

```mermaid
graph TD;
  user["登录用户"];
  admin["平台超管"];
  loginCust["用户入口登录"];
  loginAdm["管理员入口登录"];
  viewSelf["查看自己的登录历史"];
  viewOther["查看指定用户登录历史"];
  fe["taskFE 账号中心 / 超管用户表"];
  auth["taskAuth"];
  hist["auth_login_history"];
  p110["Plateau v110"];
  p111["Plateau v111"];
  g["Gap: 用户看不到登录 IP 历史"];
  wp["WP-login-history"];
  user --> loginCust;
  admin --> loginAdm;
  user --> viewSelf;
  admin --> viewOther;
  user --> fe;
  admin --> fe;
  fe --> auth;
  auth --> hist;
  p110 --> g;
  wp --|> g;
  wp --|> p111;
```
