package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestMapToOwnerByLabel(t *testing.T) {

	t.Run("resource with expected label", func(t *testing.T) {
		// given
		objMeta := metav1.ObjectMeta{
			Name: "bar",
			Labels: map[string]string{
				"type":     "che",
				"owner":    "foo",
				"revision": "123",
			},
		}
		obj := &corev1.Namespace{
			ObjectMeta: objMeta,
		}
		// when
		result := MapToOwnerByLabel("ns", "owner")(obj)
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
		result := MapToOwnerByLabel("ns", "owner")(&obj)
		// then
		require.Empty(t, result)
	})
}

func TestMapToControllerByMatchingLabel(t *testing.T) {

	t.Run("resource with expected label and value", func(t *testing.T) {
		// given
		objMeta := metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "ns",
			Labels: map[string]string{
				"type":     "che",
				"owner":    "foo",
				"revision": "123",
			},
		}
		obj := &corev1.Namespace{
			ObjectMeta: objMeta,
		}
		// when
		result := MapToControllerByMatchingLabel("owner", "foo")(obj)
		// then
		require.Len(t, result, 1)
		assert.Equal(t, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "ns",
				Name:      "bar",
			},
		}, result[0])
	})

	t.Run("resource without expected label key", func(t *testing.T) {
		// given
		objMeta := metav1.ObjectMeta{
			Name: "bar",
			Labels: map[string]string{
				"somenoise": "foo",
			},
		}
		obj := corev1.Namespace{
			ObjectMeta: objMeta,
		}
		// when
		result := MapToControllerByMatchingLabel("owner", "foo")(&obj)
		// then
		require.Empty(t, result)
	})

	t.Run("resource without expected label value", func(t *testing.T) {
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
		result := MapToControllerByMatchingLabel("owner", "bar")(&obj)
		// then
		require.Empty(t, result)
	})
}
