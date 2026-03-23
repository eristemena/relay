package http

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status        string `json:"status"`
	Version       string `json:"version,omitempty"`
	FrontendMode  string `json:"frontend_mode,omitempty"`
	ActivePort    int    `json:"active_port,omitempty"`
	BrowserLaunch string `json:"browser_launch,omitempty"`
	Message       string `json:"message,omitempty"`
}

type Provider interface {
	StatusSnapshot() Response
}

type Handler struct {
	provider Provider
}

func NewHandler(provider Provider) *Handler {
	return &Handler{provider: provider}
}

func (h *Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.Header().Set("Content-Type", "application/json")
	snapshot := h.provider.StatusSnapshot()
	statusCode := http.StatusOK
	if snapshot.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}
	responseWriter.WriteHeader(statusCode)
	_ = json.NewEncoder(responseWriter).Encode(snapshot)
}
