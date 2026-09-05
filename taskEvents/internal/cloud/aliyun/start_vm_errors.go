package aliyun

import (
	"errors"
	"fmt"
	"strings"
)

// FormatStartVMError maps Aliyun SDK errors to user-facing messages (no silent retry).
func FormatStartVMError(err error, zoneID string) error {
	if err == nil {
		return nil
	}
	if noStock, ok := AsNoStockError(err); ok {
		return noStock
	}
	raw := err.Error()
	if strings.Contains(raw, "Zone.NotOnSale") {
		zone := strings.TrimSpace(zoneID)
		if zone == "" {
			zone = "（未指定）"
		}
		return fmt.Errorf(
			"指定可用区 %s 已停售或无可售资源，请更换可用区或地域后重试",
			zone,
		)
	}
	if strings.Contains(raw, "NotEnoughBalance") || strings.Contains(raw, "InvalidAccountStatus.NotEnoughBalance") {
		return fmt.Errorf("阿里云账户余额不足，无法创建按量付费实例，请购买后重试")
	}
	return err
}

// IsPermanentStartVMError reports errors that should not be retried by the consumer.
func IsPermanentStartVMError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := AsNoStockError(err); ok {
		return true
	}
	raw := err.Error()
	return strings.Contains(raw, "NotEnoughBalance") ||
		strings.Contains(raw, "InvalidAccountStatus.NotEnoughBalance") ||
		strings.Contains(raw, "IdempotentParameterMismatch") ||
		IsZoneNotOnSale(err)
}

// IsZoneNotOnSale reports whether err is Aliyun Zone.NotOnSale.
func IsZoneNotOnSale(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Zone.NotOnSale")
}

// ZoneNotOnSaleError is a typed wrapper for tests.
type ZoneNotOnSaleError struct {
	ZoneID string
}

func (e *ZoneNotOnSaleError) Error() string {
	zone := strings.TrimSpace(e.ZoneID)
	if zone == "" {
		zone = "（未指定）"
	}
	return fmt.Sprintf("指定可用区 %s 已停售或无可售资源，请更换可用区或地域后重试", zone)
}

// IsNoStockError reports whether err is Aliyun OperationDenied.NoStock.
func IsNoStockError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NoStock")
}

// NoStockError is a typed wrapper for user-facing no-stock messages.
type NoStockError struct {
	ZoneID       string
	InstanceType string
}

func (e *NoStockError) Error() string {
	zone := strings.TrimSpace(e.ZoneID)
	if zone == "" {
		zone = "（未指定）"
	}
	instanceType := strings.TrimSpace(e.InstanceType)
	if instanceType == "" {
		instanceType = "所选规格"
	}
	return fmt.Sprintf(
		"可用区 %s 中实例规格 %s 暂无库存，请更换实例规格或地域后重试",
		zone,
		instanceType,
	)
}

// AsNoStockError unwraps NoStockError when possible.
func AsNoStockError(err error) (*NoStockError, bool) {
	var n *NoStockError
	if errors.As(err, &n) {
		return n, true
	}
	if err != nil && strings.Contains(err.Error(), "暂无库存") {
		return &NoStockError{}, true
	}
	return nil, false
}

// AsZoneNotOnSaleError unwraps FormatStartVMError output when possible.
func AsZoneNotOnSaleError(err error) (*ZoneNotOnSaleError, bool) {
	var z *ZoneNotOnSaleError
	if errors.As(err, &z) {
		return z, true
	}
	if err != nil && strings.Contains(err.Error(), "已停售或无可售资源") {
		return &ZoneNotOnSaleError{}, true
	}
	return nil, false
}
