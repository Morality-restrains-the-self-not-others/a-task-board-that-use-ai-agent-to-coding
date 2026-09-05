# v123 application-integration — conf-local 两层加载 (current)

```mermaid
graph TD;
  skel["1 conf/app/config.yaml"];
  conflocal["2 conf-local/app/config.yaml"];
  confload["confload / conf_loader"];
  svc["Go 服务 / taskEvents / taskFE"];
  ciGate["check_conf_local_secrets.py"];
  p122["Plateau v122"];
  p123["Plateau v123"];
  wp["WP-conf-local-secrets-completion"];
  confload --> skel;
  confload --> conflocal;
  confload --> svc;
  ciGate --- skel;
  p122 --> wp;
  wp --|> p123;
```
