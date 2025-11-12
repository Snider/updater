package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"
)

// NewGithubClient is a variable that holds a function to create a new GithubClient.
// This can be replaced in tests to inject a mock client.
var NewGithubClient = func() GithubClient {
	return &githubClient{}
}

// doUpdateFunc is a variable that holds the function to perform the actual update.
// This can be replaced in tests to prevent actual updates.
var doUpdateFunc = func(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(resp.Body)

	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("failed to rollback from failed update: %v", rerr)
		}
		return fmt.Errorf("update failed: %v", err)
	}

	fmt.Println("Update applied successfully.")
	return nil
}

// CheckForNewerVersion is a variable for a function that checks if a newer version is available.
var CheckForNewerVersion = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool) (*Release, bool, error) {
	client := NewGithubClient()
	ctx := context.Background()

	release, err := client.GetLatestRelease(ctx, owner, repo, channel)
	if err != nil {
		return nil, false, fmt.Errorf("error fetching latest release: %w", err)
	}

	if release == nil {
		return nil, false, nil // No release found
	}

	// Always normalize to 'v' prefix for semver comparison
	vCurrent := formatVersionForComparison(currentVersion)
	vLatest := formatVersionForComparison(release.TagName)

	if semver.Compare(vCurrent, vLatest) >= 0 {
		return release, false, nil // Current version is up-to-date or newer
	}

	return release, true, nil // A newer version is available
}

// CheckForUpdates is a variable for a function that checks for and applies new updates.
var CheckForUpdates = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool, releaseURLFormat string) error {
	release, updateAvailable, err := CheckForNewerVersion(owner, repo, channel, currentVersion, forceSemVerPrefix)
	if err != nil {
		return err
	}

	if !updateAvailable {
		if release != nil {
			fmt.Printf("Current version %s is up-to-date with latest release %s.\n",
				formatVersionForDisplay(currentVersion, forceSemVerPrefix),
				formatVersionForDisplay(release.TagName, forceSemVerPrefix))
		} else {
			fmt.Println("No releases found.")
		}
		return nil
	}

	fmt.Printf("Newer version %s found (current: %s). Applying update...\n",
		formatVersionForDisplay(release.TagName, forceSemVerPrefix),
		formatVersionForDisplay(currentVersion, forceSemVerPrefix))

	downloadURL, err := GetDownloadURL(release, releaseURLFormat)
	if err != nil {
		return fmt.Errorf("error getting download URL: %w", err)
	}

	return doUpdateFunc(downloadURL)
}

// CheckOnly is a variable for a function that checks for new updates without applying them.
var CheckOnly = func(owner, repo, channel, currentVersion string, forceSemVerPrefix bool, releaseURLFormat string) error {
	release, updateAvailable, err := CheckForNewerVersion(owner, repo, channel, currentVersion, forceSemVerPrefix)
	if err != nil {
		return err
	}

	if !updateAvailable {
		if release != nil {
			fmt.Printf("Current version %s is up-to-date with latest release %s.\n",
				formatVersionForDisplay(currentVersion, forceSemVerPrefix),
				formatVersionForDisplay(release.TagName, forceSemVerPrefix))
		} else {
			fmt.Println("No new release found.")
		}
		return nil
	}

	fmt.Printf("New release found: %s (current version: %s)\n",
		formatVersionForDisplay(release.TagName, forceSemVerPrefix),
		formatVersionForDisplay(currentVersion, forceSemVerPrefix))
	return nil
}

// CheckForUpdatesByTag is a variable for a function that checks for updates based on the channel determined by the current version tag.
var CheckForUpdatesByTag = func(owner, repo, currentVersion string) error {
	channel := determineChannel(currentVersion, false) // isPreRelease is false for current version
	return CheckForUpdates(owner, repo, channel, currentVersion, true, "")
}

// CheckOnlyByTag is a variable for a function that checks for updates based on the channel determined by the current version tag, without applying them.
var CheckOnlyByTag = func(owner, repo, currentVersion string) error {
	channel := determineChannel(currentVersion, false) // isPreRelease is false for current version
	return CheckOnly(owner, repo, channel, currentVersion, true, "")
}

// formatVersionForComparison ensures the version string has a 'v' prefix for semver comparison.
func formatVersionForComparison(version string) string {
	if version != "" && !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// formatVersionForDisplay ensures the version string has the correct 'v' prefix based on the forceSemVerPrefix flag.
func formatVersionForDisplay(version string, forceSemVerPrefix bool) string {
	hasV := strings.HasPrefix(version, "v")
	if forceSemVerPrefix && !hasV {
		return "v" + version
	}
	if !forceSemVerPrefix && hasV {
		return strings.TrimPrefix(version, "v")
	}
	return version
}
