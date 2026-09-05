package main

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// seedOrderForNumber 预置一条占用指定订单号的订单，用于模拟唯一键冲突。
func seedOrderForNumber(t *testing.T, tenantID int64, orderNumber string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
		VALUES (?, ?, ?, 'pending', 100, ?)`,
		generateSnowflakeID(), tenantID, orderNumber, utcNow(),
	)
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
}

// TestIsDuplicateKeyError 覆盖 MySQL 1062 与 SQLite 两种唯一键冲突文案
// 回归：此前仅匹配 SQLite "UNIQUE constraint failed"，MySQL 迁移后重试永不触发
// （生产故障：Duplicate entry 'ORD-20260807-001' for key 'billing_resource_order.order_number'）
func TestIsDuplicateKeyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "mysql1062",
			err:  errors.New("Error 1062 (23000): Duplicate entry 'ORD-20260807-001' for key 'billing_resource_order.order_number'"),
			want: true,
		},
		{
			name: "sqlite",
			err:  errors.New("UNIQUE constraint failed: billing_resource_order.order_number"),
			want: true,
		},
		{"other_error", errors.New("connection refused"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := isDuplicateKeyError(c.err); got != c.want {
			t.Errorf("%s: isDuplicateKeyError(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}

// TestCreateOrderRetryOnDuplicateOrderNumber 确定性复现 UNIQUE 撞号：
// generateOrderNumberFn 首轮返回已被占用的订单号（模拟注入冲突），
// createOrder 必须识别 MySQL 1062、换新 Snowflake 派生号码并成功，而非直接 400。
func TestCreateOrderRetryOnDuplicateOrderNumber(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = 9000000001
	const taken = "ORD-20990101-001"
	seedOrderForNumber(t, tenantID, taken)

	orig := generateOrderNumberFn
	defer func() { generateOrderNumberFn = orig }()
	calls := 0
	generateOrderNumberFn = func(tenantID, orderID int64) (string, error) {
		calls++
		if calls == 1 {
			return taken, nil // 首轮返回被占用号码 → INSERT 触发 MySQL 1062
		}
		return generateOrderNumber(tenantID, orderID) // 重试轮按新 id 派生号码
	}

	order, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("createOrder should retry and succeed, got: %v", err)
	}
	if calls < 2 {
		t.Fatalf("expected retry on duplicate, generateOrderNumberFn calls=%d", calls)
	}
	if order.OrderNumber == taken {
		t.Fatalf("order number should differ from collided one: %s", order.OrderNumber)
	}
}

// TestInsertOrderWithRetryCollision 回归 OPT-20260808-008：insertOrderWithRetry 首轮撞号
// 必须重新生成订单号并重试成功，而非直接把 1062 抛给调用方。
func TestInsertOrderWithRetryCollision(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = 9000000003
	const taken = "ORD-20990101-888"
	seedOrderForNumber(t, tenantID, taken)

	orig := generateOrderNumberFn
	defer func() { generateOrderNumberFn = orig }()
	calls := 0
	generateOrderNumberFn = func(tenantID, orderID int64) (string, error) {
		calls++
		if calls == 1 {
			return taken, nil // 首轮返回被占用号码 → INSERT 触发 MySQL 1062
		}
		return "ORD-20990101-889", nil
	}

	oid, onum, err := insertOrderWithRetry(context.Background(), db.Exec, tenantID, maxOrderInsertRetries, func(orderID int64, orderNumber string) (string, []any) {
		return `INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
			VALUES (?, ?, ?, 'pending', 100, ?)`,
			[]any{orderID, tenantID, orderNumber, utcNow()}
	})
	if err != nil {
		t.Fatalf("insertOrderWithRetry should retry and succeed, got: %v", err)
	}
	if calls < 2 {
		t.Fatalf("expected retry on duplicate, generateOrderNumberFn calls=%d", calls)
	}
	if onum != "ORD-20990101-889" {
		t.Fatalf("order_number = %s, want ORD-20990101-889", onum)
	}
	var got string
	if err := db.QueryRow(`SELECT order_number FROM billing_resource_order WHERE id = ?`, oid).Scan(&got); err != nil {
		t.Fatalf("query persisted order: %v", err)
	}
	if got != "ORD-20990101-889" {
		t.Fatalf("persisted order_number = %s, want ORD-20990101-889", got)
	}
}

// TestAdminGrantResourcesRetryOnDuplicateOrderNumber 回归 OPT-20260808-008：
// 管理端补建订单（adminGrantResources）在订单号撞号时必须重试收敛，
// 而非直接 1062 失败（与用户下单并发窗口内叠加场景）。
func TestAdminGrantResourcesRetryOnDuplicateOrderNumber(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = 9000000004
	const taken = "ORD-20990101-777"
	seedOrderForNumber(t, tenantID, taken)

	orig := generateOrderNumberFn
	defer func() { generateOrderNumberFn = orig }()
	calls := 0
	generateOrderNumberFn = func(tenantID, orderID int64) (string, error) {
		calls++
		if calls == 1 {
			return taken, nil // 首轮返回被占用号码 → INSERT 触发 MySQL 1062
		}
		return generateOrderNumber(tenantID, orderID)
	}

	result, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1, Reason: "测试赠送"},
	}, "user-1", "ik-retry-001")
	if err != nil {
		t.Fatalf("adminGrantResources should retry and succeed, got: %v", err)
	}
	if calls < 2 {
		t.Fatalf("expected retry on duplicate, generateOrderNumberFn calls=%d", calls)
	}
	orderNumber, _ := result["order_number"].(string)
	if orderNumber == taken {
		t.Fatalf("order number should differ from collided one: %s", orderNumber)
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_resource_order WHERE tenant_id = ? AND payment_method = 'admin_grant'`, tenantID).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("admin_grant orders count = %d, want 1", cnt)
	}
}

// TestCreateOrderConcurrentUniqueNumbers 并发创建订单（4 个租户同时下单），
// 断言全部成功且订单号唯一（Snowflake 派生，不再依赖日序号重试配额）。
func TestCreateOrderConcurrentUniqueNumbers(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const racers = 4
	var wg sync.WaitGroup
	orders := make([]*ResourceOrder, racers)
	errs := make([]error, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			o, _, err := createOrder(context.Background(), int64(9100000000+i), []orderItemInput{
				{ResourceType: ResourceTypeTaskPost, Quantity: 1},
			})
			orders[i], errs[i] = o, err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("racer %d createOrder: %v", i, err)
		}
	}
	seen := map[string]bool{}
	for i, o := range orders {
		if o == nil || o.OrderNumber == "" {
			t.Fatalf("racer %d returned nil order", i)
		}
		if seen[o.OrderNumber] {
			t.Fatalf("duplicate order_number generated: %s", o.OrderNumber)
		}
		seen[o.OrderNumber] = true
	}
}
