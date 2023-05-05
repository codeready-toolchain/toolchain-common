package masteruserrecord

import (
	"context"
	"fmt"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MasterUserRecordAssertion struct { // nolint: revive
	mur            *toolchainv1alpha1.MasterUserRecord
	client         runtimeclient.Reader
	namespacedName types.NamespacedName
	t              test.T
}

func (a *MasterUserRecordAssertion) loadMasterUserRecord() error {
	mur := &toolchainv1alpha1.MasterUserRecord{}
	err := a.client.Get(context.TODO(), a.namespacedName, mur)
	a.mur = mur
	return err
}

func AssertThatMasterUserRecord(t test.T, name string, client runtimeclient.Reader) *MasterUserRecordAssertion {
	return &MasterUserRecordAssertion{
		client:         client,
		namespacedName: test.NamespacedName(test.HostOperatorNs, name),
		t:              t,
	}
}

func (a *MasterUserRecordAssertion) Exists() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	return a
}

func (a *MasterUserRecordAssertion) DoesNotExist() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.EqualError(a.t, err, fmt.Sprintf("masteruserrecords.toolchain.dev.openshift.com \"%s\" not found", a.namespacedName.Name))
	return a
}

func (a *MasterUserRecordAssertion) Get() *toolchainv1alpha1.MasterUserRecord {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	return a.mur
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
			TemplateRef: set.TierName + "-" + nsType + "-" + revision,
		})
	}
}

func WithClusterRes(revision string) NsTemplateSetSpecExp {
	return func(set *toolchainv1alpha1.NSTemplateSetSpec) {
		if set.ClusterResources == nil {
			set.ClusterResources = &toolchainv1alpha1.NSTemplateSetClusterResources{}
		}
		set.ClusterResources.TemplateRef = set.TierName + "-" + "clusterresources" + "-" + revision
	}
}

func (a *MasterUserRecordAssertion) HasNoConditions() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Empty(a.t, a.mur.Status.Conditions)
	return a
}

func (a *MasterUserRecordAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	test.AssertConditionsMatch(a.t, a.mur.Status.Conditions, expected...)
	return a
}

func (a *MasterUserRecordAssertion) HasStatusUserAccounts(targetClusters ...string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Len(a.t, a.mur.Status.UserAccounts, len(targetClusters))
	for _, cluster := range targetClusters {
		a.hasUserAccount(cluster)
	}
	return a
}

func (a *MasterUserRecordAssertion) hasUserAccount(targetCluster string) *toolchainv1alpha1.UserAccountStatusEmbedded {
	for _, ua := range a.mur.Status.UserAccounts {
		if ua.Cluster.Name == targetCluster {
			return &ua
		}
	}
	assert.Fail(a.t, fmt.Sprintf("user account status record for the target cluster %s was not found", targetCluster))
	return nil
}

func (a *MasterUserRecordAssertion) AllUserAccountsHaveCluster(expected toolchainv1alpha1.Cluster) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.mur.Status.UserAccounts {
		assert.Equal(a.t, expected, ua.Cluster)
	}
	return a
}

func (a *MasterUserRecordAssertion) AllUserAccountsHaveCondition(expected toolchainv1alpha1.Condition) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	for _, ua := range a.mur.Status.UserAccounts {
		test.AssertConditionsMatch(a.t, ua.Conditions, expected)
	}
	return a
}

func (a *MasterUserRecordAssertion) HasTier(tier toolchainv1alpha1.UserTier) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Equal(a.t, tier.Name, a.mur.Spec.TierName)
	return a
}

func (a *MasterUserRecordAssertion) HasFinalizer() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Len(a.t, a.mur.Finalizers, 1)
	assert.Contains(a.t, a.mur.Finalizers, "finalizer.toolchain.dev.openshift.com")
	return a
}

func (a *MasterUserRecordAssertion) DoesNotHaveFinalizer() *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Len(a.t, a.mur.Finalizers, 0)
	return a
}

// DoesNotHaveLabel verifies that the MasterUserRecord does not have
// a label with the given key
func (a *MasterUserRecordAssertion) DoesNotHaveLabel(key string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.NotContains(a.t, a.mur.Labels, key)
	return a
}

// HasLabel verifies that the MasterUserRecord has
// a label with the given key
func (a *MasterUserRecordAssertion) HasLabel(key string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Contains(a.t, a.mur.Labels, key)
	assert.NotEmpty(a.t, a.mur.Labels[key])
	return a
}

// HasLabelWithValue verifies that the MasterUserRecord has
// a label with the given key and value
func (a *MasterUserRecordAssertion) HasLabelWithValue(key, value string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Equal(a.t, value, a.mur.Labels[key])
	return a
}

// HasAnnotationWithValue verifies that the MasterUserRecord has
// an annotation with the given key and value
func (a *MasterUserRecordAssertion) HasAnnotationWithValue(key, value string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Contains(a.t, a.mur.Annotations, key)
	assert.Equal(a.t, value, a.mur.Annotations[key])
	return a
}

func (a *MasterUserRecordAssertion) HasOriginalSub(sub string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	assert.Equal(a.t, sub, a.mur.Spec.OriginalSub)
	return a
}

func (a *MasterUserRecordAssertion) HasTargetCluster(targetcluster string) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.NotEmpty(a.t, a.mur.Spec.UserAccounts)
	assert.Equal(a.t, targetcluster, a.mur.Spec.UserAccounts[0].TargetCluster)
	return a
}

func (a *MasterUserRecordAssertion) HasUserAccounts(count int) *MasterUserRecordAssertion {
	err := a.loadMasterUserRecord()
	require.NoError(a.t, err)
	require.Len(a.t, a.mur.Spec.UserAccounts, count)
	return a
}

// Assertions on multiple MasterUserRecords at once

type MasterUserRecordsAssertion struct {
	murs      *toolchainv1alpha1.MasterUserRecordList
	client    runtimeclient.Reader
	namespace string
	t         test.T
}

func AssertThatMasterUserRecords(t test.T, client runtimeclient.Reader) *MasterUserRecordsAssertion {
	return &MasterUserRecordsAssertion{
		client:    client,
		namespace: test.HostOperatorNs,
		t:         t,
	}
}

func (a *MasterUserRecordsAssertion) loadMasterUserRecords() error {
	murs := &toolchainv1alpha1.MasterUserRecordList{}
	err := a.client.List(context.TODO(), murs, runtimeclient.InNamespace(a.namespace))
	a.murs = murs
	return err
}

func (a *MasterUserRecordsAssertion) HaveCount(count int) *MasterUserRecordsAssertion {
	err := a.loadMasterUserRecords()
	require.NoError(a.t, err)
	require.Len(a.t, a.murs.Items, count)
	return a
}
