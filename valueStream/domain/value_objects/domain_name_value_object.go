package value_objects

import (
	"fmt"
	"strings"
)

// DomainName 是业务域名称值对象，创建后不可变。
type DomainName struct {
	value string
}

func NewDomainName(raw string) (DomainName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DomainName{}, fmt.Errorf("domain name cannot be empty")
	}
	return DomainName{value: trimmed}, nil
}

func (d DomainName) Value() string {
	return d.value
}

func (d DomainName) Equal(other DomainName) bool {
	return d.value == other.value
}

