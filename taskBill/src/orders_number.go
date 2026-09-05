package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// generateOrderNumberFn 可注入，测试模拟 UNIQUE 撞号。
var generateOrderNumberFn = generateOrderNumber

// generateOrderNumber 生成 ORD-{UTC yyyyMMdd}-{tenantId}-{snowflakeId}（ADR-0018）。
// 末段等于主键 id；不读表、不争抢日序号（ADR-0017）。
func generateOrderNumber(tenantID, orderID int64) (string, error) {
	if tenantID <= 0 {
		return "", fmt.Errorf("订单号生成失败: 无效 tenantID")
	}
	if orderID <= 0 {
		return "", fmt.Errorf("订单号生成失败: 无效 orderID")
	}
	return fmt.Sprintf("ORD-%s-%d-%d", time.Now().UTC().Format("20060102"), tenantID, orderID), nil
}

// ParsedOrderNumber 是展示订单号的解析结果。
type ParsedOrderNumber struct {
	DateUTC   string
	TenantID  int64
	OrderID   int64
	HasTenant bool
}

// ParseResourceOrderNumber 解析展示号。
// 四段：ORD-日期-tenant-id；三段：ADR-0017 或存量 NNN（无租户基因）。
func ParseResourceOrderNumber(s string) (ParsedOrderNumber, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "-")
	if len(parts) < 3 || parts[0] != "ORD" {
		return ParsedOrderNumber{}, fmt.Errorf("非法订单号")
	}
	if len(parts[1]) != 8 {
		return ParsedOrderNumber{}, fmt.Errorf("非法订单号日期")
	}
	switch len(parts) {
	case 4:
		tid, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || tid <= 0 {
			return ParsedOrderNumber{}, fmt.Errorf("非法订单号租户")
		}
		oid, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil || oid <= 0 {
			return ParsedOrderNumber{}, fmt.Errorf("非法订单号主键")
		}
		return ParsedOrderNumber{DateUTC: parts[1], TenantID: tid, OrderID: oid, HasTenant: true}, nil
	case 3:
		oid, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || oid <= 0 {
			return ParsedOrderNumber{}, fmt.Errorf("非法订单号主键")
		}
		return ParsedOrderNumber{DateUTC: parts[1], OrderID: oid, HasTenant: false}, nil
	default:
		return ParsedOrderNumber{}, fmt.Errorf("非法订单号")
	}
}
