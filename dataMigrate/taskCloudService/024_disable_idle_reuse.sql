-- 024: disable cross-task idle machine reuse (ADR-0013)
-- 存量策略 prefer_idle_reuse 全部置 0；新建行默认 0。闲置自动回收不受影响。
-- 幂等：UPDATE 与 MODIFY DEFAULT 可重复执行。

ALTER TABLE cloud_workspace_machine_policies
  MODIFY prefer_idle_reuse TINYINT NOT NULL DEFAULT 0;

UPDATE cloud_workspace_machine_policies
   SET prefer_idle_reuse = 0
 WHERE prefer_idle_reuse <> 0;
