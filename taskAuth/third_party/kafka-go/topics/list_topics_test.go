package topics

import (
	"context"
	"errors"
	"io"
	"net"
	"regexp"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	ktesting "github.com/segmentio/kafka-go/testing"
)

// brokerUnavailable reports whether err indicates the local Kafka test broker
// was not reachable or not yet ready. The nightly sweep boots the broker and
// the test battery concurrently, so a broker that is still starting surfaces
// as "Broker Not Available" or a connection-level error. Integration tests
// skip in that case instead of failing the whole run — the failure is
// environmental, not a regression in topic listing.
func brokerUnavailable(err error) bool {
	if err == nil {
		return false
	}
	// kafka.Error satisfies net.Error (Timeout/Temporary), so it must be
	// judged by its code before falling through to the connection-level
	// checks — otherwise e.g. TopicAlreadyExists would be treated as a
	// broker outage.
	var kerr kafka.Error
	if errors.As(err, &kerr) {
		switch kerr {
		case kafka.BrokerNotAvailable, kafka.LeaderNotAvailable, kafka.NotController:
			return true
		}
		return false
	}
	var nerr net.Error
	if errors.As(err, &nerr) {
		return true
	}
	return errors.Is(err, io.EOF)
}

func TestListReNil(t *testing.T) {
	_, err := ListRe(context.Background(), nil, nil)
	if err == nil {
		t.Fatal(err)
	}
}

func TestListRe(t *testing.T) {
	client, shutdown, err := newLocalClientWithTopic("TestTopics-A", 1)
	if err != nil {
		if brokerUnavailable(err) {
			t.Skipf("local Kafka broker not available: %v", err)
		}
		t.Fatalf("create topic: %v", err)
	}
	defer shutdown()
	clientCreateTopic(client, "TestTopics-B", 1)

	allRegex := regexp.MustCompile("TestTopics-.*")
	fooRegex := regexp.MustCompile("TestTopics-B")

	// Get all the topics
	topics, err := ListRe(context.Background(), client, allRegex)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 2 {
		t.Error("the wrong number of topics were returned. ", len(topics))
	}

	// Get one topic
	topics, err = ListRe(context.Background(), client, fooRegex)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 {
		t.Error("the wrong number of topics were returned. ", len(topics))
	}
}

func newLocalClientWithTopic(topic string, partitions int) (*kafka.Client, func(), error) {
	client, shutdown := newLocalClient()
	if err := clientCreateTopic(client, topic, partitions); err != nil {
		shutdown()
		return nil, nil, err
	}
	return client, func() {
		client.DeleteTopics(context.Background(), &kafka.DeleteTopicsRequest{
			Topics: []string{topic},
		})
		shutdown()
	}, nil
}

func clientCreateTopic(client *kafka.Client, topic string, partitions int) error {
	_, err := client.CreateTopics(context.Background(), &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		}},
	})
	if err != nil {
		return err
	}

	// Topic creation seems to be asynchronous. Metadata for the topic partition
	// layout in the cluster is available in the controller before being synced
	// with the other brokers, which causes "Error:[3] Unknown Topic Or Partition"
	// when sending requests to the partition leaders.
	//
	// This loop will wait up to 2 seconds polling the cluster until no errors
	// are returned.
	for i := 0; i < 20; i++ {
		r, err := client.Fetch(context.Background(), &kafka.FetchRequest{
			Topic:     topic,
			Partition: 0,
			Offset:    0,
		})
		if err == nil && r.Error == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func newLocalClient() (*kafka.Client, func()) {
	return newClient(kafka.TCP("localhost"))
}

func newClient(addr net.Addr) (*kafka.Client, func()) {
	conns := &ktesting.ConnWaitGroup{
		DialFunc: (&net.Dialer{}).DialContext,
	}

	transport := &kafka.Transport{
		Dial:     conns.Dial,
		Resolver: kafka.NewBrokerResolver(nil),
	}

	client := &kafka.Client{
		Addr:      addr,
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	return client, func() { transport.CloseIdleConnections(); conns.Wait() }
}
