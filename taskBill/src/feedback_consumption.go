package main

func loadTenantFeedbackConsumption(tenantID int64) (map[string]float64, error) {
	out := map[string]float64{
		feedbackKindTaskPost:       0,
		feedbackKindGitlabTraffic:  0,
		feedbackKindGitlabDisk:     0,
		feedbackKindConsumedAmount: 0,
	}
	var granted, remaining float64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0), COALESCE(SUM(remaining), 0)
		FROM billing_resource_grant
		WHERE tenant_id = ? AND resource_type = ?`,
		tenantID, ResourceTypeTaskPost,
	).Scan(&granted, &remaining)
	if err != nil {
		return nil, err
	}
	consumed := granted - remaining
	if consumed < 0 {
		consumed = 0
	}
	out[feedbackKindTaskPost] = consumed

	var traffic, diskBytes float64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(traffic_used_gb), 0), COALESCE(SUM(disk_used_bytes), 0)
		FROM billing_tenant_gitlab_resource WHERE tenant_id = ?`, tenantID,
	).Scan(&traffic, &diskBytes)
	if err != nil {
		return nil, err
	}
	out[feedbackKindGitlabTraffic] = traffic
	out[feedbackKindGitlabDisk] = diskUsedGBFromBytes(int64(diskBytes + 0.5))

	var amount float64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(t.amount), 0)
		FROM billing_transaction t
		INNER JOIN billing_account a ON a.id = t.account_id
		WHERE a.tenant_id = ? AND t.transaction_type = 'consumption'`, tenantID,
	).Scan(&amount)
	if err != nil {
		return nil, err
	}
	out[feedbackKindConsumedAmount] = amount
	return out, nil
}
