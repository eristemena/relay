package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/erisristemena/relay/internal/repository"
	toml "github.com/pelletier/go-toml/v2"
)

const (
	DefaultPort              = 4747
	DefaultAppearanceVariant = "midnight"
	DefaultPlannerModel      = "anthropic/claude-opus-4"
	DefaultCoderModel        = "anthropic/claude-sonnet-4-5"
	DefaultReviewerModel     = "anthropic/claude-sonnet-4-5"
	DefaultTesterModel       = "deepseek/deepseek-chat"
	DefaultExplainerModel    = "google/gemini-2.0-flash-001"
)

var supportedAppearanceVariants = map[string]struct{}{
	"midnight": {},
	"graphite": {},
}

type Paths struct {
	HomeDir    string
	ConfigDir  string
	ConfigPath string
	Database   string
}

type Credential struct {
	Provider  string    `toml:"provider" json:"provider"`
	Label     string    `toml:"label,omitempty" json:"label,omitempty"`
	Secret    string    `toml:"secret" json:"-"`
	UpdatedAt time.Time `toml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type OpenRouter struct {
	APIKey    string    `toml:"api_key,omitempty" json:"-"`
	UpdatedAt time.Time `toml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type AgentModels struct {
	Planner   string `toml:"planner,omitempty" json:"planner,omitempty"`
	Coder     string `toml:"coder,omitempty" json:"coder,omitempty"`
	Reviewer  string `toml:"reviewer,omitempty" json:"reviewer,omitempty"`
	Tester    string `toml:"tester,omitempty" json:"tester,omitempty"`
	Explainer string `toml:"explainer,omitempty" json:"explainer,omitempty"`
}

type Config struct {
	Port               int          `toml:"port" json:"port"`
	OpenBrowserOnStart bool         `toml:"open_browser_on_start" json:"open_browser_on_start"`
	AppearanceVariant  string       `toml:"appearance_variant,omitempty" json:"appearance_variant,omitempty"`
	ProjectRoot        string       `toml:"project_root,omitempty" json:"project_root,omitempty"`
	LastSessionID      string       `toml:"last_session_id,omitempty" json:"last_session_id,omitempty"`
	OpenRouter         OpenRouter   `toml:"openrouter,omitempty" json:"openrouter,omitempty"`
	Agents             AgentModels  `toml:"agents,omitempty" json:"agents,omitempty"`
	Credentials        []Credential `toml:"credentials,omitempty" json:"credentials,omitempty"`
}

type SafePreferences struct {
	PreferredPort         int         `json:"preferred_port"`
	AppearanceVariant     string      `json:"appearance_variant"`
	HasCredentials        bool        `json:"has_credentials"`
	OpenRouterConfigured  bool        `json:"openrouter_configured"`
	ProjectRoot           string      `json:"project_root"`
	ProjectRootConfigured bool        `json:"project_root_configured"`
	ProjectRootValid      bool        `json:"project_root_valid"`
	ProjectRootMessage    string      `json:"project_root_message,omitempty"`
	AgentModels           AgentModels `json:"agent_models"`
	OpenBrowserOnStart    bool        `json:"open_browser_on_start"`
}

type ProjectRootStatus struct {
	Configured bool   `json:"configured"`
	Valid      bool   `json:"valid"`
	Message    string `json:"message,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Port:               DefaultPort,
		OpenBrowserOnStart: true,
		AppearanceVariant:  DefaultAppearanceVariant,
		Agents: AgentModels{
			Planner:   DefaultPlannerModel,
			Coder:     DefaultCoderModel,
			Reviewer:  DefaultReviewerModel,
			Tester:    DefaultTesterModel,
			Explainer: DefaultExplainerModel,
		},
	}
}

func EnsurePaths(homeOverride string) (Paths, error) {
	homeDir := homeOverride
	if homeDir == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve home directory: %w", err)
		}
		homeDir = userHome
	}

	paths := Paths{
		HomeDir:    homeDir,
		ConfigDir:  filepath.Join(homeDir, ".relay"),
		ConfigPath: filepath.Join(homeDir, ".relay", "config.toml"),
		Database:   filepath.Join(homeDir, ".relay", "relay.db"),
	}

	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		return Paths{}, fmt.Errorf("create relay config directory: %w", err)
	}

	if _, err := os.Stat(paths.ConfigPath); errors.Is(err, os.ErrNotExist) {
		if err := Save(paths, DefaultConfig()); err != nil {
			return Paths{}, err
		}
	} else if err != nil {
		return Paths{}, fmt.Errorf("stat relay config: %w", err)
	}

	return paths, nil
}

func Load(paths Paths) (Config, []string, error) {
	defaults := DefaultConfig()
	body, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		return defaults, nil, fmt.Errorf("read relay config: %w", err)
	}

	var decoded map[string]any
	if err := toml.Unmarshal(body, &decoded); err != nil {
		return defaults, nil, fmt.Errorf("parse relay config: %w", err)
	}

	cfg := defaults
	warnings := make([]string, 0)

	if rawPort, ok := decoded["port"]; ok {
		port, ok := parseInt(rawPort)
		if !ok || port < 1 || port > 65535 {
			warnings = append(warnings, "Ignored invalid preferred port; using the saved default instead.")
		} else {
			cfg.Port = port
		}
	}

	if rawOpenBrowser, ok := decoded["open_browser_on_start"]; ok {
		value, ok := rawOpenBrowser.(bool)
		if !ok {
			warnings = append(warnings, "Ignored invalid browser-start preference.")
		} else {
			cfg.OpenBrowserOnStart = value
		}
	}

	if rawAppearance, ok := decoded["appearance_variant"]; ok {
		value, ok := rawAppearance.(string)
		value = strings.TrimSpace(value)
		if !ok || value == "" {
			warnings = append(warnings, "Ignored invalid appearance variant.")
		} else if _, supported := supportedAppearanceVariants[value]; !supported {
			warnings = append(warnings, "Ignored unsupported appearance variant and kept the dark-mode default.")
		} else {
			cfg.AppearanceVariant = value
		}
	}

	if rawLastSession, ok := decoded["last_session_id"]; ok {
		value, ok := rawLastSession.(string)
		if ok {
			cfg.LastSessionID = strings.TrimSpace(value)
		}
	}

	if rawProjectRoot, ok := decoded["project_root"]; ok {
		value, ok := rawProjectRoot.(string)
		if !ok {
			warnings = append(warnings, "Ignored invalid project_root value.")
		} else {
			cfg.ProjectRoot = strings.TrimSpace(value)
		}
	}

	if rawOpenRouter, ok := decoded["openrouter"]; ok {
		openRouter, openRouterWarnings := parseOpenRouter(rawOpenRouter)
		cfg.OpenRouter = openRouter
		warnings = append(warnings, openRouterWarnings...)
	}

	if rawAgents, ok := decoded["agents"]; ok {
		agents, agentWarnings := parseAgentModels(rawAgents, cfg.Agents)
		cfg.Agents = agents
		warnings = append(warnings, agentWarnings...)
	}

	if rawCredentials, ok := decoded["credentials"]; ok {
		credentials, credentialWarnings := parseCredentials(rawCredentials)
		cfg.Credentials = credentials
		warnings = append(warnings, credentialWarnings...)
	}

	if strings.TrimSpace(cfg.OpenRouter.APIKey) == "" {
		for _, credential := range cfg.Credentials {
			if strings.EqualFold(strings.TrimSpace(credential.Provider), "openrouter") {
				cfg.OpenRouter.APIKey = credential.Secret
				if cfg.OpenRouter.UpdatedAt.IsZero() {
					cfg.OpenRouter.UpdatedAt = credential.UpdatedAt
				}
				break
			}
		}
	}

	return cfg, warnings, nil
}

func Save(paths Paths, cfg Config) error {
	body, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode relay config: %w", err)
	}

	if err := os.WriteFile(paths.ConfigPath, body, 0o600); err != nil {
		return fmt.Errorf("write relay config: %w", err)
	}

	return nil
}

func (cfg Config) SafePreferences() SafePreferences {
	projectRootStatus := cfg.ProjectRootState()
	return SafePreferences{
		PreferredPort:         cfg.Port,
		AppearanceVariant:     cfg.AppearanceVariant,
		HasCredentials:        len(cfg.Credentials) > 0 || cfg.HasOpenRouterKey(),
		OpenRouterConfigured:  cfg.HasOpenRouterKey(),
		ProjectRoot:           cfg.ProjectRoot,
		ProjectRootConfigured: projectRootStatus.Configured,
		ProjectRootValid:      projectRootStatus.Valid,
		ProjectRootMessage:    projectRootStatus.Message,
		AgentModels:           cfg.Agents.WithDefaults(),
		OpenBrowserOnStart:    cfg.OpenBrowserOnStart,
	}
}

func (cfg Config) HasOpenRouterKey() bool {
	return strings.TrimSpace(cfg.OpenRouter.APIKey) != ""
}

func ModelContextLimitHint(model string) *int {
	for _, token := range strings.FieldsFunc(strings.ToLower(strings.TrimSpace(model)), func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= '0' && r <= '9':
			return false
		default:
			return true
		}
	}) {
		if len(token) < 2 {
			continue
		}

		suffix := token[len(token)-1]
		multiplier := 0
		switch suffix {
		case 'k':
			multiplier = 1000
		case 'm':
			multiplier = 1000000
		default:
			continue
		}

		value, err := strconv.Atoi(token[:len(token)-1])
		if err != nil || value <= 0 {
			continue
		}

		contextLimit := value * multiplier
		return &contextLimit
	}

	return nil
}

func (cfg Config) ProjectRootState() ProjectRootStatus {
	status := repository.ValidateRoot(cfg.ProjectRoot)
	return ProjectRootStatus{Configured: status.Configured, Valid: status.Valid, Message: status.Message}
}

func (models AgentModels) WithDefaults() AgentModels {
	return AgentModels{
		Planner:   fallbackModel(models.Planner, DefaultPlannerModel),
		Coder:     fallbackModel(models.Coder, DefaultCoderModel),
		Reviewer:  fallbackModel(models.Reviewer, DefaultReviewerModel),
		Tester:    fallbackModel(models.Tester, DefaultTesterModel),
		Explainer: fallbackModel(models.Explainer, DefaultExplainerModel),
	}
}

func (cfg Config) RedactedJSON() string {
	type credentialView struct {
		Provider  string    `json:"provider"`
		Label     string    `json:"label,omitempty"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`
		Secret    string    `json:"secret"`
	}

	view := struct {
		Port               int              `json:"port"`
		OpenBrowserOnStart bool             `json:"open_browser_on_start"`
		AppearanceVariant  string           `json:"appearance_variant"`
		ProjectRoot        string           `json:"project_root,omitempty"`
		LastSessionID      string           `json:"last_session_id,omitempty"`
		OpenRouter         map[string]any   `json:"openrouter,omitempty"`
		Agents             AgentModels      `json:"agents"`
		Credentials        []credentialView `json:"credentials,omitempty"`
	}{
		Port:               cfg.Port,
		OpenBrowserOnStart: cfg.OpenBrowserOnStart,
		AppearanceVariant:  cfg.AppearanceVariant,
		ProjectRoot:        cfg.ProjectRoot,
		LastSessionID:      cfg.LastSessionID,
		OpenRouter: map[string]any{
			"configured": cfg.HasOpenRouterKey(),
		},
		Agents: cfg.Agents.WithDefaults(),
	}

	if cfg.HasOpenRouterKey() {
		view.OpenRouter["api_key"] = "[redacted]"
	}
	if !cfg.OpenRouter.UpdatedAt.IsZero() {
		view.OpenRouter["updated_at"] = cfg.OpenRouter.UpdatedAt
	}

	for _, credential := range cfg.Credentials {
		view.Credentials = append(view.Credentials, credentialView{
			Provider:  credential.Provider,
			Label:     credential.Label,
			UpdatedAt: credential.UpdatedAt,
			Secret:    "[redacted]",
		})
	}

	body, _ := json.Marshal(view)
	return string(body)
}

func parseOpenRouter(raw any) (OpenRouter, []string) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return OpenRouter{}, []string{"Ignored invalid openrouter block."}
	}

	openRouter := OpenRouter{}
	warnings := make([]string, 0)
	if rawAPIKey, ok := entry["api_key"]; ok {
		apiKey, ok := rawAPIKey.(string)
		if !ok {
			warnings = append(warnings, "Ignored invalid OpenRouter API key value.")
		} else {
			openRouter.APIKey = strings.TrimSpace(apiKey)
		}
	}
	if rawUpdatedAt, ok := entry["updated_at"]; ok {
		if updatedAt, ok := parseTime(rawUpdatedAt); ok {
			openRouter.UpdatedAt = updatedAt
		}
	}

	return openRouter, warnings
}

func parseAgentModels(raw any, defaults AgentModels) (AgentModels, []string) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return defaults.WithDefaults(), []string{"Ignored invalid agents block and kept the default role models."}
	}

	models := defaults.WithDefaults()
	warnings := make([]string, 0)
	for key, defaultValue := range map[string]string{
		"planner":   defaults.WithDefaults().Planner,
		"coder":     defaults.WithDefaults().Coder,
		"reviewer":  defaults.WithDefaults().Reviewer,
		"tester":    defaults.WithDefaults().Tester,
		"explainer": defaults.WithDefaults().Explainer,
	} {
		rawValue, exists := entry[key]
		if !exists {
			setAgentModel(&models, key, defaultValue)
			continue
		}

		value, ok := rawValue.(string)
		value = strings.TrimSpace(value)
		if !ok || value == "" {
			warnings = append(warnings, fmt.Sprintf("Ignored invalid %s model override and kept the default assignment.", key))
			setAgentModel(&models, key, defaultValue)
			continue
		}

		setAgentModel(&models, key, value)
	}

	return models, warnings
}

func parseCredentials(raw any) ([]Credential, []string) {
	items, ok := raw.([]any)
	if !ok {
		return nil, []string{"Ignored invalid credentials block."}
	}

	credentials := make([]Credential, 0, len(items))
	warnings := make([]string, 0)
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			warnings = append(warnings, "Ignored malformed credential entry.")
			continue
		}

		provider, _ := entry["provider"].(string)
		secret, _ := entry["secret"].(string)
		label, _ := entry["label"].(string)
		provider = strings.TrimSpace(provider)
		secret = strings.TrimSpace(secret)
		if provider == "" || secret == "" {
			warnings = append(warnings, "Ignored incomplete credential entry.")
			continue
		}

		credential := Credential{
			Provider: provider,
			Label:    strings.TrimSpace(label),
			Secret:   secret,
		}

		if rawUpdatedAt, ok := entry["updated_at"]; ok {
			if updatedAt, ok := parseTime(rawUpdatedAt); ok {
				credential.UpdatedAt = updatedAt
			}
		}

		credentials = append(credentials, credential)
	}

	return credentials, warnings
}

func setAgentModel(models *AgentModels, key string, value string) {
	switch key {
	case "planner":
		models.Planner = value
	case "coder":
		models.Coder = value
	case "reviewer":
		models.Reviewer = value
	case "tester":
		models.Tester = value
	case "explainer":
		models.Explainer = value
	}
}

func fallbackModel(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func parseInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func parseTime(raw any) (time.Time, bool) {
	switch value := raw.(type) {
	case time.Time:
		return value, true
	case string:
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
		if err != nil {
			return time.Time{}, false
		}
		return parsed, true
	default:
		return time.Time{}, false
	}
}
