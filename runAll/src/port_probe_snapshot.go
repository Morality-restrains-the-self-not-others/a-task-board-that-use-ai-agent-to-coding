package main

import (
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var listeningPortFromLsofLine = regexp.MustCompile(`:(\d+)\s+\(LISTEN\)`)

func parseListeningPortsFromLsof(output string) map[string]struct{} {
	ports := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		match := listeningPortFromLsofLine.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		port := strings.TrimSpace(match[1])
		if port == "" {
			continue
		}
		ports[port] = struct{}{}
	}
	return ports
}

func scanListeningTCPPorts() (map[string]struct{}, error) {
	cmd := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}
	return parseListeningPortsFromLsof(string(output)), nil
}

func serviceHasActivePortInSnapshot(svc *Service, ports map[string]struct{}) bool {
	if svc == nil || len(ports) == 0 {
		return false
	}
	for _, port := range resolveServicePorts(svc) {
		if _, ok := ports[port]; ok {
			return true
		}
	}
	return false
}

func (r *Runner) listeningTCPPortsSnapshot() map[string]struct{} {
	if r == nil {
		return map[string]struct{}{}
	}
	now := time.Now()
	r.listeningPortsSnapMu.RLock()
	if r.listeningPortsSnap != nil && now.Before(r.listeningPortsSnapExpires) {
		snap := r.listeningPortsSnap
		r.listeningPortsSnapMu.RUnlock()
		return snap
	}
	r.listeningPortsSnapMu.RUnlock()

	ports, err := scanListeningTCPPorts()
	if err != nil {
		log.Printf("[runAll] batch TCP port scan failed: %v", err)
		ports = map[string]struct{}{}
	}

	// Publish a fresh map; never mutate a previously returned snapshot in place
	// (callers may still be reading it — clear/write would race).
	r.listeningPortsSnapMu.Lock()
	r.listeningPortsSnap = ports
	r.listeningPortsSnapExpires = now.Add(portProbeCacheTTL)
	snap := ports
	r.listeningPortsSnapMu.Unlock()
	return snap
}

func (r *Runner) prefillStartablePortProbeCache(services []*ServiceStatus) {
	if r == nil || len(services) == 0 {
		return
	}
	startable := make([]*Service, 0)
	for _, st := range services {
		if st == nil || !shouldProbeListenPortForStatus(st.Status) {
			continue
		}
		if svc := r.findService(st.Name); svc != nil {
			startable = append(startable, svc)
		}
	}
	if len(startable) == 0 {
		return
	}

	ports := r.listeningTCPPortsSnapshot()
	now := time.Now()
	expires := now.Add(portProbeCacheTTL)

	r.portProbeCacheMu.Lock()
	if r.portProbeCache == nil {
		r.portProbeCache = make(map[string]portProbeCacheEntry)
	}
	for _, svc := range startable {
		r.portProbeCache[svc.Name] = portProbeCacheEntry{
			active:  serviceHasActivePortInSnapshot(svc, ports),
			expires: expires,
		}
	}
	r.portProbeCacheMu.Unlock()
}
