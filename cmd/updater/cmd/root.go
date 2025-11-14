package cmd

import (
	"fmt"
	"os"

	"github.com/snider/updater"
	"github.com/spf13/cobra"
)

var (
	checkUpdate        bool
	doUpdate           bool
	channel            string
	forceSemVerPrefix  bool
	releaseURLFormat   string
	pullRequest        int
)

var rootCmd = &cobra.Command{
	Use:   "updater",
	Short: "Updating Wails from GitHub releases made easy",
	Long:  `A demo CLI application showcasing the self-update functionality using GitHub releases.`,
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := "https://github.com/snider/updater"

		// If a channel is specified, use the service-based approach
		if channel != "" {
			var startupMode updater.StartupCheckMode
			if checkUpdate {
				startupMode = updater.CheckOnStartup
			} else if doUpdate {
				startupMode = updater.CheckAndUpdateOnStartup
			} else {
				cmd.Println(cmd.Version)
				return
			}

			config := updater.UpdateServiceConfig{
				RepoURL:           repoURL,
				Channel:           channel,
				CheckOnStartup:    startupMode,
				ForceSemVerPrefix: forceSemVerPrefix,
				ReleaseURLFormat:  releaseURLFormat,
			}

			service, err := updater.NewUpdateService(config)
			if err != nil {
				fmt.Printf("Error creating update service: %v\n", err)
				os.Exit(1)
			}

			if err := service.Start(); err != nil {
				fmt.Printf("Error during update check: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// If no channel is specified, use the tag-based approach
		owner, repo, err := updater.ParseRepoURL(repoURL)
		if err != nil {
			fmt.Printf("Error parsing repo URL: %v\n", err)
			os.Exit(1)
		}

		if pullRequest > 0 {
			if err := updater.CheckForUpdatesByPullRequest(owner, repo, pullRequest, releaseURLFormat); err != nil {
				fmt.Printf("Error performing update for PR: %v\n", err)
				os.Exit(1)
			}
		} else if doUpdate {
			if err := updater.CheckForUpdatesByTag(owner, repo); err != nil {
				fmt.Printf("Error performing update: %v\n", err)
				os.Exit(1)
			}
		} else if checkUpdate {
			if err := updater.CheckOnlyByTag(owner, repo); err != nil {
				fmt.Printf("Error checking for updates: %v\n", err)
				os.Exit(1)
			}
		} else {
			cmd.Println(cmd.Version)
		}
	},
	Version: updater.Version,
}

func Execute() {
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVar(&checkUpdate, "check-update", false, "Check for new updates")
	rootCmd.Flags().BoolVar(&doUpdate, "do-update", false, "Perform an update")
	rootCmd.Flags().StringVar(&channel, "channel", "", "Set the update channel (stable, beta, alpha). If not set, it's determined from the version tag.")
	rootCmd.Flags().BoolVar(&forceSemVerPrefix, "force-semver-prefix", true, "Force 'v' prefix on semver tags")
	rootCmd.Flags().StringVar(&releaseURLFormat, "release-url-format", "", "A URL format for release assets, with {os}, {arch}, and {tag} as placeholders")
	rootCmd.Flags().IntVar(&pullRequest, "pull-request", 0, "Update to a specific pull request")
}
