package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/snider/updater"
	"github.com/spf13/cobra"
)

// execute is a helper function to test cobra commands
func execute(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	err := c.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestRootCmd(t *testing.T) {
	// This function creates a new rootCmd for each test case
	newRootCmd := func() *cobra.Command {
		// Re-create the command to get a fresh set of flags
		root := &cobra.Command{
			Use:   "updater",
			Short: "Updating Wails from GitHub releases made easy",
			Long:  `A demo CLI application showcasing the self-update functionality using GitHub releases.`,
			Run:   rootCmd.Run, // Use the original Run function
		}
		// Initialize flags for this new command
		root.Flags().BoolVar(&checkUpdate, "check-update", false, "Check for new updates")
		root.Flags().BoolVar(&doUpdate, "do-update", false, "Perform an update")
		root.Flags().StringVar(&channel, "channel", "", "Set the update channel (stable, beta, alpha). If not set, it's determined from the version tag.")
		root.Flags().StringVar(&currentVersionFlag, "current-version", "", "Override the current version for testing")
		root.Flags().BoolVar(&forceSemVerPrefix, "force-semver-prefix", true, "Force 'v' prefix on semver tags")
		root.Flags().StringVar(&releaseURLFormat, "release-url-format", "", "A URL format for release assets, with {os}, {arch}, and {tag} as placeholders")
		root.Version = version
		return root
	}

	testCases := []struct {
		name            string
		args            []string
		checkOnlyCalls  int
		checkAndDoCalls int
		checkOnlyByTag  int
		checkAndDoByTag int
		expectOutput    string
		expectError     bool
	}{
		{
			name:         "No flags (prints version)",
			args:         []string{},
			expectOutput: "dev", // Default version
		},
		{
			name:           "check-update flag with channel",
			args:           []string{"--check-update", "--channel=stable"},
			checkOnlyCalls: 1,
		},
		{
			name:            "do-update flag with channel",
			args:            []string{"--do-update", "--channel=stable"},
			checkAndDoCalls: 1,
		},
		{
			name:           "check-update flag no channel",
			args:           []string{"--check-update"},
			checkOnlyByTag: 1,
		},
		{
			name:            "do-update flag no channel",
			args:            []string{"--do-update"},
			checkAndDoByTag: 1,
		},
		{
			name:         "Version flag",
			args:         []string{"--version"},
			expectOutput: "updater version dev",
		},
		{
			name:        "Invalid flag",
			args:        []string{"--invalid-flag"},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var checkOnlyCalls, checkAndDoCalls, checkOnlyByTagCalls, checkAndDoByTagCalls int

			// Mock the updater functions
			originalCheckOnly := updater.CheckOnly
			updater.CheckOnly = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool, releaseURLFormat string) error {
				checkOnlyCalls++
				return nil
			}
			defer func() { updater.CheckOnly = originalCheckOnly }()

			originalCheckForUpdates := updater.CheckForUpdates
			updater.CheckForUpdates = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool, releaseURLFormat string) error {
				checkAndDoCalls++
				return nil
			}
			defer func() { updater.CheckForUpdates = originalCheckForUpdates }()

			originalCheckOnlyByTag := updater.CheckOnlyByTag
			updater.CheckOnlyByTag = func(owner, repo, currentVersion string) error {
				checkOnlyByTagCalls++
				return nil
			}
			defer func() { updater.CheckOnlyByTag = originalCheckOnlyByTag }()

			originalCheckForUpdatesByTag := updater.CheckForUpdatesByTag
			updater.CheckForUpdatesByTag = func(owner, repo, currentVersion string) error {
				checkAndDoByTagCalls++
				return nil
			}
			defer func() { updater.CheckForUpdatesByTag = originalCheckForUpdatesByTag }()

			cmd := newRootCmd()
			output, err := execute(t, cmd, tc.args...)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			if tc.expectOutput != "" && !strings.Contains(output, tc.expectOutput) {
				t.Errorf("Expected output to contain: %q, got: %q", tc.expectOutput, output)
			}

			if checkOnlyCalls != tc.checkOnlyCalls {
				t.Errorf("Expected CheckOnly calls: %d, got: %d", tc.checkOnlyCalls, checkOnlyCalls)
			}
			if checkAndDoCalls != tc.checkAndDoCalls {
				t.Errorf("Expected CheckForUpdates calls: %d, got: %d", tc.checkAndDoCalls, checkAndDoCalls)
			}
			if checkOnlyByTagCalls != tc.checkOnlyByTag {
				t.Errorf("Expected CheckOnlyByTag calls: %d, got: %d", tc.checkOnlyByTag, checkOnlyByTagCalls)
			}
			if checkAndDoByTagCalls != tc.checkAndDoByTag {
				t.Errorf("Expected CheckForUpdatesByTag calls: %d, got: %d", tc.checkAndDoByTag, checkAndDoByTagCalls)
			}
		})
	}
}
