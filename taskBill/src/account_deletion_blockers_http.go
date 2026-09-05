package main

import (
	"net/http"
	"strings"

	"tracelog"
)

func handleInternalTaskBillUsersRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/taskbill/users/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.HasSuffix(path, "account-deletion-blockers") {
		handleInternalUserAccountDeletionBlockers(w, r)
		return
	}
	if strings.HasSuffix(path, "personal-data") {
		handleInternalUserPersonalData(w, r)
		return
	}
	writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
}
