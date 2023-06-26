package status

import (
	"net/http"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/google/go-github/v52/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckDeployedVersionIsUpToDate(t *testing.T) {
	versionCheckMgr := VersionCheckManager{
		GetGithubClientFunc: test.MockGitHubClientForRepositoryCommits("1234abcd", time.Now().Add(-time.Hour*1)),
		LastGHCallsPerRepo:  nil,
	}
	githubRepo := client.GitHubRepository{
		Org:               toolchainv1alpha1.ProviderLabelValue,
		Name:              "host-operator",
		Branch:            "HEAD",
		DeployedCommitSHA: "1234abcd",
	}

	t.Run("check deployed version status conditions", func(t *testing.T) {
		t.Run("revision check disabled when is not running in prod", func(t *testing.T) {
			// given
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckDisabledReason,
				Message: "is not running in prod environment",
			}

			// when
			conditions := versionCheckMgr.CheckDeployedVersionIsUpToDate(false, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})
		t.Run("revision check disabled when github access token key is not provided", func(t *testing.T) {
			// given
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckDisabledReason,
				Message: "access token key is not provided",
			}

			// when
			conditions := versionCheckMgr.CheckDeployedVersionIsUpToDate(true, "", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("we cannot issue a github api call but we return existing revision check condition", func(t *testing.T) {
			// given
			versionCheckMgrLastCAll := versionCheckMgr
			// let's set last call for this repository to now, so that we make sure it cannot make another call immediately.
			versionCheckMgrLastCAll.LastGHCallsPerRepo = map[string]time.Time{
				"host-operator": time.Now(),
			}
			expected := toolchainv1alpha1.Condition{
				Type:               toolchainv1alpha1.ConditionReady,
				Status:             corev1.ConditionTrue,
				Reason:             toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				LastUpdatedTime:    &metav1.Time{Time: time.Now()},
			}
			alreadyExistingConditions := []toolchainv1alpha1.Condition{
				expected,
			}

			// when
			conditions := versionCheckMgrLastCAll.CheckDeployedVersionIsUpToDate(true, "githubToken", alreadyExistingConditions, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("deployment version is up to date", func(t *testing.T) {
			// given
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
				Message: "",
			}

			// when
			conditions := versionCheckMgr.CheckDeployedVersionIsUpToDate(true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo) // deployed commit matches latest commit SHA in github

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("deployment version is not up to date", func(t *testing.T) {
			t.Run("but we are still within the given 30 minutes threshold", func(t *testing.T) {
				// given
				latestCommitTimestamp := time.Now().Add(-time.Minute * 29)
				versionCheckMgrThreshold := VersionCheckManager{
					GetGithubClientFunc: test.MockGitHubClientForRepositoryCommits("1234abcd", latestCommitTimestamp),
					LastGHCallsPerRepo:  nil,
				}
				expected := toolchainv1alpha1.Condition{
					Type:    toolchainv1alpha1.ConditionReady,
					Status:  corev1.ConditionTrue,
					Reason:  toolchainv1alpha1.ToolchainStatusDeploymentUpToDateReason,
					Message: "",
				}
				githubRepo.DeployedCommitSHA = "5678efgh" // deployed SHA is still at previous commit

				// when
				conditions := versionCheckMgrThreshold.CheckDeployedVersionIsUpToDate(true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

				// then
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

			t.Run("30 minutes threshold expired, deployment is not up to date", func(t *testing.T) {
				// given
				latestCommitTimestamp := time.Now().Add(-time.Minute * 31)
				versionCheckMgrThresholdExpired := VersionCheckManager{
					GetGithubClientFunc: test.MockGitHubClientForRepositoryCommits("1234abcd", latestCommitTimestamp),
					LastGHCallsPerRepo:  nil,
				}
				expected := toolchainv1alpha1.Condition{
					Type:    toolchainv1alpha1.ConditionReady,
					Status:  corev1.ConditionFalse,
					Reason:  toolchainv1alpha1.ToolchainStatusDeploymentNotUpToDateReason,
					Message: "deployment version is not up to date with latest github commit SHA. deployed commit SHA 5678efgh ,github latest SHA 1234abcd, expected deployment timestamp: " + latestCommitTimestamp.Add(DeploymentThreshold).Format(time.RFC3339),
				}
				githubRepo.DeployedCommitSHA = "5678efgh" // deployed SHA is still at previous commit

				// when
				conditions := versionCheckMgrThresholdExpired.CheckDeployedVersionIsUpToDate(true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

				// when
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

		})

	})

	t.Run("error", func(t *testing.T) {
		t.Run("internal server error from github", func(t *testing.T) {
			// given
			versionCheckMgrError := VersionCheckManager{
				GetGithubClientFunc: func(string) *github.Client {
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
					return github.NewClient(mockedHTTPClient)
				},
				LastGHCallsPerRepo: nil,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckGitHubErrorReason,
				Message: "github went belly up or something",
			}

			// when
			conditions := versionCheckMgrError.CheckDeployedVersionIsUpToDate(true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("response with no commits", func(t *testing.T) {
			// given
			versionCheckMgrError := VersionCheckManager{
				GetGithubClientFunc: func(string) *github.Client {
					mockedHTTPClient := test.MockGithubRepositoryCommit(nil)
					return github.NewClient(mockedHTTPClient)
				},
				LastGHCallsPerRepo: nil,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckGitHubErrorReason,
				Message: "no commits returned. repoName: host-operator, repoBranch: HEAD",
			}

			// when
			conditions := versionCheckMgrError.CheckDeployedVersionIsUpToDate(true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("we cannot issue a github api call and there are no conditions set yet", func(t *testing.T) {
			// given

			versionCheckMgrNoCond := versionCheckMgr
			versionCheckMgrNoCond.LastGHCallsPerRepo = map[string]time.Time{
				"host-operator": time.Now(), // let's set last call for this repository to now, so that we make sure it cannot make another call immediately.
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  toolchainv1alpha1.ToolchainStatusDeploymentRevisionCheckOperatorErrorReason,
				Message: "unable to find ConditionReady type in existing conditions. Waiting for next attempt ...",
			}

			// when
			conditions := versionCheckMgrNoCond.CheckDeployedVersionIsUpToDate(true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})
	})
}
