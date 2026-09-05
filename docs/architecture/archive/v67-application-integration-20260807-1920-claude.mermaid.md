# v67 Application Integration — 任务帖存续期定价 (Target, 2026-08-07 19:20)

```mermaid
graph TD;
  gateway["APISIX Gateway (:18081)"];
  taskTask["taskTaskService (:8021) [MODIFIED v67: +post_expires_at +renew 接口 +到期校验]"];
  taskBill["taskBill (:8004) [MODIFIED v67: 执行费→创建帖费 +server_start_renewal +renewal API]"];
  taskCloud["taskCloudService (:8018) [MODIFIED v67: bill_client 退役]"];
  taskEvents["taskEvents (:8023) [MODIFIED v67: +新事件 +每日到期扫描]"];
  oldChargeServerStart["charge-server-start 按次扣费 [DEPRECATED v67]"];
  taskTaskDB["taskTask DB (+post_expires_at 🆕)"];
  taskBillDB["taskBill DB (+server_start_renewal 单位)"];
  evtPostRenewed["TASK_POST_RENEWED 🆕"];
  evtPostExpired["TASK_POST_EXPIRED 🆕"];
  kafkaBroker["Kafka"];
  plateauV66["Plateau v66 — 多会话互知"];
  plateauV67["Plateau v67 — 任务帖存续期"];
  gapPost["Gap: 帖子无存续期、按次扣费、无续存"];
  wpPost["WP-post-lifecycle"];
  gateway --> taskTask;
  gateway --> taskBill;
  taskTask --> taskBill;
  taskTask ..> taskTaskDB;
  taskBill ..> taskBillDB;
  taskTask --> evtPostRenewed;
  taskEvents --> evtPostExpired;
  evtPostRenewed --> kafkaBroker;
  evtPostExpired --> kafkaBroker;
  taskCloud --> oldChargeServerStart;
  oldChargeServerStart --- taskBill;
  plateauV66 --> gapPost;
  wpPost --|> gapPost;
  wpPost --|> plateauV67;
```
