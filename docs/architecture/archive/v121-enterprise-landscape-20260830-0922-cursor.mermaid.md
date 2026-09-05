# v121 enterprise-landscape — 意见与建议链接 (current)

```mermaid
graph TD;
  admin["平台超管"];
  member["租户成员"];
  configure["配置意见链接组"];
  openNav["打开意见与建议子菜单"];
  fe["taskFE"];
  bill["taskBill"];
  p120["Plateau v120"];
  p121["Plateau v121"];
  gap["Gap: 无消耗分层意见入口"];
  wp["WP-tenant-feedback-links"];
  admin --> configure;
  member --> openNav;
  admin --> fe;
  member --> fe;
  fe --> bill;
  bill --> openNav;
  p120 --> gap;
  wp --|> gap;
  wp --|> p121;
```
