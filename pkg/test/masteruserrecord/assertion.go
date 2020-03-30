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

type MasterUserRecordAssertion struct {
	masterUserRecord *toolchainv1alpha1.MasterUserRecord
	client           client.Client
	namespacedName   types.NamespacedName
	t                test.T
}

func (a *MasterUserRecordAssertion) loadMasterUserRecord() error {
	mur := &toolchainv1alpha1.MasterUserRecord{}
	err := a.client.Get(context.TODO(), a.namespacedName, mur)
	a.masterUserRecord = mur
	return err
}

func AssertThatMasterUserRecord(t test.T, name string, client client.Client) *MasterUserRecordAssertion {
	return &MasterUserRecordAssertion{
		client:         client,
		namespacedName: types.NamespacedName{Namespace: test.HostOperatorNs, Name: name},
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
			Type:     nsType,
			Revision: revision,
		})
	}
}

func WithClusterRes(revision string) NsTemplateSetSpecExp {
	return func(set *toolchainv1alpha1.NSTemplateSetSpec) {
		if set.ClusterResources == nil {
			set.ClusterResources = &toolchainv1alpha1.NSTemplateSetClusterResources{}
		}
		set.ClusterResources.Revision = revision
	}
}

// HasNSTemplateSet verifies that the MUR has NSTemplateSetSpec with the expected values
func (a *MasterUserRecordAssertion) HasNSTemplateSet(targetCluster string, expectations ...NsTemplateSetSpecExp) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
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

func (a *MasterUserRecordAssertion) HasNoConditions() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Empty(a.t, a.masterUserRecord.Status.Conditions)
	return a
}

func (a *MasterUserRecordAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	test.AssertConditionsMatch(a.t, a.masterUserRecord.Status.Conditions, expected...)
	return a
}

func (a *MasterUserRecordAssertion) HasStatusUserAccounts(targetClusters ...string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Len(a.t, a.masterUserRecord.Status.UserAccounts, len(targetClusters))
	for _, cluster := range targetClusters {
		a.hasUserAccount(cluster)
	}
	return a
}

func (a *MasterUserRecordAssertion) hasUserAccount(targetCluster string) *toolchainv1alpha1.UserAccountStatusEmbedded {
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		if ua.Cluster.Name == targetCluster {
			return &ua
		}
	}
	assert.Fail(a.t, fmt.Sprintf("user account status record for the target cluster %s was not found", targetCluster))
	return nil
}

func (a *MasterUserRecordAssertion) AllUserAccountsHaveStatusSyncIndex(syncIndex string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		assert.Equal(a.t, syncIndex, ua.SyncIndex)
	}
	return a
}

func (a *MasterUserRecordAssertion) AllUserAccountsHaveCluster(expected toolchainv1alpha1.Cluster) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		assert.Equal(a.t, expected, ua.Cluster)
	}
	return a
}

func (a *MasterUserRecordAssertion) AllUserAccountsHaveCondition(expected toolchainv1alpha1.Condition) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		test.AssertConditionsMatch(a.t, ua.Conditions, expected)
	}
	return a
}

func (a *MasterUserRecordAssertion) AllUserAccountsHaveTier(tier toolchainv1alpha1.NSTemplateTier) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Spec.UserAccounts {
		a.userAccountHasTier(ua, tier)
	}
	return a
}

func (a *MasterUserRecordAssertion) UserAccountHasTier(targetCluster string, tier toolchainv1alpha1.NSTemplateTier) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Spec.UserAccounts {
		if ua.TargetCluster == targetCluster {
			a.userAccountHasTier(ua, tier)
		}
	}
	return a
}

func (a *MasterUserRecordAssertion) userAccountHasTier(userAccount toolchainv1alpha1.UserAccountEmbedded, tier toolchainv1alpha1.NSTemplateTier) {
	assert.Equal(a.t, tier.Name, userAccount.Spec.NSTemplateSet.TierName)
	assert.Len(a.t, userAccount.Spec.NSTemplateSet.Namespaces, len(tier.Spec.Namespaces))

TierNamespaces:
	for _, ns := range tier.Spec.Namespaces {
		for _, uaNs := range userAccount.Spec.NSTemplateSet.Namespaces {
			if ns.Type == uaNs.Type {
				assert.Equal(a.t, ns.Revision, uaNs.Revision)
				continue TierNamespaces
			}
		}
		assert.Failf(a.t, "unable to find namespace of type %s in UserAccount %v", ns.Type, userAccount)
	}
}

func (a *MasterUserRecordAssertion) HasFinalizer() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Len(a.t, a.masterUserRecord.Finalizers, 1)
	assert.Contains(a.t, a.masterUserRecord.Finalizers, "finalizer.toolchain.dev.openshift.com")
	return a
}

func (a *MasterUserRecordAssertion) DoesNotHaveFinalizer() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Len(a.t, a.masterUserRecord.Finalizers, 0)
	return a
}
