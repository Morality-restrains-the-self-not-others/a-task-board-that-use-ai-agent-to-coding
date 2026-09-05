package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/memberjoined"
)

func main() {
	h := &memberjoined.Handler{Client: memberjoined.NewDefaultGitIdentityClient()}
	eventbin.RunIntent("member_joined", "1_create_default_git_identity", h, consumer.IdempotencyKeyFromEnvelope)
}
