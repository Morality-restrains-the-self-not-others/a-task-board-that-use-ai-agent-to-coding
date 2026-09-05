package value_objects

import "fmt"

// DomainOrder 描述业务域顺序，保证非空且无重复。
type DomainOrder struct {
	domains []DomainName
}

func NewDomainOrder(raw []string) (DomainOrder, error) {
	if len(raw) == 0 {
		return DomainOrder{}, fmt.Errorf("domain order is required")
	}
	seen := make(map[string]bool, len(raw))
	domains := make([]DomainName, 0, len(raw))
	for _, candidate := range raw {
		domain, err := NewDomainName(candidate)
		if err != nil {
			return DomainOrder{}, err
		}
		if seen[domain.Value()] {
			return DomainOrder{}, fmt.Errorf("duplicate domain %q in domain order", domain.Value())
		}
		seen[domain.Value()] = true
		domains = append(domains, domain)
	}
	return DomainOrder{domains: domains}, nil
}

func (o DomainOrder) Domains() []DomainName {
	out := make([]DomainName, len(o.domains))
	copy(out, o.domains)
	return out
}

