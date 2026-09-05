package tracelog

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// PidHeader is set on every response so runAll canary restart can tell which
// process answered a health probe while two listeners share a port.
const PidHeader = "X-RunAll-Pid"

// DrainTimeout is how long Shutdown waits for in-flight requests (ADR-0058).
// Override in tests.
var DrainTimeout = 25 * time.Second

// ListenAndServe binds addr with SO_REUSEPORT (Linux) and serves handler until
// SIGINT/SIGTERM, then http.Server.Shutdown.
// Cite: https://pkg.go.dev/net/http#Server.Shutdown
// Cite: https://pkg.go.dev/net#ListenConfig
func ListenAndServe(addr string, handler http.Handler) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	stop := make(chan struct{})
	go func() {
		<-sig
		close(stop)
	}()
	return listenAndServe(addr, handler, stop)
}

// ListenAndServeContext is ListenAndServe but shuts down when ctx is cancelled
// (for processes that already own SIGTERM, e.g. taskEvents consumers).
func ListenAndServeContext(ctx context.Context, addr string, handler http.Handler) error {
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(stop)
		}
	}()
	return listenAndServe(addr, handler, stop)
}

func listenAndServe(addr string, handler http.Handler, stop <-chan struct{}) error {
	if handler == nil {
		handler = http.DefaultServeMux
	}
	handler = WithPidHeader(handler)

	lc := ReusePortListenConfig()
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: handler}
	errCh := make(chan error, 1)
	go func() {
		serveErr := srv.Serve(ln)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case serveErr := <-errCh:
		return serveErr
	case <-stop:
		slog.Info("http_graceful_shutdown", "event", "http_graceful_shutdown", "addr", addr)
		ctx, cancel := context.WithTimeout(context.Background(), DrainTimeout)
		defer cancel()
		if shutErr := srv.Shutdown(ctx); shutErr != nil {
			_ = srv.Close()
			return shutErr
		}
		return <-errCh
	}
}

// WithPidHeader injects X-RunAll-Pid on every response.
func WithPidHeader(next http.Handler) http.Handler {
	pid := strconv.Itoa(os.Getpid())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(PidHeader, pid)
		next.ServeHTTP(w, r)
	})
}

// ReusePortListenConfig returns a ListenConfig that sets SO_REUSEADDR and,
// on Linux, SO_REUSEPORT so a canary peer can bind the same port.
func ReusePortListenConfig() net.ListenConfig {
	return net.ListenConfig{Control: reusePortControl}
}
