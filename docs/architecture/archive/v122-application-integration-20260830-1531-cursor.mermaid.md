# v122 application-integration — 二进制部署交付链 (target)

```mermaid
graph TD;
  ci["源码仓 CI"];
  pin["releases.yaml"];
  sync["deploy-sync"];
  svc["Go 服务 / taskEvents / taskFE nginx"];
  runAll["runAll"];
  confload["confload FindConfigRoot"];
  srcConf["monorepo conf 运行时 DEPRECATED"];
  p121["Plateau v121"];
  p122["Plateau v122"];
  gap["Gap: 部署机必须 clone 源码仓"];
  wp["WP-binary-deploy-config-repo"];
  ci --> pin;
  pin --> sync;
  sync --> svc;
  runAll --> confload;
  runAll --> svc;
  srcConf --- confload;
  p121 --> gap;
  wp --|> gap;
  wp --|> p122;
```
