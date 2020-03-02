package cluster

import (
	"context"
	"fmt"
	"testing"

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
)

func TestEnsureKubeFedClusterCrd(t *testing.T) {
	// given
	s := addToScheme(t)
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	expectedCrd := &v1beta1.CustomResourceDefinition{}
	crd, err := Asset("core.kubefed.io_kubefedclusters.yaml")
	require.NoError(t, err)
	_, _, err = decoder.Decode([]byte(crd), nil, expectedCrd)
	require.NoError(t, err)

	t.Run("successful", func(t *testing.T) {
		t.Run("should create the KubeFedCluster CRD", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)

			// when
			err = EnsureKubeFedClusterCRD(s, cl)

			// then
			require.NoError(t, err)
			assertThatKubeFedClusterCrdExists(t, cl, expectedCrd)
		})

		t.Run("should only check the KubeFedCluster CRD as it already exists", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t, expectedCrd)

			// when
			err = EnsureKubeFedClusterCRD(s, cl)

			// then
			require.NoError(t, err)
			assertThatKubeFedClusterCrdExists(t, cl, expectedCrd)
		})
	})

	t.Run("should fail when creating CRD", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t)
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return fmt.Errorf("error")
		}

		// when
		err = EnsureKubeFedClusterCRD(s, cl)

		// then
		require.Error(t, err)
		assert.Equal(t, "unable to create the KubeFedCluster CRD: error", err.Error())
	})
}

func assertThatKubeFedClusterCrdExists(t *testing.T, client client.Client, expectedCrd *v1beta1.CustomResourceDefinition) {
	crd := &v1beta1.CustomResourceDefinition{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: "kubefedclusters.core.kubefed.io"}, crd)
	require.NoError(t, err)
	assert.Equal(t, expectedCrd.TypeMeta, crd.TypeMeta)
	assert.Equal(t, expectedCrd.ObjectMeta.Name, crd.ObjectMeta.Name)
	assert.Equal(t, expectedCrd.ObjectMeta.Annotations, crd.ObjectMeta.Annotations)
	assert.Equal(t, expectedCrd.ObjectMeta.Labels, crd.ObjectMeta.Labels)
	assert.Equal(t, expectedCrd.Spec.AdditionalPrinterColumns, crd.Spec.AdditionalPrinterColumns)
	assert.Equal(t, expectedCrd.Spec.Group, crd.Spec.Group)
	assert.Equal(t, expectedCrd.Spec.Versions, crd.Spec.Versions)
	assert.Equal(t, expectedCrd.Spec.Names, crd.Spec.Names)
	assert.Equal(t, expectedCrd.Spec.Scope, crd.Spec.Scope)
	assert.Equal(t, expectedCrd.Spec.Subresources, crd.Spec.Subresources)
	assert.Equal(t, expectedCrd.Spec.Validation, crd.Spec.Validation)
}

func addToScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	addToSchemes := append(apis.AddToSchemes, v1beta1.SchemeBuilder.AddToScheme)
	err := addToSchemes.AddToScheme(s)
	require.NoError(t, err)
	return s
}
