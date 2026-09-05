# 测试意图：GitLab 流量费定价

1. 创建套餐时未传流量费 → 默认 1；传入自定义值 → 持久化成功
2. sync 后 `billing_unit` 存在 `gitlab_traffic`，单位 `GB`
3. 切换套餐后 `locked_gitlab_traffic_points_per_gb` 等于新套餐价
4. `pricingPackageUserDescription` 含「GitLab 流量费」与「同区域内网不计费」
5. 超管价格管理 API/UI 可读写该字段
