package test

import (
	"fmt"
	"net/http"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/google/go-github/v52/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertHostOperatorStatusMatch asserts that the specified host operator status matches the expected
// host operator status
func AssertHostOperatorStatusMatch(t T, actual toolchainv1alpha1.HostOperatorStatus, expected toolchainv1alpha1.HostOperatorStatus) {
	assert.Equal(t, expected.BuildTimestamp, actual.BuildTimestamp)
	assert.Equal(t, expected.DeploymentName, actual.DeploymentName)
	assert.Equal(t, expected.Revision, actual.Revision)
	assert.Equal(t, expected.Version, actual.Version)
	AssertConditionsMatch(t, actual.Conditions, expected.Conditions...)
}

// AssertMembersMatch asserts that the specified list A of members is equal to specified
// list B of members ignoring the order of the elements.
// It compares only the list of conditions and resource usage.
// We can't use assert.ElementsMatch because the LastTransitionTime of the actual
// conditions can be modified but the conditions still should be treated as matched
func AssertMembersMatch(t T, actual []toolchainv1alpha1.Member, expected ...toolchainv1alpha1.Member) {
	require.Equal(t, len(expected), len(actual))
	for _, c := range expected {
		AssertContainsMember(t, actual, c)
	}
}

// AssertContainsMember asserts that the specified list of members contains the specified member.
// It compares only the list of conditions and resource usage.
// LastTransitionTime is ignored.
func AssertContainsMember(t T, members []toolchainv1alpha1.Member, contains toolchainv1alpha1.Member) {
	for _, c := range members {
		if c.ClusterName == contains.ClusterName {
			t.Logf("checking '%s'", c.ClusterName)
			AssertConditionsMatch(t, c.MemberStatus.Conditions, contains.MemberStatus.Conditions...)
			assert.Equal(t, contains.APIEndpoint, c.APIEndpoint)
			assert.Equal(t, contains.MemberStatus.ResourceUsage, c.MemberStatus.ResourceUsage)
			return
		}
	}
	assert.FailNow(t, fmt.Sprintf("the list of members %+v doesn't contain the expected member %+v", members, contains))
}

// AssertRegistrationServiceStatusMatch asserts that the specified registration service status matches the expected one
func AssertRegistrationServiceStatusMatch(t T, actual toolchainv1alpha1.HostRegistrationServiceStatus, expected toolchainv1alpha1.HostRegistrationServiceStatus) {
	AssertRegistrationServiceDeploymentStatusMatch(t, actual.Deployment, expected.Deployment)
	AssertRegistrationServiceResourcesStatusMatch(t, actual.RegistrationServiceResources, expected.RegistrationServiceResources)
	AssertRegistrationServiceHealthStatusMatch(t, actual.Health, expected.Health)
}

// AssertRegistrationServiceDeploymentStatusMatch asserts that the specified registration service deployment status matches the expected one
func AssertRegistrationServiceDeploymentStatusMatch(t T, actual toolchainv1alpha1.RegistrationServiceDeploymentStatus, expected toolchainv1alpha1.RegistrationServiceDeploymentStatus) {
	assert.Equal(t, expected.Name, actual.Name)
	AssertConditionsMatch(t, actual.Conditions, expected.Conditions...)
}

// AssertRegistrationServiceResourcesStatusMatch asserts that the specified registration service resources status matches the expected one
func AssertRegistrationServiceResourcesStatusMatch(t T, actual toolchainv1alpha1.RegistrationServiceResourcesStatus, expected toolchainv1alpha1.RegistrationServiceResourcesStatus) {
	AssertConditionsMatch(t, actual.Conditions, expected.Conditions...)
}

// AssertRegistrationServiceHealthStatusMatch asserts that the specified registration service health status matches the expected one
func AssertRegistrationServiceHealthStatusMatch(t T, actual toolchainv1alpha1.RegistrationServiceHealth, expected toolchainv1alpha1.RegistrationServiceHealth) {
	AssertConditionsMatch(t, actual.Conditions, expected.Conditions...)
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
