package updater

import (
	"testing"
)

func TestNewUpdateService(t *testing.T) {
	testCases := []struct {
		name        string
		config      UpdateServiceConfig
		expectError bool
	}{
		{
			name: "Valid config",
			config: UpdateServiceConfig{
				RepoURL: "https://github.com/owner/repo",
			},
			expectError: false,
		},
		{
			name: "Invalid repo URL",
			config: UpdateServiceConfig{
				RepoURL: "not-a-url",
			},
			expectError: true,
		},
		{
			name: "Invalid repo path",
			config: UpdateServiceConfig{
				RepoURL: "https://github.com/owner",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewUpdateService(tc.config)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
		})
	}
}

func TestUpdateService_Start(t *testing.T) {
	testCases := []struct {
		name           string
		config         UpdateServiceConfig
		checkOnlyCalls int
		checkAndDo     int
		expectError    bool
	}{
		{
			name: "NoCheck",
			config: UpdateServiceConfig{
				RepoURL:        "https://github.com/owner/repo",
				CheckOnStartup: NoCheck,
			},
		},
		{
			name: "CheckOnStartup",
			config: UpdateServiceConfig{
				RepoURL:        "https://github.com/owner/repo",
				CheckOnStartup: CheckOnStartup,
			},
			checkOnlyCalls: 1,
		},
		{
			name: "CheckAndUpdateOnStartup",
			config: UpdateServiceConfig{
				RepoURL:        "https://github.com/owner/repo",
				CheckOnStartup: CheckAndUpdateOnStartup,
			},
			checkAndDo: 1,
		},
		{
			name: "Unknown mode",
			config: UpdateServiceConfig{
				RepoURL:        "https://github.com/owner/repo",
				CheckOnStartup: 99, // Invalid mode
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var checkOnlyCalls, checkAndDoCalls int

			// Mock the check functions
			originalCheckOnly := CheckOnly
			CheckOnly = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool, releaseURLFormat string) error {
				checkOnlyCalls++
				return nil
			}
			defer func() { CheckOnly = originalCheckOnly }()

			originalCheckForUpdates := CheckForUpdates
			CheckForUpdates = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool, releaseURLFormat string) error {
				checkAndDoCalls++
				return nil
			}
			defer func() { CheckForUpdates = originalCheckForUpdates }()

			service, err := NewUpdateService(tc.config)
			if err != nil {
				// This check is for NewUpdateService, not Start
				if tc.name == "Invalid repo URL" || tc.name == "Invalid repo path" {
					if tc.expectError {
						return
					}
				}
				t.Fatalf("Unexpected error creating service: %v", err)
			}

			err = service.Start()
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			if checkOnlyCalls != tc.checkOnlyCalls {
				t.Errorf("Expected CheckOnly calls: %d, got: %d", tc.checkOnlyCalls, checkOnlyCalls)
			}
			if checkAndDoCalls != tc.checkAndDo {
				t.Errorf("Expected CheckForUpdates calls: %d, got: %d", tc.checkAndDo, checkAndDoCalls)
			}
		})
	}
}
