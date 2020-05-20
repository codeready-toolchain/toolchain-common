package masteruserrecord

import (
	"context"
	"fmt"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Assertion struct {
	masterUserRecord *toolchainv1alpha1.MasterUserRecord
	client           client.Client
	namespacedName   types.NamespacedName
	t                test.T
}

func (a *Assertion) loadUaAssertion() error {
	if a.masterUserRecord != nil {
		return nil
	}
	mur := &toolchainv1alpha1.MasterUserRecord{}
	err := a.client.Get(context.TODO(), a.namespacedName, mur)
	a.masterUserRecord = mur
	return err
}

func AssertThatMasterUserRecord(t test.T, name string, client client.Client) *Assertion {
	return &Assertion{
		client:         client,
		namespacedName: test.NamespacedName(test.HostOperatorNs, name),
		t:              t,
	}
}

type NsTemplateSetSpecExp func(*toolchainv1alpha1.NSTemplateSetSpec)

func WithTier(tier string) NsTemplateSetSpecExp {
	return func(set *toolchainv1alpha1.NSTemplateSetSpec) {
		set.TierName = tier
	}
}

func WithNs(nsType, revision string) NsTemplateSetSpecExp {
	return func(set *toolchainv1alpha1.NSTemplateSetSpec) {
		set.Namespaces = append(set.Namespaces, toolchainv1alpha1.NSTemplateSetNamespace{
			Type:        nsType,
			Revision:    revision,
			TemplateRef: set.TierName + "-" + nsType + "-" + revision,
		})
	}
}

func WithClusterRes(revision string) NsTemplateSetSpecExp {
	return func(set *toolchainv1alpha1.NSTemplateSetSpec) {
		if set.ClusterResources == nil {
			set.ClusterResources = &toolchainv1alpha1.NSTemplateSetClusterResources{}
		}
		set.ClusterResources.Revision = revision
		set.ClusterResources.TemplateRef = set.TierName + "-" + "clusterresources" + "-" + revision
	}
}

// HasNSTemplateSet verifies that the MUR has NSTemplateSetSpec with the expected values
func (a *Assertion) HasNSTemplateSet(targetCluster string, expectations ...NsTemplateSetSpecExp) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	expectedTmplSetSpec := &toolchainv1alpha1.NSTemplateSetSpec{}
	for _, modify := range expectations {
		modify(expectedTmplSetSpec)
	}
	for _, ua := range a.masterUserRecord.Spec.UserAccounts {
		if ua.TargetCluster == targetCluster {
			assert.Equal(a.t, *expectedTmplSetSpec, ua.Spec.NSTemplateSet)
			return a
		}
	}
	a.t.Fatalf("unable to find an NSTemplateSet for the '%s' target cluster", targetCluster)
	return a
}

func (a *Assertion) HasNoConditions() *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	require.Empty(a.t, a.masterUserRecord.Status.Conditions)
	return a
}

func (a *Assertion) HasConditions(expected ...toolchainv1alpha1.Condition) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	test.AssertConditionsMatch(a.t, a.masterUserRecord.Status.Conditions, expected...)
	return a
}

func (a *Assertion) HasStatusUserAccounts(targetClusters ...string) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	require.Len(a.t, a.masterUserRecord.Status.UserAccounts, len(targetClusters))
	for _, cluster := range targetClusters {
		a.hasUserAccount(cluster)
	}
	return a
}

func (a *Assertion) hasUserAccount(targetCluster string) *toolchainv1alpha1.UserAccountStatusEmbedded {
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		if ua.Cluster.Name == targetCluster {
			return &ua
		}
	}
	assert.Fail(a.t, fmt.Sprintf("user account status record for the target cluster %s was not found", targetCluster))
	return nil
}

func (a *Assertion) AllUserAccountsHaveStatusSyncIndex(syncIndex string) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		assert.Equal(a.t, syncIndex, ua.SyncIndex)
	}
	return a
}

func (a *Assertion) AllUserAccountsHaveCluster(expected toolchainv1alpha1.Cluster) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		assert.Equal(a.t, expected, ua.Cluster)
	}
	return a
}

func (a *Assertion) AllUserAccountsHaveCondition(expected toolchainv1alpha1.Condition) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		test.AssertConditionsMatch(a.t, ua.Conditions, expected)
	}
	return a
}

func (a *Assertion) AllUserAccountsHaveTier(tier toolchainv1alpha1.NSTemplateTier) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Spec.UserAccounts {
		a.userAccountHasTier(ua, tier)
	}
	return a
}

func (a *Assertion) UserAccountHasTier(targetCluster string, tier toolchainv1alpha1.NSTemplateTier) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Spec.UserAccounts {
		if ua.TargetCluster == targetCluster {
			a.userAccountHasTier(ua, tier)
		}
	}
	return a
}

func (a *Assertion) userAccountHasTier(ua toolchainv1alpha1.UserAccountEmbedded, tier toolchainv1alpha1.NSTemplateTier) {
	assert.Equal(a.t, tier.Name, ua.Spec.NSTemplateSet.TierName)
	actualTemplateRefs := []string{}
	for _, ns := range ua.Spec.NSTemplateSet.Namespaces {
		actualTemplateRefs = append(actualTemplateRefs, ns.TemplateRef)
	}
	expectedTemplateRefs := []string{}
	for _, ns := range tier.Spec.Namespaces {
		expectedTemplateRefs = append(expectedTemplateRefs, ns.TemplateRef)
	}
	a.t.Logf("expected templateRefs: %v vs actual: %v", expectedTemplateRefs, actualTemplateRefs)
	assert.ElementsMatch(a.t, expectedTemplateRefs, actualTemplateRefs)
	if tier.Spec.ClusterResources == nil {
		assert.Nil(a.t, ua.Spec.NSTemplateSet.ClusterResources)
	} else {
		assert.Equal(a.t, tier.Spec.ClusterResources.TemplateRef, ua.Spec.NSTemplateSet.ClusterResources.TemplateRef)
	}
}

func (a *Assertion) HasFinalizer() *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	assert.Len(a.t, a.masterUserRecord.Finalizers, 1)
	assert.Contains(a.t, a.masterUserRecord.Finalizers, "finalizer.toolchain.dev.openshift.com")
	return a
}

func (a *Assertion) DoesNotHaveFinalizer() *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	assert.Len(a.t, a.masterUserRecord.Finalizers, 0)
	return a
}
