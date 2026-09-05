package domain_test

import (
	"context"
	"testing"

	"valueStream/domain/entities"
	"valueStream/domain/repositories"
	"valueStream/domain/services"
	"valueStream/domain/value_objects"
)

type inMemoryValueStreamRepository struct {
	streams    []entities.ValueStream
	lastOrder  []string
	savedNames []string
}

var _ repositories.ValueStreamRepository = (*inMemoryValueStreamRepository)(nil)

func (r *inMemoryValueStreamRepository) List(context.Context) ([]entities.ValueStream, error) {
	out := make([]entities.ValueStream, len(r.streams))
	copy(out, r.streams)
	return out, nil
}

func (r *inMemoryValueStreamRepository) SaveDomainOrder(_ context.Context, order value_objects.DomainOrder, streams []entities.ValueStream) error {
	domains := order.Domains()
	r.lastOrder = make([]string, len(domains))
	for i, d := range domains {
		r.lastOrder[i] = d.Value()
	}
	r.savedNames = make([]string, len(streams))
	for i, s := range streams {
		r.savedNames[i] = s.Name()
	}
	r.streams = append([]entities.ValueStream(nil), streams...)
	return nil
}

func TestDomainOrderReorderService_Reorder_RequiresRepository(t *testing.T) {
	svc := services.NewDomainOrderReorderService(nil)
	_, err := svc.Reorder(context.Background(), []string{"A"})
	if err == nil {
		t.Fatal("expected nil repository to return error")
	}
}

func TestDomainOrderReorderService_Reorder_PersistsReorderedStreams(t *testing.T) {
	repo := &inMemoryValueStreamRepository{
		streams: []entities.ValueStream{
			makeStream(t, "1", "flow-a1", "A"),
			makeStream(t, "2", "flow-b1", "B"),
			makeStream(t, "3", "flow-a2", "A"),
		},
	}
	svc := services.NewDomainOrderReorderService(repo)
	evt, err := svc.Reorder(context.Background(), []string{"B", "A"})
	if err != nil {
		t.Fatalf("Reorder returned error: %v", err)
	}
	if len(repo.lastOrder) != 2 || repo.lastOrder[0] != "B" || repo.lastOrder[1] != "A" {
		t.Fatalf("saved order = %v", repo.lastOrder)
	}
	wantNames := []string{"flow-b1", "flow-a1", "flow-a2"}
	for i := range wantNames {
		if repo.savedNames[i] != wantNames[i] {
			t.Fatalf("saved names = %v, want %v", repo.savedNames, wantNames)
		}
	}
	if len(evt.DomainNames) == 0 {
		t.Fatal("expected domain reordered event")
	}
}

