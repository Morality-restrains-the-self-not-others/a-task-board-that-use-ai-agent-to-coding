package main

import (
	"context"
	"net/http"
)

const maxLogsLines = 2000

type serviceStatusPayload struct {
	*ServiceStatus
	Hint             string `json:"hint,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	Buildable        bool   `json:"buildable"`
	Language         string `json:"language"`
	ListenPortActive bool   `json:"listen_port_active,omitempty"`
}

func registerUIHandlers(mux *http.ServeMux, store *StatusStore, runner *Runner, shutdownFunc context.CancelFunc, exitReason *runAllExitReason) {
	registerExecQueueHandler(mux, runner)
	registerMigratePendingHandler(mux, runner)
	registerStatusHandlers(mux, store, runner, shutdownFunc, exitReason)
	registerServiceHandlers(mux, runner)
	registerStartStopHandlers(mux, runner)
	registerRestartAllHandlers(mux, runner)
	registerPreciseRestartHandlers(mux, runner)
	registerTraeAgentPushHandlers(mux, runner)
	registerBuildHandlers(mux, runner)
	registerLogsHandlers(mux, runner)
	registerObservabilityHandlers(mux, runner)
	registerDevHandlers(mux, runner)
	registerProxyHandlers(mux, runner)
	registerIndexHandler(mux)
}
