package tracelog

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

const (
	HeaderImpersonatorID           = "X-Impersonator-Id"
	HeaderImpersonationSessionID   = "X-Impersonation-Session-Id"
	HeaderImpersonating            = "X-Impersonating"
	headerImpersonatedUserFallback = "X-User-Id"
)

type impersonationCtxKey struct{}

// Impersonation identifies an admin-as-user session for logs and audit.
type Impersonation struct {
	ImpersonatorUserID string
	ImpersonatedUserID string
	SessionID          string
}

func (i Impersonation) Active() bool {
	return strings.TrimSpace(i.ImpersonatorUserID) != "" &&
		strings.TrimSpace(i.ImpersonatedUserID) != ""
}

type impersonationSlot struct {
	info Impersonation
}

func ContextWithImpersonation(ctx context.Context, info Impersonation) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if slot, ok := ctx.Value(impersonationCtxKey{}).(*impersonationSlot); ok && slot != nil {
		slot.info = info
		return ctx
	}
	return context.WithValue(ctx, impersonationCtxKey{}, &impersonationSlot{info: info})
}

// SetImpersonation mutates the request-scoped slot when Middleware installed it.
func SetImpersonation(ctx context.Context, info Impersonation) {
	if ctx == nil {
		return
	}
	if slot, ok := ctx.Value(impersonationCtxKey{}).(*impersonationSlot); ok && slot != nil {
		slot.info = info
	}
}

func ImpersonationFromContext(ctx context.Context) Impersonation {
	if ctx == nil {
		return Impersonation{}
	}
	slot, ok := ctx.Value(impersonationCtxKey{}).(*impersonationSlot)
	if !ok || slot == nil {
		return Impersonation{}
	}
	return slot.info
}

func ImpersonationFromRequest(r *http.Request) Impersonation {
	if r == nil {
		return Impersonation{}
	}
	actor := strings.TrimSpace(r.Header.Get(HeaderImpersonatorID))
	if actor == "" {
		return Impersonation{}
	}
	target := strings.TrimSpace(r.Header.Get(headerImpersonatedUserFallback))
	sid := strings.TrimSpace(r.Header.Get(HeaderImpersonationSessionID))
	return Impersonation{
		ImpersonatorUserID: actor,
		ImpersonatedUserID: target,
		SessionID:          sid,
	}
}

func AppendImpersonationLogArgs(args []any, ctx context.Context) []any {
	info := ImpersonationFromContext(ctx)
	if !info.Active() {
		return args
	}
	args = append(args,
		"impersonating", true,
		"impersonator_user_id", info.ImpersonatorUserID,
		"impersonated_user_id", info.ImpersonatedUserID,
	)
	if strings.TrimSpace(info.SessionID) != "" {
		args = append(args, "impersonation_session_id", info.SessionID)
	}
	return args
}

type impersonationHandler struct {
	slog.Handler
}

func (h impersonationHandler) Handle(ctx context.Context, r slog.Record) error {
	info := ImpersonationFromContext(ctx)
	if info.Active() {
		r.Add("impersonating", true)
		r.Add("impersonator_user_id", info.ImpersonatorUserID)
		r.Add("impersonated_user_id", info.ImpersonatedUserID)
		if strings.TrimSpace(info.SessionID) != "" {
			r.Add("impersonation_session_id", info.SessionID)
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h impersonationHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return impersonationHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h impersonationHandler) WithGroup(name string) slog.Handler {
	return impersonationHandler{Handler: h.Handler.WithGroup(name)}
}
