package integration_test

import (
	"testing"

	"github.com/eristemena/relay/internal/app"
	"github.com/eristemena/relay/internal/config"
)

func TestWorkspacePreferences_SavePersistsToConfig(t *testing.T) {
	server, paths, _ := newIntegrationServer(t, app.Options{NoBrowser: true})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type": "preferences.save",
		"payload": map[string]any{
			"preferred_port": 4848,
			"appearance_variant": "graphite",
			"credentials": []map[string]any{{
				"provider": "openai",
				"label": "Personal",
				"secret": "shh",
			}},
		},
	})
	response := readUntilType(t, connection, "preferences.saved")
	payload := response["payload"].(map[string]any)
	preferences := payload["preferences"].(map[string]any)
	if int(preferences["preferred_port"].(float64)) != 4848 {
		t.Fatalf("preferred_port = %v, want 4848", preferences["preferred_port"])
	}
	if preferences["has_credentials"] != true {
		t.Fatalf("has_credentials = %v, want true", preferences["has_credentials"])
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.Port != 4848 {
		t.Fatalf("cfg.Port = %d, want 4848", cfg.Port)
	}
	if cfg.AppearanceVariant != "graphite" {
		t.Fatalf("cfg.AppearanceVariant = %q, want graphite", cfg.AppearanceVariant)
	}
	if len(cfg.Credentials) != 1 {
		t.Fatalf("len(cfg.Credentials) = %d, want 1", len(cfg.Credentials))
	}
}
