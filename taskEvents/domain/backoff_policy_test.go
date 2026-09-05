package domain_test

import (
	"testing"

	"taskEvents/domain"
)

type stubMQ struct {
	connected bool
}

func (s stubMQ) Connected() bool { return s.connected }

func TestNewConnectionHealth(t *testing.T) {
	h := domain.NewConnectionHealth(stubMQ{connected: true}, domain.TransportRedis)
	if !h.Connected || h.Transport != domain.TransportRedis {
		t.Fatalf("unexpected health: %+v", h)
	}
	if h.ReadinessStatus() != "ok" {
		t.Fatalf("status = %q, want ok", h.ReadinessStatus())
	}

	h = domain.NewConnectionHealth(stubMQ{connected: false}, domain.TransportRedis)
	if h.ReadinessStatus() != "degraded" {
		t.Fatalf("status = %q, want degraded", h.ReadinessStatus())
	}
}

func TestNewConnectionHealthNilMQ(t *testing.T) {
	h := domain.NewConnectionHealth(nil, domain.TransportKafka)
	if h.Connected {
		t.Fatal("nil mq should be disconnected")
	}
	if h.ReadinessStatus() != "degraded" {
		t.Fatal("expected degraded")
	}
}
