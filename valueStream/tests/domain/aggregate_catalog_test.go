package domain_test

import (
	"testing"

	"valueStream/domain/aggregates"
	"valueStream/domain/entities"
	"valueStream/domain/value_objects"
)

func makeStream(t *testing.T, id, name, domain string) entities.ValueStream {
	t.Helper()
	d, err := value_objects.NewDomainName(domain)
	if err != nil {
		t.Fatalf("NewDomainName: %v", err)
	}
	stream, err := entities.NewValueStream(id, name, d)
	if err != nil {
		t.Fatalf("NewValueStream: %v", err)
	}
	return stream
}

func TestValueStreamCatalog_ReorderDomains_StableWithEvent(t *testing.T) {
	streams := []entities.ValueStream{
		makeStream(t, "1", "flow-a1", "A"),
		makeStream(t, "2", "flow-b1", "B"),
		makeStream(t, "3", "flow-a2", "A"),
		makeStream(t, "4", "flow-c1", "C"),
	}
	catalog, err := aggregates.NewValueStreamCatalog(streams)
	if err != nil {
		t.Fatalf("NewValueStreamCatalog: %v", err)
	}
	order, err := value_objects.NewDomainOrder([]string{"C", "A"})
	if err != nil {
		t.Fatalf("NewDomainOrder: %v", err)
	}
	event, err := catalog.ReorderDomains(order)
	if err != nil {
		t.Fatalf("ReorderDomains: %v", err)
	}
	got := catalog.Streams()
	if len(got) != 4 {
		t.Fatalf("streams len = %d, want 4", len(got))
	}
	names := []string{got[0].Name(), got[1].Name(), got[2].Name(), got[3].Name()}
	want := []string{"flow-c1", "flow-a1", "flow-a2", "flow-b1"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order = %v, want %v", names, want)
		}
	}
	if len(event.DomainNames) != 3 || event.DomainNames[0] != "C" || event.DomainNames[1] != "A" || event.DomainNames[2] != "B" {
		t.Fatalf("event domains = %v", event.DomainNames)
	}
	if event.OccurredAt.IsZero() {
		t.Fatal("event occurred_at should be set")
	}
}

func TestValueStreamCatalog_ReorderDomains_UnknownDomainFails(t *testing.T) {
	streams := []entities.ValueStream{
		makeStream(t, "1", "flow-a1", "A"),
	}
	catalog, err := aggregates.NewValueStreamCatalog(streams)
	if err != nil {
		t.Fatalf("NewValueStreamCatalog: %v", err)
	}
	order, err := value_objects.NewDomainOrder([]string{"X"})
	if err != nil {
		t.Fatalf("NewDomainOrder: %v", err)
	}
	if _, err := catalog.ReorderDomains(order); err == nil {
		t.Fatal("expected unknown domain to fail")
	}
}

