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
	"testing"
)

type Assertion struct {
	masterUserRecord *toolchainv1alpha1.MasterUserRecord
	client           client.Client
	namespacedName   types.NamespacedName
	t                *testing.T
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

func AssertThatMasterUserRecord(t *testing.T, name string, client client.Client) *Assertion {
	return &Assertion{
		client:         client,
		namespacedName: test.NamespacedName(test.HostOperatorNs, name),
		t:              t,
	}
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
		if ua.TargetCluster == targetCluster {
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

func (a *Assertion) AllUserAccountsHaveCondition(expected toolchainv1alpha1.Condition) *Assertion {
	err := a.loadUaAssertion()
	require.NoError(a.t, err)
	for _, ua := range a.masterUserRecord.Status.UserAccounts {
		test.AssertConditionsMatch(a.t, ua.Conditions, expected)
	}
	return a
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
