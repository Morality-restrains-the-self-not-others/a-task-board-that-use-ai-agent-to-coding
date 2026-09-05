package tracelog

import (
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestListenAndServe_ShutdownCompletesInFlight(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	started := make(chan struct{})
	released := make(chan struct{})
	done := make(chan error, 1)
	stop := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-released
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		done <- listenAndServe(addr, mux, stop)
	}()

	getErr := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		var resp *http.Response
		var gerr error
		for time.Now().Before(deadline) {
			resp, gerr = http.Get("http://" + addr + "/slow")
			if gerr == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if gerr != nil {
			getErr <- gerr
			return
		}
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
		if resp.Header.Get(PidHeader) != strconv.Itoa(os.Getpid()) {
			getErr <- errString("pid header=" + resp.Header.Get(PidHeader))
			return
		}
		if resp.StatusCode != http.StatusOK {
			getErr <- errString("status")
			return
		}
		getErr <- nil
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not start")
	}
	close(stop)
	time.Sleep(30 * time.Millisecond)
	close(released)

	if err := <-done; err != nil {
		t.Fatalf("listenAndServe: %v", err)
	}
	if err := <-getErr; err != nil {
		t.Fatalf("GET /slow: %v", err)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
