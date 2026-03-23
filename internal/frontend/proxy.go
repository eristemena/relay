package frontend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxProbeResponseBytes = 256 * 1024

func DiscoverDevServer(ctx context.Context, preferredPort int) (*url.URL, error) {
	if value := strings.TrimSpace(os.Getenv("RELAY_FRONTEND_URL")); value != "" {
		target, err := url.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("parse RELAY_FRONTEND_URL: %w", err)
		}
		return target, nil
	}

	client := &http.Client{Timeout: 250 * time.Millisecond}
	ports := make([]int, 0, 12)
	ports = append(ports, preferredPort)
	for candidate := preferredPort + 1; candidate <= preferredPort+10; candidate++ {
		ports = append(ports, candidate)
	}

	for _, port := range ports {
		target, ok := probeRelayDevServer(ctx, client, port)
		if ok {
			return target, nil
		}
	}

	return nil, fmt.Errorf("no Next.js dev server was reachable on ports %d-%d", preferredPort, preferredPort+10)
}

func probeRelayDevServer(ctx context.Context, client *http.Client, port int) (*url.URL, bool) {
	target := fmt.Sprintf("http://127.0.0.1:%d", port)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, false
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, false
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusInternalServerError {
		return nil, false
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxProbeResponseBytes))
	if err != nil {
		return nil, false
	}

	if !looksLikeRelayFrontend(string(body)) {
		return nil, false
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, false
	}

	return parsed, true
}

func looksLikeRelayFrontend(body string) bool {
	return strings.Contains(body, "Relay - Local Workspace") ||
		strings.Contains(body, "Local AI session control, without leaving localhost.") ||
		strings.Contains(body, "Relay workspace")
}

func NewDevProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(responseWriter http.ResponseWriter, request *http.Request, err error) {
		http.Error(responseWriter, fmt.Sprintf("Relay could not reach the frontend dev server: %v", err), http.StatusBadGateway)
	}
	return proxy
}

func NewUnavailableDevHandler(message string) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
		responseWriter.WriteHeader(http.StatusServiceUnavailable)
		_, _ = responseWriter.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Relay frontend unavailable</title><style>:root{color-scheme:dark;background:#09090F;color:#FAFAFA;font-family:sans-serif}body{margin:0;min-height:100vh;display:grid;place-items:center;background:radial-gradient(circle at top,#111120,#09090F 55%)}main{max-width:42rem;border:1px solid #1A1A2E;background:#0D0D18;padding:2rem}code{font-family:ui-monospace,SFMono-Regular,monospace}</style></head><body><main><h1>Relay is running, but the frontend dev server is unavailable.</h1><p>` + message + `</p><p>Start the frontend with <code>npm --prefix web run dev</code> or use <code>make dev</code>.</p></main></body></html>`))
	})
}
