package domain

import "strings"

// ServicePortRuntimeService answers whether a managed service still holds
// configured listen ports (detach/orphan processes).
type ServicePortRuntimeService struct {
	probe PortListenerProbeRepository
}

func NewServicePortRuntimeService(probe PortListenerProbeRepository) ServicePortRuntimeService {
	return ServicePortRuntimeService{probe: probe}
}

func (s ServicePortRuntimeService) HasActivePortListeners(ports []string) (bool, string, error) {
	if s.probe == nil {
		return false, "", nil
	}
	seen := make(map[string]struct{}, len(ports))
	for _, raw := range ports {
		port := strings.TrimSpace(raw)
		if port == "" {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}

		pids, err := s.probe.ListListeningPIDs(port)
		if err != nil {
			return false, port, err
		}
		if len(pids) > 0 {
			return true, port, nil
		}
	}
	return false, "", nil
}

// ServicePortStopEscalationService terminates listeners on service ports.
type ServicePortStopEscalationService struct {
	probe       PortListenerProbeRepository
	termination ForeignProcessTerminationRepository
}

func NewServicePortStopEscalationService(
	probe PortListenerProbeRepository,
	termination ForeignProcessTerminationRepository,
) ServicePortStopEscalationService {
	return ServicePortStopEscalationService{probe: probe, termination: termination}
}

func (s ServicePortStopEscalationService) ReleasePorts(ports []string) (string, []int, error) {
	if s.probe == nil || s.termination == nil {
		return "", nil, nil
	}
	seen := make(map[string]struct{}, len(ports))
	for _, raw := range ports {
		port := strings.TrimSpace(raw)
		if port == "" {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}

		pids, err := s.probe.ListListeningPIDs(port)
		if err != nil {
			return port, nil, err
		}
		if len(pids) == 0 {
			continue
		}
		if err := s.termination.Terminate(pids); err != nil {
			return port, pids, err
		}
		remaining, err := s.probe.ListListeningPIDs(port)
		if err != nil {
			return port, pids, err
		}
		if len(remaining) > 0 {
			return port, remaining, nil
		}
	}
	return "", nil, nil
}
