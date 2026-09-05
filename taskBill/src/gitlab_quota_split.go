package main

// OPT-20260820-041: GitLab 磁盘/流量按赠送与购买拆分展示。
//
// 购买侧订单不写 billing_resource_grant 批次，仅更新 billing_tenant_gitlab_resource 合计；
// 后台赠送（admin_grant）写 billing_resource_grant（source_kind=gift, region=区域 slug）。
// 因此「赠送」由 grant 批次直接汇总，「购买」由区域合计 - 赠送推导。

type gitlabQuotaSplit struct {
	DiskGifted    int64
	TrafficGifted int64
}

// getGitlabQuotaSplit 汇总租户各区域 GitLab 磁盘/流量赠送批次，键为区域 slug（空串表示未标区域）。
func getGitlabQuotaSplit(tenantID int64) (map[string]gitlabQuotaSplit, error) {
	rows, err := db.Query(`
		SELECT region, resource_type, COALESCE(SUM(quantity), 0)
		FROM billing_resource_grant
		WHERE tenant_id = ? AND source_kind = ?
		  AND resource_type IN (?, ?)
		GROUP BY region, resource_type`,
		tenantID, grantSourceGift, ResourceTypeGitlabDisk, ResourceTypeGitlabTraffic,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]gitlabQuotaSplit{}
	for rows.Next() {
		var region, resourceType string
		var qty int64
		if err := rows.Scan(&region, &resourceType, &qty); err != nil {
			return nil, err
		}
		s := out[region]
		switch resourceType {
		case ResourceTypeGitlabDisk:
			s.DiskGifted += qty
		case ResourceTypeGitlabTraffic:
			s.TrafficGifted += qty
		}
		out[region] = s
	}
	return out, rows.Err()
}
