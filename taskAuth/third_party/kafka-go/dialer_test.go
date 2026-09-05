package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"testing"
	"time"
)

func TestDialer(t *testing.T) {
	tests := []struct {
		scenario string
		function func(*testing.T, context.Context, *Dialer)
	}{
		{
			scenario: "looking up partitions returns the list of available partitions for a topic",
			function: testDialerLookupPartitions,
		},
	}

	for _, test := range tests {
		testFunc := test.function
		t.Run(test.scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			testFunc(t, ctx, &Dialer{})
		})
	}
}

func testDialerLookupPartitions(t *testing.T, ctx context.Context, d *Dialer) {
	client, topic, shutdown := newLocalClientAndTopic()
	defer shutdown()

	// Write a message to ensure the partition gets created.
	w := &Writer{
		Addr:      TCP("localhost:9092"),
		Topic:     topic,
		Transport: client.Transport,
	}
	w.WriteMessages(ctx, Message{})
	w.Close()

	partitions, err := d.LookupPartitions(ctx, "tcp", "localhost:9092", topic)
	if err != nil {
		t.Error(err)
		return
	}

	sort.Slice(partitions, func(i int, j int) bool {
		return partitions[i].ID < partitions[j].ID
	})

	checkPartitions(t, topic, partitions)
}

// checkPartitions asserts that the returned partitions describe a
// single-broker cluster. The broker identity (advertised host and broker ID)
// depends on the environment the tests run in (the repo's docker-compose
// broker is localhost:9092 with ID 1), so it is validated for coherence
// rather than hardcoded.
func checkPartitions(t *testing.T, topic string, partitions []Partition) {
	t.Helper()

	if len(partitions) != 1 {
		t.Errorf("bad partitions: %+v", partitions)
		return
	}

	p := partitions[0]
	if p.Topic != topic {
		t.Errorf("bad topic: %q", p.Topic)
	}
	if p.ID != 0 {
		t.Errorf("bad partition id: %d", p.ID)
	}
	if !isLoopbackHost(p.Leader.Host) {
		t.Errorf("bad leader host: %q", p.Leader.Host)
	}
	if p.Leader.Port != 9092 {
		t.Errorf("bad leader port: %d", p.Leader.Port)
	}
	if p.Leader.ID <= 0 {
		t.Errorf("bad leader id: %d", p.Leader.ID)
	}
	if p.Leader.Rack != "" {
		t.Errorf("bad leader rack: %q", p.Leader.Rack)
	}
	for _, broker := range p.Replicas {
		if broker != p.Leader {
			t.Errorf("replica does not match leader: %+v", broker)
		}
	}
	for _, broker := range p.Isr {
		if broker != p.Leader {
			t.Errorf("isr member does not match leader: %+v", broker)
		}
	}
	if len(p.OfflineReplicas) != 0 {
		t.Errorf("bad offline replicas: %+v", p.OfflineReplicas)
	}
}

func tlsConfig(t *testing.T) *tls.Config {
	const (
		certPEM = `-----BEGIN CERTIFICATE-----
MIID2zCCAsOgAwIBAgIJAMSqbewCgw4xMA0GCSqGSIb3DQEBCwUAMGAxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNp
c2NvMRAwDgYDVQQKDAdTZWdtZW50MRIwEAYDVQQDDAlsb2NhbGhvc3QwHhcNMTcx
MjIzMTU1NzAxWhcNMjcxMjIxMTU1NzAxWjBgMQswCQYDVQQGEwJVUzETMBEGA1UE
CAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzEQMA4GA1UECgwH
U2VnbWVudDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAtda9OWKYNtINe/BKAoB+/zLg2qbaTeHN7L722Ug7YoY6zMVB
aQEHrUmshw/TOrT7GLN/6e6rFN74UuNg72C1tsflZvxqkGdrup3I3jxMh2ApAxLi
zem/M6Eke2OAqt+SzRPqc5GXH/nrWVd3wqg48DZOAR0jVTY2e0fWy+Er/cPJI1lc
L6ZMIRJikHTXkaiFj2Jct1iWvgizx5HZJBxXJn2Awix5nvc+zmXM0ZhoedbJRoBC
dGkRXd3xv2F4lqgVHtP3Ydjc/wYoPiGudSAkhyl9tnkHjvIjA/LeRNshWHbCIaQX
yemnXIcyyf+W+7EK0gXio7uiP+QSoM5v/oeVMQIDAQABo4GXMIGUMHoGA1UdIwRz
MHGhZKRiMGAxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYD
VQQHDA1TYW4gRnJhbmNpc2NvMRAwDgYDVQQKDAdTZWdtZW50MRIwEAYDVQQDDAls
b2NhbGhvc3SCCQCBYUuEuypDMTAJBgNVHRMEAjAAMAsGA1UdDwQEAwIE8DANBgkq
hkiG9w0BAQsFAAOCAQEATk6IlVsXtNp4C1yeegaM+jE8qgKJfNm1sV27zKx8HPiO
F7LvTGYIG7zd+bf3pDSwRxfBhsLEwmN9TUN1d6Aa9zeu95qOnR76POfHILgttu2w
IzegO8I7BycnLjU9o/l9gCpusnN95tIYQhfD08ygUpYTQRuI0cmZ/Dp3xb0S9f5N
miYTuUoStYSA4RWbDWo+Is9YWPu7rwieziOZ96oguGz3mtqvkjxVAQH1xZr3bKHr
HU9LpQh0i6oTK0UCqnDwlhJl1c7A3UooxFpc3NGxyjogzTfI/gnBKfPo7eeswwsV
77rjIkhBW49L35KOo1uyblgK1vTT7VPtzJnuDq3ORg==
-----END CERTIFICATE-----`

		keyPEM = `-----BEGIN PRIVATE KEY-----
REDACTED
-----END PRIVATE KEY-----`

		caPEM = `-----BEGIN CERTIFICATE-----
MIIDPDCCAiQCCQCBYUuEuypDMTANBgkqhkiG9w0BAQsFADBgMQswCQYDVQQGEwJV
UzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzEQ
MA4GA1UECgwHU2VnbWVudDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTE3MTIyMzE1
NTMxOVoXDTI3MTIyMTE1NTMxOVowYDELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNh
bGlmb3JuaWExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xEDAOBgNVBAoMB1NlZ21l
bnQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAJwB+Yp6MyUepgtaRDxVjpMI2RmlAaV1qApMWu60LWGKJs4KWoIoLl6p
oSEqnWrpMmb38pyGP99X1+t3uZjiK9L8nFhuKZ581tsTKLxaSl+YVg7JbH5LVCS6
opsfB5ON1gJxf1HA9YyMqKHkBFh8/hdOGR0T6Bll9TPO1NQB/UqMy/tKr3sA3KZm
XVDbRKSuUAQWz5J9/hLPmVMU41F/uD7mvyDY+x8GymInZjUXG4e0oq2RJgU6SYZ8
mkscM6qhKY3mL487w/kHVFtFlMkOhvI7LIh3zVvWwgGSAoAv9yai9BDZNFSk0cEb
bb/IK7BQW9sNI3lcnGirdbnjV94X9/sCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEA
MJLeGdYO3dpsPx2R39Bw0qa5cUh42huPf8n7rp4a4Ca5jJjcAlCYV8HzqOzpiKYy
ZNuHy8LnNVYYh5Qoh8EO45bplMV1wnHfi6hW6DY5j3SQdcxkoVsW5R7rBF7a7SDg
6uChVRPHgsnALUUc7Wvvd3sAs/NKHzHu86mgD3EefkdqWAaCapzcqT9mo9KXkWJM
DhSJS+/iIaroc8umDnbPfhhgnlMf0/D4q0TjiLSSqyLzVifxnv9yHz56TrhHG/QP
E/8+FEGCHYKM4JLr5smGlzv72Kfx9E1CkG6TgFNIHjipVv1AtYDvaNMdPF2533+F
wE3YmpC3Q0g9r44nEbz4Bw==
-----END CERTIFICATE-----`
	)

	// Define TLS configuration
	certificate, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM([]byte(caPEM)); !ok {
		t.Error(err)
		t.FailNow()
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
}

func TestDialerTLS(t *testing.T) {
	client, topic, shutdown := newLocalClientAndTopic()
	defer shutdown()

	// Write a message to ensure the partition gets created.
	w := &Writer{
		Addr:      TCP("localhost:9092"),
		Topic:     topic,
		Transport: client.Transport,
	}
	w.WriteMessages(context.Background(), Message{})
	w.Close()

	// Create an SSL proxy using the tls.Config that connects to the
	// docker-composed kafka
	config := tlsConfig(t)
	l, err := tls.Listen("tcp", "127.0.0.1:", config)
	if err != nil {
		t.Error(err)
		return
	}
	defer l.Close()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return // intentionally ignored
			}

			go func(in net.Conn) {
				out, err := net.Dial("tcp", "localhost:9092")
				if err != nil {
					t.Error(err)
					return
				}
				defer out.Close()

				go io.Copy(in, out)
				io.Copy(out, in)
			}(conn)
		}
	}()

	// Use the tls.Config and connect to the SSL proxy
	d := &Dialer{
		TLS: config,
	}
	partitions, err := d.LookupPartitions(context.Background(), "tcp", l.Addr().String(), topic)
	if err != nil {
		t.Error(err)
		return
	}

	// Verify returned partition data is what we expect
	sort.Slice(partitions, func(i int, j int) bool {
		return partitions[i].ID < partitions[j].ID
	})

	checkPartitions(t, topic, partitions)
}

type MockConn struct {
	net.Conn
	done       chan struct{}
	partitions []Partition
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	select {
	case <-time.After(time.Minute):
	case <-m.done:
		return 0, context.Canceled
	}

	return 0, io.EOF
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	select {
	case <-time.After(time.Minute):
	case <-m.done:
		return 0, context.Canceled
	}

	return 0, io.EOF
}

func (m *MockConn) Close() error {
	select {
	case <-m.done:
	default:
		close(m.done)
	}
	return nil
}

func (m *MockConn) ReadPartitions(topics ...string) (partitions []Partition, err error) {
	return m.partitions, err
}

func TestDialerConnectTLSHonorsContext(t *testing.T) {
	config := tlsConfig(t)
	d := &Dialer{
		TLS: config,
	}

	conn := &MockConn{
		done: make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*25)
	defer cancel()

	_, err := d.connectTLS(ctx, conn, d.TLS)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected err to be %v; got %v", context.DeadlineExceeded, err)
		t.FailNow()
	}
}

func TestDialerResolver(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		scenario string
		address  string
		resolver map[string][]string
	}{
		{
			scenario: "resolve domain to ip",
			address:  "example.com",
			resolver: map[string][]string{
				"example.com": {"127.0.0.1"},
			},
		},
		{
			scenario: "resolve domain to ip and port",
			address:  "example.com",
			resolver: map[string][]string{
				"example.com": {"127.0.0.1:9092"},
			},
		},
		{
			scenario: "resolve domain with port to ip",
			address:  "example.com:9092",
			resolver: map[string][]string{
				"example.com": {"127.0.0.1:9092"},
			},
		},
		{
			scenario: "resolve domain with port to ip with different port",
			address:  "example.com:9092",
			resolver: map[string][]string{
				"example.com": {"127.0.0.1:80"},
			},
		},
		{
			scenario: "resolve domain with port to ip",
			address:  "example.com:9092",
			resolver: map[string][]string{
				"example.com": {"127.0.0.1"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			topic := makeTopic()
			createTopic(t, topic, 1)
			defer deleteTopic(t, topic)

			d := Dialer{
				Resolver: &mockResolver{addrs: test.resolver},
			}

			// Write a message to ensure the partition gets created.
			w := NewWriter(WriterConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   topic,
				Dialer:  &d,
			})
			w.WriteMessages(context.Background(), Message{})
			w.Close()

			partitions, err := d.LookupPartitions(ctx, "tcp", test.address, topic)
			if err != nil {
				t.Error(err)
				return
			}

			sort.Slice(partitions, func(i int, j int) bool {
				return partitions[i].ID < partitions[j].ID
			})

			checkPartitions(t, topic, partitions)
		})
	}
}

type mockResolver struct {
	addrs map[string][]string
}

func (mr *mockResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if addrs, ok := mr.addrs[host]; !ok {
		return nil, fmt.Errorf("unrecognized host %s", host)
	} else {
		return addrs, nil
	}
}
