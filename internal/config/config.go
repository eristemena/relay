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

	toml "github.com/pelletier/go-toml/v2"
)

const (
	DefaultPort              = 4747
	DefaultAppearanceVariant = "midnight"
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

type Config struct {
	Port               int          `toml:"port" json:"port"`
	OpenBrowserOnStart bool         `toml:"open_browser_on_start" json:"open_browser_on_start"`
	AppearanceVariant  string       `toml:"appearance_variant,omitempty" json:"appearance_variant,omitempty"`
	LastSessionID      string       `toml:"last_session_id,omitempty" json:"last_session_id,omitempty"`
	Credentials        []Credential `toml:"credentials,omitempty" json:"credentials,omitempty"`
}

type SafePreferences struct {
	PreferredPort      int    `json:"preferred_port"`
	AppearanceVariant  string `json:"appearance_variant"`
	HasCredentials     bool   `json:"has_credentials"`
	OpenBrowserOnStart bool   `json:"open_browser_on_start"`
}

func DefaultConfig() Config {
	return Config{
		Port:               DefaultPort,
		OpenBrowserOnStart: true,
		AppearanceVariant:  DefaultAppearanceVariant,
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

	if rawCredentials, ok := decoded["credentials"]; ok {
		credentials, credentialWarnings := parseCredentials(rawCredentials)
		cfg.Credentials = credentials
		warnings = append(warnings, credentialWarnings...)
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
	return SafePreferences{
		PreferredPort:      cfg.Port,
		AppearanceVariant:  cfg.AppearanceVariant,
		HasCredentials:     len(cfg.Credentials) > 0,
		OpenBrowserOnStart: cfg.OpenBrowserOnStart,
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
		LastSessionID      string           `json:"last_session_id,omitempty"`
		Credentials        []credentialView `json:"credentials,omitempty"`
	}{
		Port:               cfg.Port,
		OpenBrowserOnStart: cfg.OpenBrowserOnStart,
		AppearanceVariant:  cfg.AppearanceVariant,
		LastSessionID:      cfg.LastSessionID,
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
