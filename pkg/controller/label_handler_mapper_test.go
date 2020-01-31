package controller_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/controller"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestLabelMapper(t *testing.T) {

	t.Run("resource with expected label", func(t *testing.T) {
		// given
		objMeta := metav1.ObjectMeta{
			Name: "bar",
			Labels: map[string]string{
				"owner": "foo",
			},
		}
		obj := corev1.Namespace{
			ObjectMeta: objMeta,
		}
		// when
		result := controller.EnqueueRequestForOwnerByLabel{Namespace: "ns", Label: "owner"}.Map(handler.MapObject{
			Meta:   &objMeta,
			Object: &obj,
		})
		// then
		require.Len(t, result, 1)
		assert.Equal(t, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "ns",
				Name:      "foo",
			},
		}, result[0])
	})

	t.Run("resource without expected label", func(t *testing.T) {
		// given
		objMeta := metav1.ObjectMeta{
			Name: "bar",
			Labels: map[string]string{
				"somethingelse": "foo",
			},
		}
		obj := corev1.Namespace{
			ObjectMeta: objMeta,
		}
		// when
		result := controller.EnqueueRequestForOwnerByLabel{Namespace: "ns", Label: "owner"}.Map(handler.MapObject{
			Meta:   &objMeta,
			Object: &obj,
		})
		// then
		require.Empty(t, result)
	})
}
