package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// listenFor turns a test server address into the PC_LISTEN form the container uses.
func listenFor(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return ":" + port
}

func TestHealthcheckExitCodeFollowsHealthEndpoint(t *testing.T) {
	statusServer := func(status int) *httptest.Server {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("200 means healthy", func(t *testing.T) {
		if code := healthcheck(listenFor(t, statusServer(http.StatusOK))); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	})

	t.Run("non-200 means unhealthy", func(t *testing.T) {
		if code := healthcheck(listenFor(t, statusServer(http.StatusServiceUnavailable))); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
	})

	t.Run("only /api/health counts", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)
		if code := healthcheck(listenFor(t, srv)); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
	})

	t.Run("unreachable means unhealthy", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		listen := listenFor(t, srv)
		srv.Close()
		if code := healthcheck(listen); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
	})

	t.Run("host part of PC_LISTEN is ignored, loopback is dialled", func(t *testing.T) {
		listen := "0.0.0.0" + listenFor(t, statusServer(http.StatusOK))
		if code := healthcheck(listen); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	})
}
