package main

import (
	"context"
	"log"
	"strings"
)

func publishTenantGitLabOidcSsoEvent(ctx context.Context, eventType, companyID, clientID, actorUserID, redirectURI string) {
	data := map[string]interface{}{
		"company_id":    companyID,
		"client_id":     clientID,
		"actor_user_id": actorUserID,
	}
	if strings.TrimSpace(redirectURI) != "" {
		data["redirect_uri"] = redirectURI
	}
	if err := publishDomainEventKafka(ctx, eventType, data, companyID); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish %s failed: %v", eventType, err)
		}
	}
}
