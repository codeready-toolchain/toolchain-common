package test

import (
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

const (
	NameHost   = "dsaas"
	NameMember = "east"
)

func NewKubeFedCluster(name, secName string, status v1beta1.KubeFedClusterStatus, labels map[string]string) (*v1beta1.KubeFedCluster, *corev1.Secret) {
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

func NewClusterStatus(conType common.ClusterConditionType, conStatus corev1.ConditionStatus) v1beta1.KubeFedClusterStatus {
	return v1beta1.KubeFedClusterStatus{
		Conditions: []v1beta1.ClusterCondition{{
			Type:   conType,
			Status: conStatus,
		}},
	}
}
