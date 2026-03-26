package workspace

import (
	"context"
	"strings"
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
	projectRoot := initWorkspaceRepositoryRoot(t)
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

func TestService_SavePreferencesProjectRootHandling(t *testing.T) {
	t.Parallel()

	validRoot := initWorkspaceRepositoryRoot(t)
	testCases := []struct {
		name                 string
		projectRoot          string
		wantProjectRoot      string
		wantConfigured       bool
		wantValid            bool
		wantMessageSubstring string
	}{
		{
			name:                 "trimmed valid git repository",
			projectRoot:          "  " + validRoot + "  ",
			wantProjectRoot:      validRoot,
			wantConfigured:       true,
			wantValid:            true,
			wantMessageSubstring: "",
		},
		{
			name:                 "relative repository path",
			projectRoot:          " relative/path ",
			wantProjectRoot:      "relative/path",
			wantConfigured:       true,
			wantValid:            false,
			wantMessageSubstring: "absolute path",
		},
		{
			name:                 "cleared project root",
			projectRoot:          "   ",
			wantProjectRoot:      "",
			wantConfigured:       false,
			wantValid:            false,
			wantMessageSubstring: "valid project_root",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			paths, store := newTestServiceStore(t)
			defer store.Close()

			service := NewService(store, paths)
			snapshot, err := service.SavePreferences(context.Background(), PreferencesInput{ProjectRoot: &testCase.projectRoot})
			if err != nil {
				t.Fatalf("SavePreferences() error = %v", err)
			}

			if snapshot.Preferences.ProjectRoot != testCase.wantProjectRoot {
				t.Fatalf("snapshot.Preferences.ProjectRoot = %q, want %q", snapshot.Preferences.ProjectRoot, testCase.wantProjectRoot)
			}
			if snapshot.Preferences.ProjectRootConfigured != testCase.wantConfigured || snapshot.Preferences.ProjectRootValid != testCase.wantValid {
				t.Fatalf("project root status = %+v, want configured=%t valid=%t", snapshot.Preferences, testCase.wantConfigured, testCase.wantValid)
			}
			if testCase.wantMessageSubstring == "" {
				if snapshot.Preferences.ProjectRootMessage != "" {
					t.Fatalf("snapshot.Preferences.ProjectRootMessage = %q, want empty", snapshot.Preferences.ProjectRootMessage)
				}
			} else if !strings.Contains(snapshot.Preferences.ProjectRootMessage, testCase.wantMessageSubstring) {
				t.Fatalf("snapshot.Preferences.ProjectRootMessage = %q, want substring %q", snapshot.Preferences.ProjectRootMessage, testCase.wantMessageSubstring)
			}

			loaded, _, err := config.Load(paths)
			if err != nil {
				t.Fatalf("config.Load() error = %v", err)
			}
			if loaded.ProjectRoot != testCase.wantProjectRoot {
				t.Fatalf("loaded.ProjectRoot = %q, want %q", loaded.ProjectRoot, testCase.wantProjectRoot)
			}
		})
	}
}
