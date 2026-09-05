package sasl_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	ktesting "github.com/segmentio/kafka-go/testing"
)

const (
	saslTestConnect = "localhost:9093" // connect to sasl listener
	saslTestTopic   = "test-writer-0"  // this topic is guaranteed to exist.
)

// brokerSupportsSASL reports whether the test broker exposes a SASL listener
// on localhost:9093 (as configured in the repo's docker-compose). When the
// listener is plaintext-only, the broker rejects the SASL handshake with
// IllegalSaslState.
func brokerSupportsSASL(t *testing.T) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d := kafka.Dialer{
		SASLMechanism: plain.Mechanism{
			Username: "adminplain",
			Password: "admin-secret",
		},
	}
	conn, err := d.DialContext(ctx, "tcp", saslTestConnect)
	if err == nil {
		conn.Close()
		return true
	}
	return !errors.Is(err, kafka.IllegalSASLState)
}

func TestSASL(t *testing.T) {
	if !brokerSupportsSASL(t) {
		t.Skip("broker does not expose a SASL listener on localhost:9093")
	}

	tests := []struct {
		valid    func() sasl.Mechanism
		invalid  func() sasl.Mechanism
		minKafka string
	}{
		{
			valid: func() sasl.Mechanism {
				return plain.Mechanism{
					Username: "adminplain",
					Password: "admin-secret",
				}
			},
			invalid: func() sasl.Mechanism {
				return plain.Mechanism{
					Username: "adminplain",
					Password: "badpassword",
				}
			},
		},
		{
			valid: func() sasl.Mechanism {
				mech, _ := scram.Mechanism(scram.SHA256, "adminscram", "admin-secret-256")
				return mech
			},
			invalid: func() sasl.Mechanism {
				mech, _ := scram.Mechanism(scram.SHA256, "adminscram", "badpassword")
				return mech
			},
			minKafka: "0.10.2.0",
		},
		{
			valid: func() sasl.Mechanism {
				mech, _ := scram.Mechanism(scram.SHA512, "adminscram", "admin-secret-512")
				return mech
			},
			invalid: func() sasl.Mechanism {
				mech, _ := scram.Mechanism(scram.SHA512, "adminscram", "badpassword")
				return mech
			},
			minKafka: "0.10.2.0",
		},
	}

	for _, tt := range tests {
		mech := tt.valid()
		if !ktesting.KafkaIsAtLeast(tt.minKafka) {
			t.Skip("requires min kafka version " + tt.minKafka)
		}

		t.Run(mech.Name()+" success", func(t *testing.T) {
			testConnect(t, tt.valid(), true)
		})
		t.Run(mech.Name()+" failure", func(t *testing.T) {
			testConnect(t, tt.invalid(), false)
		})
		t.Run(mech.Name()+" is reusable", func(t *testing.T) {
			mech := tt.valid()
			testConnect(t, mech, true)
			testConnect(t, mech, true)
			testConnect(t, mech, true)
		})

	}
}

func testConnect(t *testing.T, mechanism sasl.Mechanism, success bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	d := kafka.Dialer{
		SASLMechanism: mechanism,
	}
	_, err := d.DialLeader(ctx, "tcp", saslTestConnect, saslTestTopic, 0)
	if success && err != nil {
		t.Errorf("should have logged in correctly, got err: %v", err)
	} else if !success && err == nil {
		t.Errorf("should not have logged in correctly")
	}
}
