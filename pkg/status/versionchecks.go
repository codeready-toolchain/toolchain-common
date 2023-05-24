package status

import (
	"context"
	"fmt"
	"net/http"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/google/go-github/v52/github"
	errs "github.com/pkg/errors"
)

const (
	// ErrMsgDeploymentIsNotUpToDate means that deployment version is not aligned with source code version
	ErrMsgDeploymentIsNotUpToDate = "deployment version is not up to date with latest github commit SHA"

	// DeploymentThresholdInMinutes is the threshold after which we can be almost sure the deployment was not updated on the cluster with the latest version/commit,
	// in this case some issue is preventing the new deployment to happen.
	DeploymentThresholdInMinutes = 30 * time.Minute
)

// CheckDeployedVersionIsUpToDate verifies if there is a match between the latest commit in Github for a given repo and branch matches the provided commit SHA.
// There is some preconfigured delay/threshold that we keep in account before returning an `error condition`.
func CheckDeployedVersionIsUpToDate(githubClient *github.Client, repoName, repoBranch, deployedCommitSHA string) *toolchainv1alpha1.Condition {
	// get the latest commit from given repository and branch
	commits, commitsResponse, err := githubClient.Repositories.ListCommits(context.TODO(), toolchainv1alpha1.ProviderLabelValue, repoName, &github.CommitsListOptions{
		SHA: repoBranch,
	})
	defer commitsResponse.Body.Close()
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			return NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason, ghErr.Message)
		}
	}
	if commitsResponse.StatusCode != http.StatusOK {
		err = errs.New(fmt.Sprintf("invalid response code from github commits API. resp.Response.StatusCode: %d, repoName: %s, repoBranch: %s", commitsResponse.Response.StatusCode, repoName, repoBranch))
		return NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason, err.Error())
	}

	if commits == nil || len(commits) == 0 {
		err = errs.New(fmt.Sprintf("no commits returned. repoName: %s, repoBranch: %s", repoName, repoBranch))
		return NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason, err.Error())
	}
	// check if there is a mismatch between the commit id of the running version and latest commit id from the source code repo (deployed version according to Github actions)
	// we also consider some delay ( time that usually takes the deployment to happen on all our environments)
	githubCommitTimestamp := commits[0].Commit.Author.GetDate()
	expectedDeploymentTime := githubCommitTimestamp.Add(DeploymentThresholdInMinutes) // let's consider some threshold for the deployment to happen
	githubCommitSHA := *commits[0].SHA
	if githubCommitSHA != deployedCommitSHA && time.Now().After(expectedDeploymentTime) {
		// deployed version is not up-to-date after expected threshold
		err := fmt.Errorf("%s. deployed commit SHA %s ,github latest SHA %s, expected deployment timestamp: %s", ErrMsgDeploymentIsNotUpToDate, deployedCommitSHA, githubCommitSHA, expectedDeploymentTime.Format(time.RFC3339))
		return NewComponentErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason, err.Error())
	}

	// no problems with the deployment version, return a ready condition
	return NewComponentReadyCondition(toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason)
}
