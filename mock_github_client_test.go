package updater

import (
	"context"
	"fmt"
)

// MockGithubClient is a mock implementation of the GithubClient interface for testing.
type MockGithubClient struct {
	GetLatestReleaseFunc func(ctx context.Context, owner, repo, channel string) (*Release, error)
}

// GetPublicRepos is a mock implementation of the GetPublicRepos method.
func (m *MockGithubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetLatestRelease is a mock implementation of the GetLatestRelease method.
func (m *MockGithubClient) GetLatestRelease(ctx context.Context, owner, repo, channel string) (*Release, error) {
	if m.GetLatestReleaseFunc != nil {
		return m.GetLatestReleaseFunc(ctx, owner, repo, channel)
	}
	return nil, fmt.Errorf("GetLatestReleaseFunc not set")
}
