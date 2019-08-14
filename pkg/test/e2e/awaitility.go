package e2e

import (
	"context"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"testing"
	"time"
)

// Awaitility contains information necessary for verifying availability of resources in both operators
type Awaitility struct {
	T        *testing.T
	Client   framework.FrameworkClient
	MemberNs string
	HostNs   string
}

// SingleAwaitility contains information necessary for verifying availability of resources in a single operator
type SingleAwaitility struct {
	T      *testing.T
	Client framework.FrameworkClient
	Ns     string
}

// Member creates SingleAwaitility for the member operator
func (a *Awaitility) Member() *SingleAwaitility {
	return &SingleAwaitility{
		T:      a.T,
		Client: a.Client,
		Ns:     a.MemberNs,
	}
}

// Host creates SingleAwaitility for the host operator
func (a *Awaitility) Host() *SingleAwaitility {
	return &SingleAwaitility{
		T:      a.T,
		Client: a.Client,
		Ns:     a.HostNs,
	}
}

// ReadyKubeFedCluster is a ClusterCondition that represents cluster that is ready
var ReadyKubeFedCluster = &v1beta1.ClusterCondition{
	Type:   common.ClusterReady,
	Status: v1.ConditionTrue,
}

// WaitForReadyKubeFedClusters waits until both KubeFedClusters (host and member) exist and has ready ClusterCondition
func (a *Awaitility) WaitForReadyKubeFedClusters() error {
	if err := a.Host().WaitForKubeFedCluster(a.MemberNs, cluster.Member, ReadyKubeFedCluster); err != nil {
		return err
	}
	if err := a.Member().WaitForKubeFedCluster(a.HostNs, cluster.Host, ReadyKubeFedCluster); err != nil {
		return err
	}
	return nil
}

// WaitForKubeFedCluster waits until there is a KubeFedCluster representing a operator of the given type
// and running in the given expected namespace. If the given condition is not nil, then it also checks
// if the CR has the ClusterCondition
func (a *SingleAwaitility) WaitForKubeFedCluster(expNs string, clusterType cluster.Type, condition *v1beta1.ClusterCondition) error {
	timeout := Timeout
	if condition != nil {
		timeout = (util.DefaultClusterHealthCheckPeriod + 5) * time.Second
	}
	return wait.Poll(RetryInterval, timeout, func() (done bool, err error) {
		_, ok, err := a.GetKubeFedCluster(expNs, clusterType, condition)
		if ok {
			return true, nil
		}
		a.T.Logf("waiting for availability of %s KubeFedCluster CR (in namespace %s) representing operator running in namespace '%s'", clusterType, a.Ns, expNs)
		return false, err
	})
}

// WaitForKubeFedClusterConditionWithName waits until there is a KubeFedCluster with the given name
// and with the given ClusterCondition (if it the condition is nil, then it skips this check)
func (a *SingleAwaitility) WaitForKubeFedClusterConditionWithName(name string, condition *v1beta1.ClusterCondition) error {
	timeout := Timeout
	if condition != nil {
		timeout = (util.DefaultClusterHealthCheckPeriod + 5) * time.Second
	}
	return wait.Poll(RetryInterval, timeout, func() (done bool, err error) {
		cluster := &v1beta1.KubeFedCluster{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Ns, Name: name}, cluster); err != nil {
			return false, err
		}
		if containsClusterCondition(cluster.Status.Conditions, condition) {
			a.T.Logf("found %s KubeFedCluster", name)
			return true, nil
		}
		a.T.Logf("waiting for %s KubeFedCluster having the expected condition", name)
		return false, err
	})
}

// GetKubeFedCluster retrieves and returns a KubeFedCluster representing a operator of the given type
// and running in the given expected namespace. If the given condition is not nil, then it also checks
// if the CR has the ClusterCondition
func (a *SingleAwaitility) GetKubeFedCluster(expNs string, clusterType cluster.Type, condition *v1beta1.ClusterCondition) (v1beta1.KubeFedCluster, bool, error) {
	clusters := &v1beta1.KubeFedClusterList{}
	if err := a.Client.List(context.TODO(), &client.ListOptions{Namespace: a.Ns}, clusters); err != nil {
		return v1beta1.KubeFedCluster{}, false, err
	}
	for _, cl := range clusters.Items {
		if cl.Labels["namespace"] == expNs && cluster.Type(cl.Labels["type"]) == clusterType {
			if containsClusterCondition(cl.Status.Conditions, condition) {
				a.T.Logf("found %s KubeFedCluster running in namespace '%s'", clusterType, expNs)
				return cl, true, nil
			}
		}
	}
	return v1beta1.KubeFedCluster{}, false, nil
}

func containsClusterCondition(conditions []v1beta1.ClusterCondition, contains *v1beta1.ClusterCondition) bool {
	if contains == nil {
		return true
	}
	for _, c := range conditions {
		if c.Type == contains.Type {
			return contains.Status == c.Status
		}
	}
	return false
}
