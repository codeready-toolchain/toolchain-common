package client

import (
	"context"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

// NewGitHubClient return a client that interacts with GitHub and has rate limiter configured.
// With authenticated GitHub api you can make 5,000 requests per hour.
// But the Search API has a custom rate limit.
// Unauthenticated clients are limited to 10 requests per minute, while authenticated clients can make up to 30 requests per minute.
// see: https://github.com/google/go-github#rate-limiting
//
// The RoundTripper waits for the secondary rate limit to finish in a blocking mode and then issues/retries requests.
// see: https://github.com/gofri/go-github-ratelimit
func NewGitHubClient(accessToken string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(context.TODO(), ts)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(tc.Transport)
	return github.NewClient(rateLimiter), err
}
