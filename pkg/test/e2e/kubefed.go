package e2e

import (
	"context"
	"github.com/codeready-toolchain/api/pkg/apis"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
)

const (
	OperatorRetryInterval = time.Second * 5
	OperatorTimeout       = time.Second * 60
	RetryInterval         = time.Millisecond * 100
	Timeout               = time.Second * 3
	CleanupRetryInterval  = time.Second * 1
	CleanupTimeout        = time.Second * 5
	MemberNsVar           = "MEMBER_NS"
	HostNsVar             = "HOST_NS"
)

// VerifyKubeFedCluster verifies existence and correct conditions of KubeFedCluster CRD
// in the target cluster type operator
func VerifyKubeFedCluster(t *testing.T, targetClusterType cluster.Type) {
	// given
	fedClusterList := &v1beta1.KubeFedClusterList{}
	ctx, awaitility := InitializeOperators(t, fedClusterList, targetClusterType)
	defer ctx.Cleanup()

	var kubeFedClusterType cluster.Type
	var singleAwait *SingleAwaitility
	var expNs string

	if targetClusterType == cluster.Host {
		kubeFedClusterType = cluster.Member
		singleAwait = awaitility.Host()
		expNs = awaitility.MemberNs
	} else {
		kubeFedClusterType = cluster.Host
		singleAwait = awaitility.Member()
		expNs = awaitility.HostNs
	}

	current, ok, err := singleAwait.GetKubeFedCluster(expNs, kubeFedClusterType, nil)
	require.NoError(awaitility.T, err)
	require.True(awaitility.T, ok, "KubeFedCluster should exist")
	labels := KubeFedLabels(kubeFedClusterType, current.Labels["namespace"], current.Labels["ownerClusterName"])

	t.Run("create new KubeFedCluster with correct data and expect to be ready", func(t *testing.T) {
		// given
		newName := "new-ready-" + string(kubeFedClusterType)
		newFedCluster := NewKubeFedCluster(singleAwait.Ns, newName, current.Spec.CABundle,
			current.Spec.APIEndpoint, current.Spec.SecretRef.Name, labels)

		// when
		err := awaitility.Client.Create(context.TODO(), newFedCluster, CleanupOptions(ctx))

		// then the KubeFedCluster should be ready
		require.NoError(t, err)
		err = singleAwait.WaitForKubeFedClusterConditionWithName(newFedCluster.Name, ReadyKubeFedCluster)
		require.NoError(t, err)
		err = awaitility.WaitForReadyKubeFedClusters()
		require.NoError(t, err)
		err = singleAwait.WaitForKubeFedClusterConditionWithName(current.Name, ReadyKubeFedCluster)
		require.NoError(t, err)
	})
	t.Run("create new KubeFedCluster with incorrect data and expect to be offline", func(t *testing.T) {
		// given
		newName := "new-offline-" + string(kubeFedClusterType)
		newFedCluster := NewKubeFedCluster(singleAwait.Ns, newName, current.Spec.CABundle,
			"https://1.2.3.4:8443", current.Spec.SecretRef.Name, labels)

		// when
		err := awaitility.Client.Create(context.TODO(), newFedCluster, CleanupOptions(ctx))

		// then the KubeFedCluster should be offline
		require.NoError(t, err)
		err = singleAwait.WaitForKubeFedClusterConditionWithName(newFedCluster.Name, &v1beta1.ClusterCondition{
			Type:   common.ClusterOffline,
			Status: corev1.ConditionTrue,
		})
		require.NoError(t, err)
		err = awaitility.WaitForReadyKubeFedClusters()
		require.NoError(t, err)
	})
}

// InitializeOperators initializes test context, registers schemes and waits until both operators (host, member)
// and corresponding KubeFedCluster CRDs are present, running and ready. Based on the given cluster type
// that represents the current operator that is the target of the e2e test it retrieves namespace names.
// Returns the test context and an instance of Awaitility that contains all necessary information
func InitializeOperators(t *testing.T, obj runtime.Object, clusterType cluster.Type) (*framework.TestCtx, *Awaitility) {
	err := framework.AddToFrameworkScheme(apis.AddToSchemes.AddToScheme, obj)
	require.NoError(t, err, "failed to add custom resource scheme to framework")

	ctx := framework.NewTestCtx(t)

	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval})
	require.NoError(t, err, "failed to initialize cluster resources")
	t.Log("Initialized cluster resources")

	hostNs, err := ctx.GetNamespace()
	memberNs := os.Getenv(MemberNsVar)
	require.NoError(t, err, "failed to get namespace where operator needs to run")
	if clusterType == cluster.Member {
		memberNs = hostNs
		hostNs = os.Getenv(HostNsVar)
	}

	// get global framework variables
	f := framework.Global

	// wait for host operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, hostNs, "host-operator", 1, OperatorRetryInterval, OperatorTimeout)
	require.NoError(t, err, "failed while waiting for host operator deployment")

	// wait for member operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, memberNs, "member-operator", 1, OperatorRetryInterval, OperatorTimeout)
	require.NoError(t, err, "failed while waiting for member operator deployment")

	awaitility := &Awaitility{
		T:                t,
		Client:           f.Client,
		KubeClient:       f.KubeClient,
		ControllerClient: f.Client.Client,
		HostNs:           hostNs,
		MemberNs:         memberNs,
	}

	err = awaitility.WaitForReadyKubeFedClusters()
	require.NoError(t, err)

	t.Log("both operators are ready and in running state")
	return ctx, awaitility
}

// NewKubeFedCluster creates KubeFedCluster CR object with the given values
func NewKubeFedCluster(ns, name string, caBundle []byte, apiEndpoint, secretRef string, labels map[string]string) *v1beta1.KubeFedCluster {
	return &v1beta1.KubeFedCluster{
		ObjectMeta: v1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    labels,
		},
		Spec: v1beta1.KubeFedClusterSpec{
			CABundle:    caBundle,
			APIEndpoint: apiEndpoint,
			SecretRef: v1beta1.LocalSecretReference{
				Name: secretRef,
			},
		},
	}
}

// KubeFedLabels takes the label values and returns a key-value map containing label names key and values
func KubeFedLabels(clType cluster.Type, ns, ownerClusterName string) map[string]string {
	labels := map[string]string{}
	if clType != "" {
		labels["type"] = string(clType)
	}
	if ns != "" {
		labels["namespace"] = ns
	}
	labels["ownerClusterName"] = ownerClusterName
	return labels
}
