package domain_test

import (
	"context"
	"errors"
	"testing"

	"valueStream/domain/repositories"
	"valueStream/domain/services"
)

type stubCommitChangeRepository struct {
	changedFiles []string
	err          error
}

var _ repositories.CommitChangeRepository = (*stubCommitChangeRepository)(nil)

func (s *stubCommitChangeRepository) ListHeadChangedFiles(context.Context) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]string(nil), s.changedFiles...), nil
}

func TestHeadCommitImpactService_Evaluate_MatchedTestFile(t *testing.T) {
	repo := &stubCommitChangeRepository{
		changedFiles: []string{"tests/smoke/test_checkout.py"},
	}
	svc := services.NewHeadCommitImpactService(repo)

	results, err := svc.Evaluate(context.Background(), []services.ImpactStream{
		{
			Name: "Checkout",
			Steps: []services.ImpactStep{
				{Name: "Smoke", TestFile: "tests/smoke/test_checkout.py"},
				{Name: "Regression", TestFile: "tests/regression/test_cart.py"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	stream := results[0]
	if stream.ImpactStatus != services.ImpactStatusKnown {
		t.Fatalf("impact status = %q, want known", stream.ImpactStatus)
	}
	if !stream.Impacted {
		t.Fatal("expected stream impacted")
	}
	if len(stream.ImpactedSteps) != 1 || stream.ImpactedSteps[0] != "Smoke" {
		t.Fatalf("impacted steps = %v, want [Smoke]", stream.ImpactedSteps)
	}
	if len(stream.Steps) != 2 || !stream.Steps[0].Impacted || stream.Steps[1].Impacted {
		t.Fatalf("step impacts = %+v", stream.Steps)
	}
}

func TestHeadCommitImpactService_Evaluate_NoMatchedTestFile(t *testing.T) {
	repo := &stubCommitChangeRepository{
		changedFiles: []string{"tests/smoke/test_profile.py"},
	}
	svc := services.NewHeadCommitImpactService(repo)

	results, err := svc.Evaluate(context.Background(), []services.ImpactStream{
		{
			Name: "Checkout",
			Steps: []services.ImpactStep{
				{Name: "Smoke", TestFile: "tests/smoke/test_checkout.py"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	stream := results[0]
	if stream.ImpactStatus != services.ImpactStatusKnown {
		t.Fatalf("impact status = %q, want known", stream.ImpactStatus)
	}
	if stream.Impacted {
		t.Fatal("expected stream not impacted")
	}
	if len(stream.ImpactedSteps) != 0 {
		t.Fatalf("impacted steps = %v, want empty", stream.ImpactedSteps)
	}
}

func TestHeadCommitImpactService_Evaluate_RepositoryErrorReturnsUnknown(t *testing.T) {
	repo := &stubCommitChangeRepository{err: errors.New("git show failed")}
	svc := services.NewHeadCommitImpactService(repo)

	results, err := svc.Evaluate(context.Background(), []services.ImpactStream{
		{
			Name: "Checkout",
			Steps: []services.ImpactStep{
				{Name: "Smoke", TestFile: "tests/smoke/test_checkout.py"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	stream := results[0]
	if stream.ImpactStatus != services.ImpactStatusUnknown {
		t.Fatalf("impact status = %q, want unknown", stream.ImpactStatus)
	}
	if stream.Impacted {
		t.Fatal("expected stream impacted false when unknown")
	}
	if len(stream.Steps) != 1 || stream.Steps[0].ImpactStatus != services.ImpactStatusUnknown {
		t.Fatalf("step impacts = %+v", stream.Steps)
	}
}

func TestHeadCommitImpactService_Evaluate_MatchesRelativeChangeWithAbsoluteTestFile(t *testing.T) {
	repo := &stubCommitChangeRepository{
		changedFiles: []string{"tests/smoke/test_checkout.py"},
	}
	svc := services.NewHeadCommitImpactService(repo)

	results, err := svc.Evaluate(context.Background(), []services.ImpactStream{
		{
			Name: "Checkout",
			Steps: []services.ImpactStep{
				{Name: "Smoke", TestFile: "/repo/tests/smoke/test_checkout.py"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	stream := results[0]
	if stream.ImpactStatus != services.ImpactStatusKnown {
		t.Fatalf("impact status = %q, want known", stream.ImpactStatus)
	}
	if !stream.Impacted {
		t.Fatal("expected stream impacted for absolute test_file")
	}
	if len(stream.ImpactedSteps) != 1 || stream.ImpactedSteps[0] != "Smoke" {
		t.Fatalf("impacted steps = %v, want [Smoke]", stream.ImpactedSteps)
	}
	if len(stream.Steps) != 1 || !stream.Steps[0].Impacted {
		t.Fatalf("step impacts = %+v, want impacted step", stream.Steps)
	}
}
