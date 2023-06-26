package client

import (
	"context"
	"time"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

// GitHubAPICallDelay it's used to "slow" down the number of requests we perform to GitHub API , in order to avoid rate limit issues.
const GitHubAPICallDelay = 1 * time.Minute

// GetGitHubClientFunc a func that returns a GitHub client instance
type GetGitHubClientFunc func(string) *github.Client

type GitHubRepository struct {
	Org, Name, Branch, DeployedCommitSHA string
}

// NewGitHubClient return a client that interacts with GitHub and has rate limiter configured.
// With authenticated GitHub api you can make 5,000 requests per hour.
// see: https://github.com/google/go-github#rate-limiting
func NewGitHubClient(accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(context.TODO(), ts)
	return github.NewClient(tc)
}

// CanIssueGitHubRequest checks if we already called the GitHub API and if call it again since the preconfigured threshold delay expired.
func CanIssueGitHubRequest(lastGitHubAPICall time.Time) bool {
	return lastGitHubAPICall.IsZero() || time.Now().After(lastGitHubAPICall.Add(GitHubAPICallDelay))
}
