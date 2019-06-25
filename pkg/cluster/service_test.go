package cluster_test

import (
	"github.com/codeready-toolchain/toolchain/pkg/cluster"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"testing"
)

func newKubeFedCluster(name, secName string, status v1beta1.KubeFedClusterStatus, labels map[string]string) (*v1beta1.KubeFedCluster, *corev1.Secret) {
	logf.SetLogger(zap.Logger())
	gock.New("http://cluster.com").
		Get("api").
		Persist().
		Reply(200).
		BodyString("{}")
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secName,
			Namespace: "test-namespace",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte("mycooltoken"),
		},
	}

	return &v1beta1.KubeFedCluster{
		Spec: v1beta1.KubeFedClusterSpec{
			SecretRef: v1beta1.LocalSecretReference{
				Name: secName,
			},
			APIEndpoint: "http://cluster.com",
			CABundle:    []byte{},
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "test-namespace",
			Labels:    labels,
		},
		Status: status,
	}, secret
}

func TestAddKubeFedClusterAsMember(t *testing.T) {
	// given
	defer gock.Off()
	status := newClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	memberLabels := []map[string]string{
		labels("", ""),
		labels(cluster.Member, ""),
		labels(cluster.Member, "member-ns")}
	for _, labels := range memberLabels {

		t.Run("add member KubeFedCluster", func(t *testing.T) {
			kubeFedCluster, sec := newKubeFedCluster("east", "secret", status, labels)
			cl := fake.NewFakeClient(sec)
			service := cluster.KubeFedClusterService{Log: logf.Log, Client: cl}
			defer service.DeleteKubeFedCluster(kubeFedCluster)

			// when
			service.AddKubeFedCluster(kubeFedCluster)

			// then
			fedCluster, ok := cluster.GetFedCluster("east")
			require.True(t, ok)
			assert.Equal(t, cluster.Member, fedCluster.Type)
			if labels["namespace"] == "" {
				assert.Equal(t, "toolchain-member-operator", fedCluster.OperatorNamespace)
			} else {
				assert.Equal(t, labels["namespace"], fedCluster.OperatorNamespace)
			}
			assert.Equal(t, status, *fedCluster.ClusterStatus)
		})
	}
}

func TestAddKubeFedClusterAsHost(t *testing.T) {
	// given
	defer gock.Off()
	status := newClusterStatus(common.ClusterReady, corev1.ConditionFalse)
	memberLabels := []map[string]string{
		labels(cluster.Host, ""),
		labels(cluster.Host, "host-ns")}
	for _, labels := range memberLabels {

		t.Run("add host KubeFedCluster", func(t *testing.T) {
			kubeFedCluster, sec := newKubeFedCluster("east", "secret", status, labels)
			cl := fake.NewFakeClient(sec)
			service := cluster.KubeFedClusterService{Log: logf.Log, Client: cl}
			defer service.DeleteKubeFedCluster(kubeFedCluster)

			// when
			service.AddKubeFedCluster(kubeFedCluster)

			// then
			fedCluster, ok := cluster.GetFedCluster("east")
			require.True(t, ok)
			assert.Equal(t, cluster.Host, fedCluster.Type)
			if labels["namespace"] == "" {
				assert.Equal(t, "toolchain-host-operator", fedCluster.OperatorNamespace)
			} else {
				assert.Equal(t, labels["namespace"], fedCluster.OperatorNamespace)
			}
			assert.Equal(t, status, *fedCluster.ClusterStatus)
		})
	}
}

func TestAddKubeFedClusterFailsBecauseOfMissingSecret(t *testing.T) {
	// given
	defer gock.Off()
	status := newClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, _ := newKubeFedCluster("east", "secret", status, labels("", ""))
	cl := fake.NewFakeClient()
	service := cluster.KubeFedClusterService{Log: logf.Log, Client: cl}

	// when
	service.AddKubeFedCluster(kubeFedCluster)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.False(t, ok)
	assert.Nil(t, fedCluster)
}

func TestAddKubeFedClusterFailsBecauseOfEmptySecret(t *testing.T) {
	// given
	defer gock.Off()
	status := newClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, _ := newKubeFedCluster("east", "secret", status, labels("", ""))
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret",
			Namespace: "test-namespace",
		}}
	cl := fake.NewFakeClient(secret)
	service := cluster.KubeFedClusterService{Log: logf.Log, Client: cl}

	// when
	service.AddKubeFedCluster(kubeFedCluster)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.False(t, ok)
	assert.Nil(t, fedCluster)
}

func TestUpdateKubeFedCluster(t *testing.T) {
	// given
	defer gock.Off()
	statusTrue := newClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster1, sec1 := newKubeFedCluster("east", "secret1", statusTrue, labels("", ""))
	statusFalse := newClusterStatus(common.ClusterReady, corev1.ConditionFalse)
	kubeFedCluster2, sec2 := newKubeFedCluster("east", "secret2", statusFalse, labels(cluster.Host, ""))
	cl := fake.NewFakeClient(sec1, sec2)
	service := cluster.KubeFedClusterService{Log: logf.Log, Client: cl}
	defer service.DeleteKubeFedCluster(kubeFedCluster2)
	service.AddKubeFedCluster(kubeFedCluster1)

	// when
	service.AddKubeFedCluster(kubeFedCluster2)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.True(t, ok)
	assert.Equal(t, cluster.Host, fedCluster.Type)
	assert.Equal(t, "toolchain-host-operator", fedCluster.OperatorNamespace)
	assert.Equal(t, statusFalse, *fedCluster.ClusterStatus)
}

func TestDeleteKubeFedCluster(t *testing.T) {
	// given
	defer gock.Off()
	status := newClusterStatus(common.ClusterReady, corev1.ConditionTrue)
	kubeFedCluster, sec := newKubeFedCluster("east", "sec", status, labels("", ""))
	cl := fake.NewFakeClient(sec)
	service := cluster.KubeFedClusterService{Log: logf.Log, Client: cl}
	service.AddKubeFedCluster(kubeFedCluster)

	// when
	service.DeleteKubeFedCluster(kubeFedCluster)

	// then
	fedCluster, ok := cluster.GetFedCluster("east")
	require.False(t, ok)
	assert.Nil(t, fedCluster)
}

func newClusterStatus(conType common.ClusterConditionType, conStatus corev1.ConditionStatus) v1beta1.KubeFedClusterStatus {
	return v1beta1.KubeFedClusterStatus{
		Conditions: []v1beta1.ClusterCondition{{
			Type:   conType,
			Status: conStatus,
		}},
	}
}

func labels(clType cluster.Type, ns string) map[string]string {
	labels := map[string]string{}
	if clType != "" {
		labels["type"] = string(clType)
	}
	if ns != "" {
		labels["namespace"] = ns
	}
	return labels
}
