package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

const healthcheckTimeout = 3 * time.Second

// healthcheck probes the running server from inside the container: it always dials
// loopback on the port of PC_LISTEN, whatever interface the server is bound to, because
// the container's health probe runs in the server's own network namespace.
// Exit status: 0 on HTTP 200, 1 otherwise (Docker treats any non-zero as unhealthy).
func healthcheck(listen string) int {
	if listen == "" {
		listen = ":8080"
	}
	_, port, err := net.SplitHostPort(listen)
	if err != nil || port == "" {
		fmt.Fprintf(os.Stderr, "healthcheck: invalid PC_LISTEN %q: %v\n", listen, err)
		return 1
	}
	url := "http://" + net.JoinHostPort("127.0.0.1", port) + "/api/health"
	client := &http.Client{Timeout: healthcheckTimeout}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: %s returned %s\n", url, resp.Status)
		return 1
	}
	return 0
}
