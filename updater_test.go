package updater

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

func TestDetermineChannel(t *testing.T) {
	testCases := []struct {
		name         string
		tagName      string
		isPreRelease bool
		expected     string
	}{
		{"Stable release", "v1.0.0", false, "stable"},
		{"Beta release", "v1.0.0-beta.1", true, "beta"},
		{"Alpha release", "v1.0.0-alpha.1", true, "alpha"},
		{"Pre-release without alpha/beta is beta", "v1.0.0-rc.1", true, "beta"},
		{"No 'v' prefix stable", "1.0.0", false, "stable"},
		{"No 'v' prefix beta", "1.0.0-beta.1", true, "beta"},
		{"No 'v' prefix alpha", "1.0.0-alpha.1", true, "alpha"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			channel := determineChannel(tc.tagName, tc.isPreRelease)
			if channel != tc.expected {
				t.Errorf("Expected channel %s, got %s", tc.expected, channel)
			}
		})
	}
}

func TestCheckForNewerVersion(t *testing.T) {
	type testCase struct {
		name                 string
		channel              string
		currentVersion       string
		forceSemVerPrefix    bool
		mockRelease          *Release
		expectUpdate         bool
		expectError          bool
		GetLatestReleaseFunc func(ctx context.Context, owner, repo, channel string) (*Release, error)
	}

	testCases := []testCase{
		{
			name:           "Stable: Newer version available (v-prefixed)",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease: &Release{
				TagName:    "v1.1.0",
				PreRelease: false,
			},
			expectUpdate: true,
		},
		{
			name:           "Stable: Same version (v-prefixed)",
			channel:        "stable",
			currentVersion: "1.1.0",
			mockRelease: &Release{
				TagName:    "v1.1.0",
				PreRelease: false,
			},
			expectUpdate: false,
		},
		{
			name:           "Stable: Older version (v-prefixed)",
			channel:        "stable",
			currentVersion: "1.2.0",
			mockRelease: &Release{
				TagName:    "v1.1.0",
				PreRelease: false,
			},
			expectUpdate: false,
		},
		{
			name:           "Beta: Newer beta version available (v-prefixed)",
			channel:        "beta",
			currentVersion: "1.0.0",
			mockRelease: &Release{
				TagName:    "v1.1.0-beta.1",
				PreRelease: true,
			},
			expectUpdate: true,
		},
		{
			name:           "Beta: Same beta version (v-prefixed)",
			channel:        "beta",
			currentVersion: "1.1.0-beta.1",
			mockRelease: &Release{
				TagName:    "v1.1.0-beta.1",
				PreRelease: true,
			},
			expectUpdate: false,
		},
		{
			name:           "Alpha: Newer alpha version available (v-prefixed)",
			channel:        "alpha",
			currentVersion: "1.0.0",
			mockRelease: &Release{
				TagName:    "v1.1.0-alpha.1",
				PreRelease: true,
			},
			expectUpdate: true,
		},
		{
			name:           "No release found",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease:    nil,
			expectUpdate:   false,
		},
		{
			name:              "ForceSemVerPrefix: current without v, release with v, newer",
			channel:           "stable",
			currentVersion:    "1.0.0",
			forceSemVerPrefix: true,
			mockRelease: &Release{
				TagName:    "v1.1.0",
				PreRelease: false,
			},
			expectUpdate: true,
		},
		{
			name:              "ForceSemVerPrefix: current with v, release without v, newer",
			channel:           "stable",
			currentVersion:    "v1.0.0",
			forceSemVerPrefix: true,
			mockRelease: &Release{
				TagName:    "1.1.0",
				PreRelease: false,
			},
			expectUpdate: true,
		},
		{
			name:              "ForceSemVerPrefix: current without v, release without v, newer",
			channel:           "stable",
			currentVersion:    "1.0.0",
			forceSemVerPrefix: true,
			mockRelease: &Release{
				TagName:    "1.1.0",
				PreRelease: false,
			},
			expectUpdate: true,
		},
		{
			name:              "ForceSemVerPrefix: current with v, release with v, same",
			channel:           "stable",
			currentVersion:    "v1.1.0",
			forceSemVerPrefix: true,
			mockRelease: &Release{
				TagName:    "v1.1.0",
				PreRelease: false,
			},
			expectUpdate: false,
		},
		{
			name:              "NoForceSemVerPrefix: current with v, release without v, newer",
			channel:           "stable",
			currentVersion:    "v1.0.0",
			forceSemVerPrefix: false,
			mockRelease: &Release{
				TagName:    "1.1.0",
				PreRelease: false,
			},
			expectUpdate: true,
		},
		{
			name:              "NoForceSemVerPrefix: current without v, release with v, newer",
			channel:           "stable",
			currentVersion:    "1.0.0",
			forceSemVerPrefix: false,
			mockRelease: &Release{
				TagName:    "v1.1.0",
				PreRelease: false,
			},
			expectUpdate: true,
		},
		{
			name:              "NoForceSemVerPrefix: current without v, release without v, same",
			channel:           "stable",
			currentVersion:    "1.1.0",
			forceSemVerPrefix: false,
			mockRelease: &Release{
				TagName:    "1.1.0",
				PreRelease: false,
			},
			expectUpdate: false,
		},
		{
			name:           "Error from GetLatestRelease",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease:    nil,
			expectError:    true,
			GetLatestReleaseFunc: func(ctx context.Context, owner, repo, channel string) (*Release, error) {
				return nil, fmt.Errorf("mock error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalVersion := Version
			Version = tc.currentVersion
			defer func() { Version = originalVersion }()

			mockClient := &MockGithubClient{}
			if tc.GetLatestReleaseFunc != nil {
				mockClient.GetLatestReleaseFunc = tc.GetLatestReleaseFunc
			} else {
				mockClient.GetLatestReleaseFunc = func(ctx context.Context, owner, repo, channel string) (*Release, error) {
					return tc.mockRelease, nil
				}
			}

			// Replace the original client with the mock
			originalNewGithubClient := NewGithubClient
			NewGithubClient = func() GithubClient { return mockClient }
			defer func() { NewGithubClient = originalNewGithubClient }()

			_, updateAvailable, err := CheckForNewerVersion("owner", "repo", tc.channel, tc.forceSemVerPrefix)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			if updateAvailable != tc.expectUpdate {
				t.Errorf("Expected update available: %v, got: %v", tc.expectUpdate, updateAvailable)
			}
		})
	}
}

func TestGetDownloadURL(t *testing.T) {
	testCases := []struct {
		name             string
		release          *Release
		releaseURLFormat string
		expectedURL      string
		expectError      bool
	}{
		{
			name: "With ReleaseURLFormat",
			release: &Release{
				TagName: "v1.0.0",
			},
			releaseURLFormat: "https://example.com/downloads/my-app-{tag}-{os}-{arch}.zip",
			expectedURL:      fmt.Sprintf("https://example.com/downloads/my-app-v1.0.0-%s-%s.zip", runtime.GOOS, runtime.GOARCH),
			expectError:      false,
		},
		{
			name: "No ReleaseURLFormat, matching asset",
			release: &Release{
				TagName: "v1.0.0",
				Assets: []ReleaseAsset{
					{Name: fmt.Sprintf("my-app-%s-%s", runtime.GOOS, runtime.GOARCH), DownloadURL: "https://example.com/my-app-os-arch.zip"},
				},
			},
			expectedURL: "https://example.com/my-app-os-arch.zip",
			expectError: false,
		},
		{
			name: "No ReleaseURLFormat, matching asset (os only)",
			release: &Release{
				TagName: "v1.0.0",
				Assets: []ReleaseAsset{
					{Name: fmt.Sprintf("my-app-%s", runtime.GOOS), DownloadURL: "https://example.com/my-app-os.zip"},
				},
			},
			expectedURL: "https://example.com/my-app-os.zip",
			expectError: false,
		},
		{
			name: "No ReleaseURLFormat, no matching asset",
			release: &Release{
				TagName: "v1.0.0",
				Assets: []ReleaseAsset{
					{Name: "other-app-1.0.0.zip", DownloadURL: "https://example.com/other.zip"},
				},
			},
			expectedURL: "",
			expectError: true,
		},
		{
			name:        "Nil release",
			release:     nil,
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url, err := GetDownloadURL(tc.release, tc.releaseURLFormat)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if url != tc.expectedURL {
				t.Errorf("Expected URL: %s, got: %s", tc.expectedURL, url)
			}
		})
	}
}

func TestCheckForUpdates(t *testing.T) {
	testCases := []struct {
		name              string
		channel           string
		currentVersion    string
		forceSemVerPrefix bool
		releaseURLFormat  string
		mockRelease       *Release
		expectUpdateCall  bool
		expectError       bool
	}{
		{
			name:           "Update available, should call doUpdateFunc",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease: &Release{
				TagName: "v1.1.0",
				Assets: []ReleaseAsset{
					{Name: fmt.Sprintf("test-app-%s-%s", runtime.GOOS, runtime.GOARCH), DownloadURL: "http://example.com/download"},
				},
			},
			expectUpdateCall: true,
		},
		{
			name:           "No update available",
			channel:        "stable",
			currentVersion: "1.1.0",
			mockRelease: &Release{
				TagName: "v1.1.0",
				Assets: []ReleaseAsset{
					{Name: fmt.Sprintf("test-app-%s-%s", runtime.GOOS, runtime.GOARCH), DownloadURL: "http://example.com/download"},
				},
			},
			expectUpdateCall: false,
		},
		{
			name:           "Error fetching release",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease:    nil,
			expectError:    true,
		},
		{
			name:              "Update available with custom URL format",
			channel:           "stable",
			currentVersion:    "1.0.0",
			forceSemVerPrefix: true,
			releaseURLFormat:  "http://example.com/custom-{tag}.zip",
			mockRelease: &Release{
				TagName: "v1.1.0",
			},
			expectUpdateCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalVersion := Version
			Version = tc.currentVersion
			defer func() { Version = originalVersion }()

			mockClient := &MockGithubClient{
				GetLatestReleaseFunc: func(ctx context.Context, owner, repo, channel string) (*Release, error) {
					if tc.expectError && tc.mockRelease == nil {
						return nil, fmt.Errorf("mock error")
					}
					return tc.mockRelease, nil
				},
			}
			originalNewGithubClient := NewGithubClient
			NewGithubClient = func() GithubClient { return mockClient }
			defer func() { NewGithubClient = originalNewGithubClient }()

			updateCalled := false
			originalDoUpdateFunc := doUpdateFunc
			doUpdateFunc = func(url string) error {
				updateCalled = true
				return nil
			}
			defer func() { doUpdateFunc = originalDoUpdateFunc }()

			err := CheckForUpdates("owner", "repo", tc.channel, tc.forceSemVerPrefix, tc.releaseURLFormat)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if updateCalled != tc.expectUpdateCall {
				t.Errorf("Expected doUpdateFunc called: %v, got: %v", tc.expectUpdateCall, updateCalled)
			}
		})
	}
}

func TestCheckOnly(t *testing.T) {
	testCases := []struct {
		name              string
		channel           string
		currentVersion    string
		forceSemVerPrefix bool
		releaseURLFormat  string
		mockRelease       *Release
		expectError       bool
	}{
		{
			name:           "Update available",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease: &Release{
				TagName: "v1.1.0",
			},
			expectError: false,
		},
		{
			name:           "No update available",
			channel:        "stable",
			currentVersion: "1.1.0",
			mockRelease: &Release{
				TagName: "v1.1.0",
			},
			expectError: false,
		},
		{
			name:           "Error fetching release",
			channel:        "stable",
			currentVersion: "1.0.0",
			mockRelease:    nil,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalVersion := Version
			Version = tc.currentVersion
			defer func() { Version = originalVersion }()

			mockClient := &MockGithubClient{
				GetLatestReleaseFunc: func(ctx context.Context, owner, repo, channel string) (*Release, error) {
					if tc.expectError && tc.mockRelease == nil {
						return nil, fmt.Errorf("mock error")
					}
					return tc.mockRelease, nil
				},
			}
			originalNewGithubClient := NewGithubClient
			NewGithubClient = func() GithubClient { return mockClient }
			defer func() { NewGithubClient = originalNewGithubClient }()

			err := CheckOnly("owner", "repo", tc.channel, tc.forceSemVerPrefix, tc.releaseURLFormat)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
		})
	}
}
