package workspace

import (
	"context"
	"testing"

	"github.com/erisristemena/relay/internal/config"
)

func TestService_SavePreferencesKeepsValidValuesOnInvalidInput(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()

	invalidPort := 70000
	unsupportedAppearance := "sunrise"
	snapshot, err := service.SavePreferences(ctx, PreferencesInput{
		PreferredPort:     &invalidPort,
		AppearanceVariant: &unsupportedAppearance,
	})
	if err != nil {
		t.Fatalf("SavePreferences() error = %v", err)
	}

	if snapshot.Preferences.PreferredPort != config.DefaultPort {
		t.Fatalf("snapshot.Preferences.PreferredPort = %d, want %d", snapshot.Preferences.PreferredPort, config.DefaultPort)
	}
	if snapshot.Preferences.AppearanceVariant != config.DefaultAppearanceVariant {
		t.Fatalf("snapshot.Preferences.AppearanceVariant = %q, want %q", snapshot.Preferences.AppearanceVariant, config.DefaultAppearanceVariant)
	}
	if len(snapshot.Warnings) == 0 {
		t.Fatalf("snapshot.Warnings = %v, want warning messages", snapshot.Warnings)
	}
}

func TestService_SavePreferencesPersistsProjectRootAndOpenRouterKey(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	projectRoot := t.TempDir()
	apiKey := "or-test-key"

	snapshot, err := service.SavePreferences(ctx, PreferencesInput{
		ProjectRoot:      &projectRoot,
		OpenRouterAPIKey: &apiKey,
	})
	if err != nil {
		t.Fatalf("SavePreferences() error = %v", err)
	}
	if snapshot.Preferences.ProjectRoot != projectRoot {
		t.Fatalf("snapshot.Preferences.ProjectRoot = %q, want %q", snapshot.Preferences.ProjectRoot, projectRoot)
	}
	if !snapshot.Preferences.ProjectRootConfigured || !snapshot.Preferences.ProjectRootValid {
		t.Fatalf("project root status = %+v, want configured valid", snapshot.Preferences)
	}
	if !snapshot.Preferences.OpenRouterConfigured {
		t.Fatal("snapshot.Preferences.OpenRouterConfigured = false, want true")
	}

	loaded, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if loaded.ProjectRoot != projectRoot {
		t.Fatalf("loaded.ProjectRoot = %q, want %q", loaded.ProjectRoot, projectRoot)
	}
	if loaded.OpenRouter.APIKey != apiKey {
		t.Fatalf("loaded.OpenRouter.APIKey = %q, want saved key", loaded.OpenRouter.APIKey)
	}
}
