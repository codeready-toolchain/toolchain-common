package test

import (
	"net/http"
	"time"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/google/go-github/v52/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

var GetReposCommitsByOwnerByRepoByRef mock.EndpointPattern = mock.EndpointPattern{
	Pattern: "/repos/{owner}/{repo}/commits/{ref}",
	Method:  "GET",
}

// MockGitHubClientForRepositoryCommits provides a GitHub client which will return the given commit and commit timestamp as a response.
func MockGitHubClientForRepositoryCommits(githubCommitSHA string, commitTimestamp time.Time) client.GetGitHubClientFunc {
	mockedHTTPClient := MockGithubRepositoryCommit(
		NewMockedGithubCommit(githubCommitSHA, commitTimestamp),
	)
	mockedGitHubClient := github.NewClient(mockedHTTPClient)
	return func(string) *github.Client {
		return mockedGitHubClient
	}
}

// NewMockedGithubCommit create a GitHub.Commit object with given SHA and timestamp
func NewMockedGithubCommit(commitSHA string, commitTimestamp time.Time) *github.RepositoryCommit {
	return &github.RepositoryCommit{
		SHA: github.String(commitSHA),
		Commit: &github.Commit{
			Author: &github.CommitAuthor{
				Date: &github.Timestamp{Time: commitTimestamp},
			},
		},
	}
}

// MockGithubRepositoryCommit creates a http handler that returns a commit for a given org/repo.
func MockGithubRepositoryCommit(repositoryCommit *github.RepositoryCommit) *http.Client {
	return mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			GetReposCommitsByOwnerByRepoByRef,
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Write(mock.MustMarshal(repositoryCommit)) //nolint: errcheck
			}),
		),
	)
}
