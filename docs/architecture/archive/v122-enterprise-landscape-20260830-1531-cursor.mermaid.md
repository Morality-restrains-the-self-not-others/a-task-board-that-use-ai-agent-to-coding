# v122 enterprise-landscape — 二进制部署与独立配置仓 (target)

```mermaid
graph TD;
  dev["研发"];
  ops["平台运维"];
  publish["发布产物"];
  deploy["钉版本并部署"];
  srcRepo["源码仓 GitHub"];
  cfgRepo["daydaymoney-deploy"];
  pkg["GitHub Packages"];
  host["部署机 DEPLOY_ROOT"];
  runAll["runAll :9999"];
  bins["Go 二进制 + taskEvents + taskFE dist"];
  oldConf["源码仓 conf 运行时 SSOT DEPRECATED"];
  p121["Plateau v121"];
  p122["Plateau v122"];
  gap["Gap: 源码与运行时 conf 同树"];
  wp["WP-binary-deploy-config-repo"];
  dev --> publish;
  ops --> deploy;
  srcRepo --> pkg;
  cfgRepo --> host;
  pkg --> host;
  host --> runAll;
  runAll --> bins;
  runAll -.-> cfgRepo;
  oldConf --- srcRepo;
  p121 --> gap;
  wp --|> gap;
  wp --|> p122;
```
