package domain

import "testing"

type stubProbe struct {
	byPort map[string][]int
}

func (s *stubProbe) ListListeningPIDs(port string) ([]int, error) {
	return s.byPort[port], nil
}

type stubTerminate struct {
	called [][]int
}

func (s *stubTerminate) Terminate(pids []int) error {
	s.called = append(s.called, append([]int(nil), pids...))
	return nil
}

func TestServicePortRuntimeService_HasActivePortListeners(t *testing.T) {
	probe := &stubProbe{byPort: map[string][]int{"18022": {99}}}
	svc := NewServicePortRuntimeService(probe)
	active, port, err := svc.HasActivePortListeners([]string{"18022", "18023"})
	if err != nil {
		t.Fatalf("HasActivePortListeners: %v", err)
	}
	if !active || port != "18022" {
		t.Fatalf("active=%v port=%q", active, port)
	}
}

func TestServicePortStopEscalationService_ReleasePorts(t *testing.T) {
	probe := &stubProbe{byPort: map[string][]int{"18022": {99}}}
	term := &stubTerminate{}
	esc := NewServicePortStopEscalationService(probe, term)
	port, remaining, err := esc.ReleasePorts([]string{"18022"})
	if err != nil {
		t.Fatalf("ReleasePorts: %v", err)
	}
	if port != "18022" || len(remaining) != 1 || remaining[0] != 99 {
		t.Fatalf("port=%q remaining=%v", port, remaining)
	}
	if len(term.called) != 1 {
		t.Fatalf("terminate calls = %d", len(term.called))
	}
	probe.byPort["18022"] = nil
	port, remaining, err = esc.ReleasePorts([]string{"18022"})
	if err != nil {
		t.Fatalf("ReleasePorts second: %v", err)
	}
	if port != "" || len(remaining) != 0 {
		t.Fatalf("port=%q remaining=%v", port, remaining)
	}
}
