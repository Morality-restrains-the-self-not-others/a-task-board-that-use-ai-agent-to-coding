package routing

import (
	"taskEvents/config"
	"taskEvents/domain"
)

// DispatchRouter builds routes that all POST to the domain dispatch endpoint.
func DispatchRouter(domainName string, events []string) domain.DomainEventRouter {
	routes := make(map[string]domain.EventRoute)
	path := config.DispatchPath(domainName)
	for _, e := range events {
		routes[e] = domain.EventRoute{EventType: e, Path: path}
	}
	return domain.DomainEventRouter{Domain: domainName, Routes: routes}
}
