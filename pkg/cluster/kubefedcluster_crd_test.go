package cluster

import (
	"context"
	"fmt"
	"github.com/codeready-toolchain/api/pkg/apis"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestEnsureKubeFedClusterCrd(t *testing.T) {
	// given
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	expectedCrd := &v1beta1.CustomResourceDefinition{}
	_, _, err = decoder.Decode([]byte(kubeFedClusterCrd), nil, expectedCrd)
	require.NoError(t, err)

	t.Run("successful", func(t *testing.T) {
		t.Run("should create the KubeFedCluster CRD", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)

			// when
			err = EnsureKubeFedClusterCrd(s, cl)

			// then
			require.NoError(t, err)
			assertThatKubeFedClusterCrdExists(t, cl, expectedCrd)
		})

		t.Run("should only check the KubeFedCluster CRD as it already exists", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t, expectedCrd)

			// when
			err = EnsureKubeFedClusterCrd(s, cl)

			// then
			require.NoError(t, err)
			assertThatKubeFedClusterCrdExists(t, cl, expectedCrd)
		})
	})

	t.Run("should fail when creating CRD", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t)
		cl.MockCreate = func(ctx context.Context, obj runtime.Object) error {
			return fmt.Errorf("error")
		}

		// when
		err = EnsureKubeFedClusterCrd(s, cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "unable to create the KubeFedCluster CRD: error", err.Error())
	})
}

func assertThatKubeFedClusterCrdExists(t *testing.T, client client.Client, expectedCrd *v1beta1.CustomResourceDefinition) {
	crd := &v1beta1.CustomResourceDefinition{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: "kubefedclusters.core.kubefed.k8s.io"}, crd)
	require.NoError(t, err)
	assert.Equal(t, expectedCrd, crd)
}
