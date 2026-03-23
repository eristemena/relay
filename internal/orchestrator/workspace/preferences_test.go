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
