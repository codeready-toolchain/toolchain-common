package status

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/google/go-github/v52/github"
	errs "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ErrMsgDeploymentIsNotUpToDate means that deployment version is not aligned with source code version
	ErrMsgDeploymentIsNotUpToDate = "deployment version is not up to date with latest github commit SHA"

	// DeploymentThreshold is the threshold after which we can be almost sure the deployment was not updated on the cluster with the latest version/commit,
	// in this case some issue is preventing the new deployment to happen.
	DeploymentThreshold = 30 * time.Minute
)

// CheckDeployedVersionIsUpToDate verifies if there is a match between the latest commit in Github for a given repo and branch matches the provided commit SHA.
// There is some preconfigured delay/threshold that we keep in account before returning an `error condition`.
func CheckDeployedVersionIsUpToDate(githubClient *github.Client, repoName, repoBranch, deployedCommitSHA string) *toolchainv1alpha1.Condition {
	// get the latest commit from given repository and branch
	latestCommit, commitResponse, err := githubClient.Repositories.GetCommit(context.TODO(), toolchainv1alpha1.ProviderLabelValue, repoName, repoBranch, &github.ListOptions{})
	defer commitResponse.Body.Close()
	if err != nil {
		errMsg := err.Error()
		if ghErr, ok := err.(*github.ErrorResponse); ok { //nolint:errorlint
			errMsg = ghErr.Message // this strips out the URL called, useful when unit testing since the port changes with each test execution.
		}
		return NewDeploymentVersionUpToDateErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentVersionCheckGitHubErrorReason, errMsg, corev1.ConditionUnknown)
	}
	if commitResponse.StatusCode != http.StatusOK {
		err = errs.New(fmt.Sprintf("invalid response code from github commits API. resp.Response.StatusCode: %d, repoName: %s, repoBranch: %s", commitResponse.Response.StatusCode, repoName, repoBranch))
		return NewDeploymentVersionUpToDateErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentVersionCheckGitHubErrorReason, err.Error(), corev1.ConditionUnknown)
	}

	if reflect.DeepEqual(latestCommit, &github.RepositoryCommit{}) {
		err = errs.New(fmt.Sprintf("no commits returned. repoName: %s, repoBranch: %s", repoName, repoBranch))
		return NewDeploymentVersionUpToDateErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentVersionCheckGitHubErrorReason, err.Error(), corev1.ConditionUnknown)
	}
	// check if there is a mismatch between the commit id of the running version and latest commit id from the source code repo (deployed version according to GitHub actions)
	// we also consider some delay ( time that usually takes the deployment to happen on all our environments)
	githubCommitTimestamp := latestCommit.Commit.Author.GetDate()
	expectedDeploymentTime := githubCommitTimestamp.Add(DeploymentThreshold) // let's consider some threshold for the deployment to happen
	githubCommitSHA := *latestCommit.SHA
	if githubCommitSHA != deployedCommitSHA && time.Now().After(expectedDeploymentTime) {
		// deployed version is not up-to-date after expected threshold
		err := fmt.Errorf("%s. deployed commit SHA %s ,github latest SHA %s, expected deployment timestamp: %s", ErrMsgDeploymentIsNotUpToDate, deployedCommitSHA, githubCommitSHA, expectedDeploymentTime.Format(time.RFC3339))
		return NewDeploymentVersionUpToDateErrorCondition(toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason, err.Error(), corev1.ConditionFalse)
	}

	// no problems with the deployment version, return a ready condition
	return NewDeploymentVersionUpToDateCondition(toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason)
}
