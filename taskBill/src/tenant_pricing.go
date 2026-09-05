package main

import (
	"database/sql"
	"errors"
	"log"
)

type BillingAccount struct {
	ID            int64
	TenantID      int64
	Balance       int64
	FrozenBalance int64
	TaskPostQuota int64
	CreatedAt     string
	UpdatedAt     string
}

// ErrAccountNotFound is returned by getBillingAccount when the tenant has no billing account.
var ErrAccountNotFound = errors.New("billing account not found")

// getBillingAccount 只读查询 BillingAccount。创建由 taskEvents 消费者（COMPANY_CREATED → 4_init_billing_account）负责。
func getBillingAccount(tenantID int64) (*BillingAccount, error) {
	var acc BillingAccount
	err := db.QueryRow(`
		SELECT id, tenant_id, balance, frozen_balance,
		       COALESCE(task_post_quota, 0), created_at, updated_at
		FROM billing_account WHERE tenant_id = ?`, tenantID).Scan(
		&acc.ID, &acc.TenantID, &acc.Balance, &acc.FrozenBalance,
		&acc.TaskPostQuota, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func getOrCreateBillingAccount(tenantID int64, forUpdate bool) (*BillingAccount, bool, error) {
	var acc BillingAccount
	err := db.QueryRow(`
		SELECT id, tenant_id, balance, frozen_balance,
		       COALESCE(task_post_quota, 0), created_at, updated_at
		FROM billing_account WHERE tenant_id = ?`, tenantID).Scan(
		&acc.ID, &acc.TenantID, &acc.Balance, &acc.FrozenBalance,
		&acc.TaskPostQuota, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err == nil {
		return &acc, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}

	id := generateSnowflakeID()
	now := utcNow()
	_, err = db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`,
		id, tenantID, now, now,
	)
	if err != nil {
		return nil, false, err
	}
	// 新租户默认创建 normal 会员记录
	if _, err := ensureMembership(tenantID); err != nil {
		// 会员创建失败不阻塞账户创建，仅记录日志
		log.Printf("[taskBill] ensureMembership failed for new tenant %d: %v", tenantID, err)
	}

	return getOrCreateBillingAccount(tenantID, forUpdate)
}
