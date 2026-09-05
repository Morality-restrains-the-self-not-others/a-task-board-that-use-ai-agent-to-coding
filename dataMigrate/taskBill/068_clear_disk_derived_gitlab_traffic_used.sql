-- 068: 清除磁盘占用误写入的 GitLab 已用流量水位（ADR-0036 存量）
-- persistTrafficFloorFromDisk 曾把 disk_used_bytes 换算 GB 写入 traffic_used_gb，
-- 设置页「已用磁盘」与「已用流量」因此相同。读路径已丢弃该水位；本迁移清库存量。
-- 整数 GB ≥ 1 可能来自 chargeGitlabTraffic，与磁盘碰巧相等时保留。
-- 比较用 1e-6 GB 容差：Go diskUsedGBFromBytes 保留 6 位小数，MySQL ROUND
-- 与 DOUBLE 显示会导致 0.01049 对不上 0.010490417。

UPDATE billing_tenant_gitlab_resource
SET
    traffic_used_gb = 0,
    updated_at = NOW()
WHERE
    disk_used_bytes > 0
    AND traffic_used_gb > 0
    AND traffic_used_gb < 1
    AND ABS(
        traffic_used_gb
        - disk_used_bytes / (1024.0 * 1024.0 * 1024.0)
    ) < 0.000001;
