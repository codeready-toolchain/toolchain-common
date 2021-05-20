package test

import (
	"github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	NameHost   = "dsaas"
	NameMember = "east"
)

func NewToolchainCluster(name, secName string, status v1alpha1.ToolchainClusterStatus, labels map[string]string) (*v1alpha1.ToolchainCluster, *corev1.Secret) {
	return NewToolchainClusterWithEndpoint(name, secName, "http://cluster.com", status, labels)
}

func NewToolchainClusterWithEndpoint(name, secName, apiEndpoint string, status v1alpha1.ToolchainClusterStatus, labels map[string]string) (*v1alpha1.ToolchainCluster, *corev1.Secret) {
	logf.SetLogger(zap.Logger())
	gock.New(apiEndpoint).
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

	return &v1alpha1.ToolchainCluster{
		Spec: v1alpha1.ToolchainClusterSpec{
			SecretRef: v1alpha1.LocalSecretReference{
				Name: secName,
			},
			APIEndpoint: apiEndpoint,
			CABundle:    "",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "test-namespace",
			Labels:    labels,
		},
		Status: status,
	}, secret
}

func NewClusterStatus(conType v1alpha1.ToolchainClusterConditionType, conStatus corev1.ConditionStatus) v1alpha1.ToolchainClusterStatus {
	return v1alpha1.ToolchainClusterStatus{
		Conditions: []v1alpha1.ToolchainClusterCondition{{
			Type:   conType,
			Status: conStatus,
		}},
	}
}
