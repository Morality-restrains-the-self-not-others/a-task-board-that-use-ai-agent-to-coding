package events

import "time"

// DomainOrderReordered 表示业务域顺序已重排这一业务事实。
type DomainOrderReordered struct {
	OccurredAt  time.Time
	DomainNames []string
}

