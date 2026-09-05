package main

import (
	"context"
	"tracelog"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/domain"
	"taskEvents/eventbin"
)

// welcomeStub acknowledges USER_CREATED for the welcome-email intent slot (delivery TBD).
type welcomeStub struct{}

func (welcomeStub) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if cmd.EventType != "USER_CREATED" {
		return domain.DispatchPermanent, nil
	}
	return domain.DispatchSuccess, nil
}

func main() {
	if _, _, _, err := config.LoadEvent("user_created"); err != nil {
		tracelog.Fatal("user_created", "config", err)
	}
	eventbin.RunIntent("user_created", "1_send_welcome_email", welcomeStub{}, consumer.IdempotencyKeyFromEnvelope)
}
