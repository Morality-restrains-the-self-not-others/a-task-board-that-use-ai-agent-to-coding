# v123 enterprise-landscape — conf-local 唯一 overlay (current)

```mermaid
graph TD;
  ops["平台运维"];
  rotate["废弃已泄露凭据并签发"];
  cfgRepo["daydaymoney-deploy"];
  host["部署机 DEPLOY_ROOT"];
  runAll["runAll :9999"];
  confload["confload / conf_loader"];
  conflocal["conf-local 唯一 overlay"];
  example["conf-local.example"];
  p122["Plateau v122"];
  p123["Plateau v123"];
  wp["WP-conf-local-secrets-completion"];
  ops --> rotate;
  cfgRepo --> host;
  host --> runAll;
  confload --> conflocal;
  confload --> runAll;
  example --- conflocal;
  p122 --> wp;
  wp --|> p123;
```
