package status

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/google/go-github/v52/github"
	errs "github.com/pkg/errors"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ErrMsgDeploymentIsNotUpToDate means that deployment version is not aligned with source code version
	ErrMsgDeploymentIsNotUpToDate = "deployment version is not up to date with latest github commit SHA"

	// DeploymentThreshold is the threshold after which we can be almost sure the deployment was not updated on the cluster with the latest version/commit,
	// in this case some issue is preventing the new deployment to happen.
	DeploymentThreshold = 30 * time.Minute
)

type VersionCheckManager struct {
	GetGithubClientFunc client.GetGitHubClientFunc
	LastGHCallsPerRepo  map[string]time.Time
}

// CheckDeployedVersionIsUpToDate verifies if there is a match between the latest commit in GitHub for a given repo and branch matches the provided commit SHA.
// There is some preconfigured delay/threshold that we keep in account before returning an `error condition`.
func (m *VersionCheckManager) CheckDeployedVersionIsUpToDate(ctx context.Context, isProd bool, accessTokenKey string, alreadyExistingConditions []toolchainv1alpha1.Condition, githubRepo client.GitHubRepository) *toolchainv1alpha1.Condition {
	// the first two checks are pretty much the same for all components
	if !isProd {
		cond := NewComponentReadyCondition(toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckDisabledReason)
		cond.Message = "is not running in prod environment"
		return cond
	}
	if accessTokenKey == "" {
		cond := NewComponentReadyCondition(toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckDisabledReason)
		cond.Message = "access token key is not provided"
		return cond
	}
	// we can store the last call per repo name, so it will solve the gaps between calls for host & reg-service which is done form the same controller
	if m.LastGHCallsPerRepo == nil {
		m.LastGHCallsPerRepo = map[string]time.Time{}
	}
	lastCall, present := m.LastGHCallsPerRepo[githubRepo.Name]
	if present && !client.CanIssueGitHubRequest(lastCall) {
		// return existing condition when we cannot make a new GitHub api call due to rate limiting issues.
		previouslySet, found := condition.FindConditionByType(alreadyExistingConditions, toolchainv1alpha1.ConditionReady)
		if !found {
			cond := NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckOperatorErrorReason, "unable to find ConditionReady type in existing conditions. Waiting for next attempt ...")
			return cond
		}
		return &previouslySet
	}
	m.LastGHCallsPerRepo[githubRepo.Name] = time.Now()
	githubClient := m.GetGithubClientFunc(ctx, accessTokenKey)
	// get the latest commit from given repository and branch
	latestCommit, err := getLatestCommit(ctx, githubClient.Repositories.GetCommit, githubRepo)
	if err != nil {
		return NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckGitHubErrorReason, err.Error())
	}
	// check if there is a mismatch between the commit id of the running version and latest commit id from the source code repo (deployed version according to GitHub actions)
	// we also consider some delay ( time that usually takes the deployment to happen on all our environments)
	githubCommitTimestamp := latestCommit.Commit.Author.GetDate()
	expectedDeploymentTime := githubCommitTimestamp.Add(DeploymentThreshold) // let's consider some threshold for the deployment to happen
	githubCommitSHA := *latestCommit.SHA
	if githubCommitSHA != githubRepo.DeployedCommitSHA && time.Now().After(expectedDeploymentTime) {
		// deployed version is not up-to-date after expected threshold
		err := fmt.Errorf("%s. deployed commit SHA %s ,github latest SHA %s, expected deployment timestamp: %s", ErrMsgDeploymentIsNotUpToDate, githubRepo.DeployedCommitSHA, githubCommitSHA, expectedDeploymentTime.Format(time.RFC3339))
		return NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason, err.Error())
	}

	// no problems with the deployment version, return a ready condition
	return NewComponentReadyCondition(toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason)
}

type getCommitFunc func(ctx context.Context, owner string, repo string, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error)

func getLatestCommit(ctx context.Context, GetCommit getCommitFunc, githubRepo client.GitHubRepository) (*github.RepositoryCommit, error) {
	latestCommit, commitResponse, err := GetCommit(ctx, githubRepo.Org, githubRepo.Name, githubRepo.Branch, &github.ListOptions{})
	defer func() {
		if commitResponse != nil && commitResponse.Body != nil {
			if err := commitResponse.Body.Close(); err != nil {
				logger.FromContext(ctx).Error(err, "unable to close response body")
			}
		}
	}()
	if err != nil {
		errMsg := err.Error()
		if ghErr, ok := err.(*github.ErrorResponse); ok { //nolint:errorlint
			errMsg = ghErr.Message // this strips out the URL called, useful when unit testing since the port changes with each test execution.
		}
		return nil, errors.New(errMsg)
	}
	if commitResponse.StatusCode != http.StatusOK {
		err = errs.New(fmt.Sprintf("invalid response code from github commits API. resp.Response.StatusCode: %d, repoName: %s, repoBranch: %s", commitResponse.StatusCode, githubRepo.Name, githubRepo.Branch))
		return nil, err
	}

	if latestCommit == nil || reflect.DeepEqual(latestCommit, &github.RepositoryCommit{}) {
		err = errs.New(fmt.Sprintf("no commits returned. repoName: %s, repoBranch: %s", githubRepo.Name, githubRepo.Branch))
		return nil, err
	}
	return latestCommit, nil
}
