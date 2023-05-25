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
			mockedHTTPClient := test.MockGithubRepositoryCommits(
				test.NewMockedGithubCommit("1234abcd", time.Now().Add(-time.Hour*1)), // latest commit is already deployed
				test.NewMockedGithubCommit("5678efgh", time.Now().Add(-time.Hour*2)),
			)
			githubClient := github.NewClient(mockedHTTPClient)
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "master", "1234abcd") // deployed commit matches latest commit SHA in github

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
				Message: "",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("deployment version is not up to date", func(t *testing.T) {

			t.Run("but we are still within the given 30 minutes threshold", func(t *testing.T) {
				mockedHTTPClient := test.MockGithubRepositoryCommits(
					test.NewMockedGithubCommit("1234abcd", time.Now().Add(-time.Minute*29)), // the latest commit was submitted 29 minutes ago, so still within the threshold.
					test.NewMockedGithubCommit("5678efgh", time.Now().Add(-time.Hour*2)),
				)
				githubClient := github.NewClient(mockedHTTPClient)
				conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "master", "5678efgh") // deployed SHA is still at previous commit

				expected := toolchainv1alpha1.Condition{
					Type:    toolchainv1alpha1.ConditionReady,
					Status:  corev1.ConditionTrue,
					Reason:  toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
					Message: "",
				}
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

			t.Run("30 minutes threshold expired, deployment is not up to date", func(t *testing.T) {
				latestCommitTimestamp := time.Now().Add(-time.Minute * 31)
				mockedHTTPClient := test.MockGithubRepositoryCommits(
					test.NewMockedGithubCommit("1234abcd", latestCommitTimestamp), // the latest commit was submitted 31 minutes ago, threshold has expired and deployment is not up to date.
					test.NewMockedGithubCommit("5678efgh", time.Now().Add(-time.Hour*2)),
				)
				githubClient := github.NewClient(mockedHTTPClient)
				conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "master", "5678efgh") // deployed SHA is still at previous commit

				expected := toolchainv1alpha1.Condition{
					Type:    toolchainv1alpha1.ConditionReady,
					Status:  corev1.ConditionFalse,
					Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
					Message: "deployment version is not up to date with latest github commit SHA. deployed commit SHA 5678efgh ,github latest SHA 1234abcd, expected deployment timestamp: " + latestCommitTimestamp.Add(DeploymentThresholdInMinutes).Format(time.RFC3339),
				}
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

		})

	})

	t.Run("error", func(t *testing.T) {

		t.Run("internal server error from github", func(t *testing.T) {
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mock.WriteError(
							w,
							http.StatusInternalServerError,
							"github went belly up or something",
						)
					}),
				),
			)
			githubClient := github.NewClient(mockedHTTPClient)
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "master", "5678efgh")

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
				Message: "github went belly up or something",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("unexpected response code", func(t *testing.T) {
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusAccepted) // if GitHub returns something different from 200 we consider it invalid
						w.Write(mock.MustMarshal([]*github.RepositoryCommit{}))
					}),
				),
			)
			githubClient := github.NewClient(mockedHTTPClient)
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "master", "5678efgh")

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
				Message: "invalid response code from github commits API. resp.Response.StatusCode: 202, repoName: host-operator, repoBranch: master",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("response with no commits", func(t *testing.T) {
			mockedHTTPClient := test.MockGithubRepositoryCommits([]*github.RepositoryCommit{}...)
			githubClient := github.NewClient(mockedHTTPClient)
			conditions := CheckDeployedVersionIsUpToDate(githubClient, "host-operator", "master", "5678efgh")

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
				Message: "no commits returned. repoName: host-operator, repoBranch: master",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})
	})
}
