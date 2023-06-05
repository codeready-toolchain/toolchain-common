package status

import (
	"net/http"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/google/go-github/v52/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	corev1 "k8s.io/api/core/v1"
)

func TestCheckDeployedVersionIsUpToDate(t *testing.T) {

	t.Run("check deployed version status conditions", func(t *testing.T) {
		t.Run("deployment version is up to date", func(t *testing.T) {
			// given
			mockedHTTPClient := test.MockGithubRepositoryCommit(
				test.NewMockedGithubCommit("1234abcd", time.Now().Add(-time.Hour*1)), // latest commit is already deployed
			)
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
				Message: "",
			}
			githubClient := github.NewClient(mockedHTTPClient)

			// when
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "HEAD", "1234abcd") // deployed commit matches latest commit SHA in github

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("deployment version is not up to date", func(t *testing.T) {
			t.Run("but we are still within the given 30 minutes threshold", func(t *testing.T) {
				// given
				mockedHTTPClient := test.MockGithubRepositoryCommit(
					test.NewMockedGithubCommit("1234abcd", time.Now().Add(-time.Minute*29)), // the latest commit was submitted 29 minutes ago, so still within the threshold.
				)
				expected := toolchainv1alpha1.Condition{
					Type:    toolchainv1alpha1.ConditionReady,
					Status:  corev1.ConditionTrue,
					Reason:  toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
					Message: "",
				}
				githubClient := github.NewClient(mockedHTTPClient)

				// when
				conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "HEAD", "5678efgh") // deployed SHA is still at previous commit

				// then
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

			t.Run("30 minutes threshold expired, deployment is not up to date", func(t *testing.T) {
				// given
				latestCommitTimestamp := time.Now().Add(-time.Minute * 31)
				mockedHTTPClient := test.MockGithubRepositoryCommit(
					test.NewMockedGithubCommit("1234abcd", latestCommitTimestamp), // the latest commit was submitted 31 minutes ago, threshold has expired and deployment is not up to date.
				)
				expected := toolchainv1alpha1.Condition{
					Type:    toolchainv1alpha1.ConditionReady,
					Status:  corev1.ConditionFalse,
					Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
					Message: "deployment version is not up to date with latest github commit SHA. deployed commit SHA 5678efgh ,github latest SHA 1234abcd, expected deployment timestamp: " + latestCommitTimestamp.Add(DeploymentThreshold).Format(time.RFC3339),
				}
				githubClient := github.NewClient(mockedHTTPClient)

				// when
				conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "HEAD", "5678efgh") // deployed SHA is still at previous commit

				// when
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

		})

	})

	t.Run("error", func(t *testing.T) {

		t.Run("internal server error from github", func(t *testing.T) {
			// given
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					test.GetReposCommitsByOwnerByRepoByRef,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mock.WriteError(
							w,
							http.StatusInternalServerError,
							"github went belly up or something",
						)
					}),
				),
			)
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
				Message: "github went belly up or something",
			}
			githubClient := github.NewClient(mockedHTTPClient)

			// when
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "HEAD", "5678efgh")

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("response with no commits", func(t *testing.T) {
			// given
			mockedHTTPClient := test.MockGithubRepositoryCommit(nil)
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
				Message: "no commits returned. repoName: host-operator, repoBranch: HEAD",
			}
			githubClient := github.NewClient(mockedHTTPClient)

			// when
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "HEAD", "5678efgh")

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})
	})
}
