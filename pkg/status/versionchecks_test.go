package status

import (
	"context"
	"net/http"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/google/go-github/v52/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			conditions := versionCheckMgr.CheckDeployedVersionIsUpToDate(context.TODO(), false, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

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
			conditions := versionCheckMgr.CheckDeployedVersionIsUpToDate(context.TODO(), true, "", []toolchainv1alpha1.Condition{}, githubRepo)

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
			conditions := versionCheckMgrLastCAll.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", alreadyExistingConditions, githubRepo)

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
			conditions := versionCheckMgr.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo) // deployed commit matches latest commit SHA in github

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
				conditions := versionCheckMgrThreshold.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

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
				conditions := versionCheckMgrThresholdExpired.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

				// when
				test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
			})

		})

	})

	t.Run("error", func(t *testing.T) {
		t.Run("internal server error from github", func(t *testing.T) {
			// given
			versionCheckMgrError := VersionCheckManager{
				GetGithubClientFunc: func(context.Context, string) *github.Client {
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
			conditions := versionCheckMgrError.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})

		t.Run("response with no commits", func(t *testing.T) {
			// given
			versionCheckMgrError := VersionCheckManager{
				GetGithubClientFunc: func(context.Context, string) *github.Client {
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
			conditions := versionCheckMgrError.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

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
			conditions := versionCheckMgrNoCond.CheckDeployedVersionIsUpToDate(context.TODO(), true, "githubToken", []toolchainv1alpha1.Condition{}, githubRepo)

			// then
			test.AssertConditionsMatchAndRecentTimestamps(t, []toolchainv1alpha1.Condition{*conditions}, expected)
		})
	})
}

func TestGetLatestCommit(t *testing.T) {
	githubRepo := client.GitHubRepository{
		Org:               toolchainv1alpha1.ProviderLabelValue,
		Name:              "host-operator",
		Branch:            "HEAD",
		DeployedCommitSHA: "1234abcd",
	}

	t.Run("happy path", func(t *testing.T) {
		// given
		ghClient := test.MockGitHubClientForRepositoryCommits("1234abcd", time.Now().Add(-time.Hour*1))

		// when
		latestCommit, err := getLatestCommit(context.TODO(), ghClient(context.TODO(), "").Repositories.GetCommit, githubRepo)

		// then
		require.NoError(t, err)
		require.NotNil(t, latestCommit)
		assert.Equal(t, "1234abcd", *latestCommit.SHA)
	})

	t.Run("error", func(t *testing.T) {
		// when
		latestCommit, err := getLatestCommit(context.TODO(), func(ctx context.Context, owner string, repo string, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
			return nil, nil, errors.New("some error")
		}, githubRepo)

		// then
		require.EqualError(t, err, "some error")
		require.Nil(t, latestCommit)
	})

	t.Run("with not found response", func(t *testing.T) {
		// when
		latestCommit, err := getLatestCommit(context.TODO(), func(ctx context.Context, owner string, repo string, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
			resp := &github.Response{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
			}
			return nil, resp, nil
		}, githubRepo)

		// then
		require.EqualError(t, err, "invalid response code from github commits API. resp.Response.StatusCode: 404, repoName: host-operator, repoBranch: HEAD")
		require.Nil(t, latestCommit)
	})

	t.Run("with empty latest commit", func(t *testing.T) {
		// given
		resp := &github.Response{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}

		t.Run("latest commit is nil", func(t *testing.T) {
			// given
			latestCommit, err := getLatestCommit(context.TODO(), func(ctx context.Context, owner string, repo string, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
				return nil, resp, nil
			}, githubRepo)

			// then
			require.EqualError(t, err, "no commits returned. repoName: host-operator, repoBranch: HEAD")
			require.Nil(t, latestCommit)
		})

		t.Run("latest commit is empty", func(t *testing.T) {
			// given
			latestCommit, err := getLatestCommit(context.TODO(), func(ctx context.Context, owner string, repo string, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
				return &github.RepositoryCommit{}, resp, nil
			}, githubRepo)

			// then
			require.EqualError(t, err, "no commits returned. repoName: host-operator, repoBranch: HEAD")
			require.Nil(t, latestCommit)
		})
	})
}
