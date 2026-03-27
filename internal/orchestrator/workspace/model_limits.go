package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/erisristemena/relay/internal/config"
)

const (
	defaultModelContextLimitTTL = 6 * time.Hour
	defaultOpenRouterModelsURL  = "https://openrouter.ai/api/v1/models"
)

type modelContextLimitResolver struct {
	client  *http.Client
	baseURL string
	ttl     time.Duration
	now     func() time.Time

	mu        sync.Mutex
	models    map[string]int
	expiresAt time.Time
}

func newModelContextLimitResolver() *modelContextLimitResolver {
	return &modelContextLimitResolver{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: defaultOpenRouterModelsURL,
		ttl:     defaultModelContextLimitTTL,
		now:     time.Now,
		models:  make(map[string]int),
	}
}

func (r *modelContextLimitResolver) warm(ctx context.Context, cfg config.Config) error {
	_, err := r.resolve(ctx, cfg, "", true)
	return err
}

func (r *modelContextLimitResolver) resolve(ctx context.Context, cfg config.Config, model string, refreshOnly bool) (*int, error) {
	model = strings.TrimSpace(model)
	if cached := r.cached(model); cached != nil || refreshOnly {
		if cached != nil || !r.needsRefresh() {
			if cached != nil {
				return cached, nil
			}
			return r.fallback(model), nil
		}
	}

	if cfg.HasOpenRouterKey() {
		limits, err := r.fetch(ctx, cfg.OpenRouter.APIKey)
		if err != nil {
			if cached := r.cached(model); cached != nil {
				return cached, nil
			}
			return r.fallback(model), err
		}
		r.mu.Lock()
		r.models = limits
		r.expiresAt = r.now().UTC().Add(r.ttl)
		r.mu.Unlock()
	}

	if refreshOnly {
		return nil, nil
	}

	if cached := r.cached(model); cached != nil {
		return cached, nil
	}
	return r.fallback(model), nil
}

func (r *modelContextLimitResolver) cached(model string) *int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.now().UTC().After(r.expiresAt) {
		return nil
	}
	limit, ok := r.models[model]
	if !ok || limit <= 0 {
		return nil
	}
	value := limit
	return &value
}

func (r *modelContextLimitResolver) needsRefresh() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.now().UTC().After(r.expiresAt)
}

func (r *modelContextLimitResolver) fallback(model string) *int {
	return config.ModelContextLimitHint(model)
}

func (r *modelContextLimitResolver) fetch(ctx context.Context, apiKey string) (map[string]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build model metadata request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request model metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request model metadata: unexpected status %s", resp.Status)
	}

	var payload struct {
		Data []struct {
			ID            string `json:"id"`
			ContextLength int    `json:"context_length"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode model metadata: %w", err)
	}

	limits := make(map[string]int, len(payload.Data))
	for _, model := range payload.Data {
		id := strings.TrimSpace(model.ID)
		if id == "" || model.ContextLength <= 0 {
			continue
		}
		limits[id] = model.ContextLength
	}
	return limits, nil
}