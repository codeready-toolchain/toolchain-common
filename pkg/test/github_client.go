package test

import (
	"net/http"
	"time"

	"github.com/google/go-github/v52/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

// MockGitHubClientForRepositoryCommits provides a GitHub client which will return the given commit and commit timestamp as a response.
func MockGitHubClientForRepositoryCommits(githubCommitSHA string, commitTimestamp time.Time) *github.Client {
	mockedHTTPClient := MockGithubRepositoryCommits(
		NewMockedGithubCommit(githubCommitSHA, commitTimestamp),
	)
	mockedGitHubClient := github.NewClient(mockedHTTPClient)
	return mockedGitHubClient
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

// MockGithubRepositoryCommits creates a http handler that returns a list of commits for a given org/repo.
func MockGithubRepositoryCommits(repositoryCommits ...*github.RepositoryCommit) *http.Client {
	return mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposCommitsByOwnerByRepo,
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Write(mock.MustMarshal(repositoryCommits)) //nolint: errcheck
			}),
		),
	)
}
