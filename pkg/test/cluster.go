package test

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	NameHost   = "dsaas"
	NameMember = "east"
)

func NewToolchainCluster(t *testing.T, name, tcNs, operatorNs, secName string, status toolchainv1alpha1.ToolchainClusterStatus, insecure bool) (*toolchainv1alpha1.ToolchainCluster, *corev1.Secret) {
	t.Helper()
	return NewToolchainClusterWithEndpoint(t, name, tcNs, operatorNs, secName, "https://cluster.com", status, insecure)
}

func NewToolchainClusterWithEndpoint(t *testing.T, name, tcNs, operatorNs, secName, apiEndpoint string, status toolchainv1alpha1.ToolchainClusterStatus, insecureTls bool) (*toolchainv1alpha1.ToolchainCluster, *corev1.Secret) {
	t.Helper()
	gock.New(apiEndpoint).
		Get("api").
		Persist().
		Reply(200).
		BodyString("{}")

	kubeConfig := createKubeConfigContent(t, createKubeConfig(apiEndpoint, operatorNs, "mycooltoken", insecureTls))

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secName,
			Namespace: tcNs,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeConfig,
		},
	}

	return &toolchainv1alpha1.ToolchainCluster{
		Spec: toolchainv1alpha1.ToolchainClusterSpec{
			SecretRef: toolchainv1alpha1.LocalSecretReference{
				Name: secName,
			},
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: tcNs,
		},
		Status: status,
	}, secret
}

func NewClusterStatus(conType toolchainv1alpha1.ConditionType, conStatus corev1.ConditionStatus) toolchainv1alpha1.ToolchainClusterStatus {
	return toolchainv1alpha1.ToolchainClusterStatus{
		Conditions: []toolchainv1alpha1.Condition{{
			Type:   conType,
			Status: conStatus,
		}},
	}
}

func createKubeConfig(apiEndpoint, namespace, token string, insecureTls bool) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"cluster": {
				Server:                apiEndpoint,
				InsecureSkipTLSVerify: insecureTls,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"auth": {
				Token: token,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"ctx": {
				AuthInfo:  "auth",
				Cluster:   "cluster",
				Namespace: namespace,
			},
		},
		CurrentContext: "ctx",
	}
}

func createKubeConfigContent(t *testing.T, kubeConfig *clientcmdapi.Config) []byte {
	data, err := clientcmd.Write(*kubeConfig)
	require.NoError(t, err)
	return data
}
