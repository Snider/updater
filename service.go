package updater

import (
	"fmt"
	"net/url"
	"strings"
)

// StartupCheckMode defines the updater's behavior on startup.
type StartupCheckMode int

const (
	// NoCheck disables any checks on startup.
	NoCheck StartupCheckMode = iota
	// CheckOnStartup checks for updates on startup but does not apply them.
	CheckOnStartup
	// CheckAndUpdateOnStartup checks for and applies updates on startup.
	CheckAndUpdateOnStartup
)

// UpdateServiceConfig holds the configuration for the UpdateService.
type UpdateServiceConfig struct {
	RepoURL           string
	Channel           string
	CurrentVersion    string
	CheckOnStartup    StartupCheckMode
	ForceSemVerPrefix bool   // If true, ensures 'v' prefix. If false, ensures no 'v' prefix.
	ReleaseURLFormat  string // A URL format for release assets, with {tag} as a placeholder.
}

// UpdateService provides a configurable interface for handling application updates.
type UpdateService struct {
	config   UpdateServiceConfig
	isGitHub bool
	owner    string
	repo     string
}

// NewUpdateService creates and configures a new UpdateService.
func NewUpdateService(config UpdateServiceConfig) (*UpdateService, error) {
	isGitHub := strings.Contains(config.RepoURL, "github.com")
	var owner, repo string
	var err error

	if isGitHub {
		owner, repo, err = ParseRepoURL(config.RepoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GitHub repo URL: %w", err)
		}
	}

	return &UpdateService{
		config:   config,
		isGitHub: isGitHub,
		owner:    owner,
		repo:     repo,
	}, nil
}

// Start initiates the update check based on the service configuration.
func (s *UpdateService) Start() error {
	if s.isGitHub {
		return s.startGitHubCheck()
	}
	return s.startHTTPCheck()
}

func (s *UpdateService) startGitHubCheck() error {
	switch s.config.CheckOnStartup {
	case NoCheck:
		return nil // Do nothing
	case CheckOnStartup:
		return CheckOnly(s.owner, s.repo, s.config.Channel, s.config.CurrentVersion, s.config.ForceSemVerPrefix, s.config.ReleaseURLFormat)
	case CheckAndUpdateOnStartup:
		return CheckForUpdates(s.owner, s.repo, s.config.Channel, s.config.CurrentVersion, s.config.ForceSemVerPrefix, s.config.ReleaseURLFormat)
	default:
		return fmt.Errorf("unknown startup check mode: %d", s.config.CheckOnStartup)
	}
}

func (s *UpdateService) startHTTPCheck() error {
	switch s.config.CheckOnStartup {
	case NoCheck:
		return nil // Do nothing
	case CheckOnStartup:
		return CheckOnlyHTTP(s.config.RepoURL, s.config.CurrentVersion)
	case CheckAndUpdateOnStartup:
		return CheckForUpdatesHTTP(s.config.RepoURL, s.config.CurrentVersion)
	default:
		return fmt.Errorf("unknown startup check mode: %d", s.config.CheckOnStartup)
	}
}

// ParseRepoURL extracts the owner and repository name from a GitHub URL.
func ParseRepoURL(repoURL string) (owner string, repo string, err error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repo URL path: %s", u.Path)
	}
	return parts[0], parts[1], nil
}
