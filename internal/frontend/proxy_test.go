package frontend

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscoverDevServerSkipsUnrelatedAppOnPreferredPort(t *testing.T) {
	firstListener, secondListener, startingPort := reserveSequentialPorts(t)

	unrelated := httptest.NewUnstartedServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte("<html><head><title>Different App</title></head><body>Another Next app</body></html>"))
	}))
	unrelated.Listener = firstListener
	unrelated.Start()
	defer unrelated.Close()

	relay := httptest.NewUnstartedServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte("<html><head><title>Relay - Local Workspace</title></head><body>Local AI session control, without leaving localhost.</body></html>"))
	}))
	relay.Listener = secondListener
	relay.Start()
	defer relay.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	target, err := DiscoverDevServer(ctx, startingPort)
	if err != nil {
		t.Fatalf("DiscoverDevServer() error = %v", err)
	}

	want := fmt.Sprintf("http://127.0.0.1:%d", startingPort+1)
	if target.String() != want {
		t.Fatalf("target.String() = %q, want %q", target.String(), want)
	}
}

func TestDiscoverDevServerWaitsForDelayedRelayFrontend(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	relay := httptest.NewUnstartedServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte("<html><head><title>Relay workspace</title></head><body>Local AI session control, without leaving localhost.</body></html>"))
	}))
	relay.Listener = listener

	go func() {
		time.Sleep(300 * time.Millisecond)
		relay.Start()
	}()
	defer relay.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	target, err := DiscoverDevServer(ctx, port)
	if err != nil {
		t.Fatalf("DiscoverDevServer() error = %v", err)
	}

	want := fmt.Sprintf("http://127.0.0.1:%d", port)
	if target.String() != want {
		t.Fatalf("target.String() = %q, want %q", target.String(), want)
	}
}

func reserveSequentialPorts(t *testing.T) (net.Listener, net.Listener, int) {
	t.Helper()

	for port := 38000; port < 39000; port++ {
		firstListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}

		secondListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port+1))
		if err != nil {
			_ = firstListener.Close()
			continue
		}

		return firstListener, secondListener, port
	}

	t.Fatal("could not reserve sequential test ports")
	return nil, nil, 0
}
