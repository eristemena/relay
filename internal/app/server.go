package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/erisristemena/relay/internal/browser"
	"github.com/erisristemena/relay/internal/config"
	frontendpkg "github.com/erisristemena/relay/internal/frontend"
	healthhandler "github.com/erisristemena/relay/internal/handlers/http"
	ws "github.com/erisristemena/relay/internal/handlers/ws"
	workspaceorchestrator "github.com/erisristemena/relay/internal/orchestrator/workspace"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

type Options struct {
	PreferredPort int
	DevMode       bool
	NoBrowser     bool
	ProjectRoot   string
	HomeDir       string
	Version       string
	Logger        *slog.Logger
	Launcher      browser.Launcher
}

type Server struct {
	options     Options
	logger      *slog.Logger
	status      *runtimeStatus
	httpServer  *http.Server
	store       *sqlite.Store
	listener    net.Listener
	service     *workspaceorchestrator.Service
	configPaths config.Paths
}

type runtimeStatus struct {
	mu      sync.RWMutex
	health  healthhandler.Response
	events  []ws.RuntimeEvent
	baseURL string
}

func Run(ctx context.Context, options Options) error {
	server, err := NewServer(ctx, options)
	if err != nil {
		return err
	}
	defer server.Close()

	return server.Run(ctx)
}

func NewServer(ctx context.Context, options Options) (*Server, error) {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if options.Launcher == nil {
		options.Launcher = browser.ExecLauncher{}
	}
	if options.Version == "" {
		options.Version = "dev"
	}

	paths, err := config.EnsurePaths(options.HomeDir)
	if err != nil {
		return nil, err
	}

	loadedConfig, warnings, err := config.Load(paths)
	if err != nil {
		return nil, err
	}
	if root := strings.TrimSpace(options.ProjectRoot); root != "" {
		loadedConfig.ProjectRoot = root
		if err := config.Save(paths, loadedConfig); err != nil {
			return nil, err
		}
	}

	preferredPort := options.PreferredPort
	if preferredPort == 0 {
		preferredPort = loadedConfig.Port
	}

	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		return nil, err
	}

	service := workspaceorchestrator.NewService(store, paths)
	service.SetLogger(logger)
	status := &runtimeStatus{
		health: healthhandler.Response{
			Status:        "starting",
			Version:       options.Version,
			FrontendMode:  "embedded",
			BrowserLaunch: "pending",
			Message:       "Relay is starting.",
		},
	}
	for _, warning := range warnings {
		status.appendEvent("config-warning", warning)
	}

	server := &Server{
		options:     options,
		logger:      logger,
		status:      status,
		store:       store,
		service:     service,
		configPaths: paths,
	}

	if err := server.buildHTTPServer(ctx, preferredPort, loadedConfig); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	s.logger.Info("relay workspace ready",
		"url", s.status.baseURL,
		"frontend_mode", s.StatusSnapshot().FrontendMode,
		"browser_launch", s.StatusSnapshot().BrowserLaunch,
	)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return fmt.Errorf("serve relay workspace: %w", err)
	}
}

func (s *Server) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Server) StatusSnapshot() healthhandler.Response {
	s.status.mu.RLock()
	defer s.status.mu.RUnlock()
	return s.status.health
}

func (s *Server) RuntimeEvents() []ws.RuntimeEvent {
	s.status.mu.RLock()
	defer s.status.mu.RUnlock()
	copyOfEvents := make([]ws.RuntimeEvent, len(s.status.events))
	copy(copyOfEvents, s.status.events)
	return copyOfEvents
}

func (s *Server) BaseURL() string {
	s.status.mu.RLock()
	defer s.status.mu.RUnlock()
	return s.status.baseURL
}

func (s *Server) buildHTTPServer(ctx context.Context, preferredPort int, loadedConfig config.Config) error {
	mux := http.NewServeMux()
	mux.Handle("/api/healthz", healthhandler.NewHandler(s))
	mux.Handle("/ws", ws.NewHandler(s.service, s, s.logger))

	frontendHandler, frontendMode := s.frontendHandler(ctx)
	mux.Handle("/", frontendHandler)

	listener, activePort, err := listen(preferredPort)
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", activePort)
	s.listener = listener
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.status.setReady(frontendMode, activePort, baseURL)
	if activePort != preferredPort {
		s.status.appendEvent("port-fallback", fmt.Sprintf("Relay preferred port %d was unavailable, so it started on %s instead.", preferredPort, baseURL))
	}

	browserAllowed := !s.options.NoBrowser && loadedConfig.OpenBrowserOnStart
	if !browserAllowed {
		s.status.setBrowserLaunch("skipped")
		s.status.appendEvent("browser-skipped", fmt.Sprintf("Relay is ready at %s. Automatic browser launch is disabled.", baseURL))
		return nil
	}

	launchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.options.Launcher.Open(launchCtx, baseURL); err != nil {
		s.status.setBrowserLaunch("failed")
		s.status.appendEvent("browser-launch-failed", fmt.Sprintf("Relay started at %s, but the browser could not be opened automatically. Open the address manually.", baseURL))
		s.logger.Warn("browser launch failed", "url", baseURL, "error", err)
		return nil
	}

	s.status.setBrowserLaunch("opened")
	return nil
}

func (s *Server) frontendHandler(ctx context.Context) (http.Handler, string) {
	if s.options.DevMode || strings.EqualFold(strings.TrimSpace(getenv("RELAY_DEV", "")), "true") {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		target, err := frontendpkg.DiscoverDevServer(probeCtx, 3000)
		if err != nil {
			message := "Relay could not reach the Next.js development server."
			s.status.appendEvent("frontend-unavailable", message+" Start the frontend with npm --prefix web run dev or use make dev.")
			return frontendpkg.NewUnavailableDevHandler(message), "proxied"
		}
		s.status.appendEvent("frontend-proxy-ready", fmt.Sprintf("Relay is proxying browser routes to %s.", target.String()))
		return frontendpkg.NewDevProxy(target), "proxied"
	}

	handler, err := frontendpkg.NewStaticHandler()
	if err != nil {
		message := "Relay could not load the embedded frontend assets."
		s.status.appendEvent("frontend-unavailable", message)
		return frontendpkg.NewUnavailableDevHandler(message), "embedded"
	}

	return handler, "embedded"
}

func listen(preferredPort int) (net.Listener, int, error) {
	if preferredPort > 0 {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", preferredPort))
		if err == nil {
			return listener, preferredPort, nil
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, fmt.Errorf("allocate relay listener: %w", err)
	}

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, 0, fmt.Errorf("read listener address: unexpected type %T", listener.Addr())
	}

	return listener, address.Port, nil
}

func (s *runtimeStatus) setReady(frontendMode string, activePort int, baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.Status = "ok"
	s.health.FrontendMode = frontendMode
	s.health.ActivePort = activePort
	s.health.Message = "Relay is ready."
	s.baseURL = baseURL
}

func (s *runtimeStatus) setBrowserLaunch(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.BrowserLaunch = status
}

func (s *runtimeStatus) appendEvent(phase string, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ws.RuntimeEvent{Phase: phase, Message: message})
	if message != "" {
		s.health.Message = message
	}
}

func getenv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
