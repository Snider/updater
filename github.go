package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"golang.org/x/oauth2"
)

type Repo struct {
	CloneURL string `json:"clone_url"`
}

// ReleaseAsset represents a release asset from the GitHub API.
type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// Release represents a release from the GitHub API.
type Release struct {
	TagName    string         `json:"tag_name"`
	PreRelease bool           `json:"prerelease"`
	Assets     []ReleaseAsset `json:"assets"`
}

// GithubClient is an interface for interacting with the Github API.
type GithubClient interface {
	GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error)
	GetLatestRelease(ctx context.Context, owner, repo, channel string) (*Release, error)
}

type githubClient struct{}

// NewAuthenticatedClient creates a new authenticated http client.
var NewAuthenticatedClient = func(ctx context.Context) *http.Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return http.DefaultClient
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return oauth2.NewClient(ctx, ts)
}

func (g *githubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return g.getPublicReposWithAPIURL(ctx, "https://api.github.com", userOrOrg)
}

func (g *githubClient) getPublicReposWithAPIURL(ctx context.Context, apiURL, userOrOrg string) ([]string, error) {
	client := NewAuthenticatedClient(ctx)
	var allCloneURLs []string
	url := fmt.Sprintf("%s/users/%s/repos", apiURL, userOrOrg)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Borg-Data-Collector")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			// Try organization endpoint
			url = fmt.Sprintf("%s/orgs/%s/repos", apiURL, userOrOrg)
			req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("User-Agent", "Borg-Data-Collector")
			resp, err = client.Do(req)
			if err != nil {
				return nil, err
			}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch repos: %s", resp.Status)
		}

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, repo := range repos {
			allCloneURLs = append(allCloneURLs, repo.CloneURL)
		}

		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}
		nextURL := g.findNextURL(linkHeader)
		if nextURL == "" {
			break
		}
		url = nextURL
	}

	return allCloneURLs, nil
}

func (g *githubClient) findNextURL(linkHeader string) string {
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(link, ";")
		if len(parts) == 2 && strings.TrimSpace(parts[1]) == `rel="next"` {
			return strings.Trim(strings.TrimSpace(parts[0]), "<>")
		}
	}
	return ""
}

// GetLatestRelease fetches the latest release for a given repository and channel.
// The channel can be "stable", "beta", or "alpha".
func (g *githubClient) GetLatestRelease(ctx context.Context, owner, repo, channel string) (*Release, error) {
	client := NewAuthenticatedClient(ctx)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Borg-Data-Collector")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch releases: %s", resp.Status)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return filterReleases(releases, channel), nil
}

// filterReleases filters releases based on the specified channel.
func filterReleases(releases []Release, channel string) *Release {
	for _, release := range releases {
		releaseChannel := determineChannel(release.TagName, release.PreRelease)
		if releaseChannel == channel {
			return &release
		}
	}
	return nil
}

// determineChannel determines the stability channel of a release based on its tag and PreRelease flag.
func determineChannel(tagName string, isPreRelease bool) string {
	tagLower := strings.ToLower(tagName)
	if strings.Contains(tagLower, "alpha") {
		return "alpha"
	}
	if strings.Contains(tagLower, "beta") {
		return "beta"
	}
	if isPreRelease { // A pre-release without alpha/beta is treated as beta
		return "beta"
	}
	return "stable"
}

// GetDownloadURL finds the appropriate download URL for the current OS and architecture.
// If a releaseURLFormat is provided, it will be used to construct the URL.
func GetDownloadURL(release *Release, releaseURLFormat string) (string, error) {
	if release == nil {
		return "", fmt.Errorf("no release provided")
	}

	if releaseURLFormat != "" {
		// Replace {tag}, {os}, and {arch} placeholders
		r := strings.NewReplacer(
			"{tag}", release.TagName,
			"{os}", runtime.GOOS,
			"{arch}", runtime.GOARCH,
		)
		return r.Replace(releaseURLFormat), nil
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH

	for _, asset := range release.Assets {
		assetNameLower := strings.ToLower(asset.Name)
		// Match asset that contains both OS and architecture
		if strings.Contains(assetNameLower, osName) && strings.Contains(assetNameLower, archName) {
			return asset.DownloadURL, nil
		}
	}

	// Fallback for OS only if no asset matched both OS and arch
	for _, asset := range release.Assets {
		assetNameLower := strings.ToLower(asset.Name)
		if strings.Contains(assetNameLower, osName) {
			return asset.DownloadURL, nil
		}
	}

	return "", fmt.Errorf("no suitable download asset found for %s/%s", osName, archName)
}
