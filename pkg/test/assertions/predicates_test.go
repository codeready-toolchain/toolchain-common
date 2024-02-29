package assertions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNamePredicate(t *testing.T) {
	pred := &named{name: "expected"}

	t.Run("positive", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("expected")

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("negative", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("different")

		// when & then
		assert.False(t, pred.Matches(obj))
	})

	t.Run("fix", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("different")

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, "expected", obj.Name)
	})
}

func TestInNamespacePredicate(t *testing.T) {
	pred := &inNamespace{namespace: "expected"}

	t.Run("positive", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetNamespace("expected")

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("negative", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetNamespace("different")

		// when & then
		assert.False(t, pred.Matches(obj))
	})

	t.Run("fix", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetNamespace("different")

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, "expected", obj.Namespace)
	})
}

func TestWithKeyPredicate(t *testing.T) {
	pred := &withKey{NamespacedName: client.ObjectKey{Name: "expected", Namespace: "expected"}}

	t.Run("positive", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("expected")
		obj.SetNamespace("expected")

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("different name", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("different")
		obj.SetNamespace("expected")

		// when & then
		assert.False(t, pred.Matches(obj))
	})

	t.Run("different namespace", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("expected")
		obj.SetNamespace("different")

		// when & then
		assert.False(t, pred.Matches(obj))
	})

	t.Run("fix name", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("different")
		obj.SetNamespace("expected")

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, "expected", obj.Name)
		assert.Equal(t, "expected", obj.Namespace)
	})

	t.Run("fix namespace", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("expected")
		obj.SetNamespace("difference")

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, "expected", obj.Name)
		assert.Equal(t, "expected", obj.Namespace)
	})

	t.Run("fix both", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetName("different")
		obj.SetNamespace("difference")

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, "expected", obj.Name)
		assert.Equal(t, "expected", obj.Namespace)
	})
}

func TestLabelsPredicate(t *testing.T) {
	expectedLabels := map[string]string{"ka": "va", "kb": "vb"}
	pred := &hasLabels{requiredLabels: expectedLabels}

	t.Run("exact match", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(map[string]string{"ka": "va", "kb": "vb"})

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("subset match", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(map[string]string{"ka": "va", "kb": "vb", "kc": "vc"})

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("nil", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(nil)

		// when & then
		assert.False(t, pred.Matches(obj))
	})

	t.Run("fix nil labels", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(nil)

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, expectedLabels, obj.GetLabels())
	})
	t.Run("fix empty labels", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(map[string]string{})

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, expectedLabels, obj.GetLabels())
	})
	t.Run("fix different labels", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(map[string]string{"kd": "vd"})

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Len(t, obj.GetLabels(), 3)
		assert.Equal(t, "va", obj.GetLabels()["ka"])
		assert.Equal(t, "vb", obj.GetLabels()["kb"])
		assert.Equal(t, "vd", obj.GetLabels()["kd"])
	})
	t.Run("fix partially matching labels", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetLabels(map[string]string{"ka": "va", "kb": "different", "kd": "vd"})

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Len(t, obj.GetLabels(), 3)
		assert.Equal(t, "va", obj.GetLabels()["ka"])
		assert.Equal(t, "vb", obj.GetLabels()["kb"])
		assert.Equal(t, "vd", obj.GetLabels()["kd"])
	})
}

func TestAnnotationsPredicate(t *testing.T) {
	expectedAnnotations := map[string]string{"ka": "va", "kb": "vb"}
	pred := &hasAnnotations{expectedAnnotations}

	t.Run("exact match", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(map[string]string{"ka": "va", "kb": "vb"})

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("subset match", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(map[string]string{"ka": "va", "kb": "vb", "kc": "vc"})

		// when & then
		assert.True(t, pred.Matches(obj))
	})

	t.Run("nil", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(nil)

		// when & then
		assert.False(t, pred.Matches(obj))
	})
	t.Run("fix nil annotations", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(nil)

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, expectedAnnotations, obj.GetAnnotations())
	})
	t.Run("fix empty annotations", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(map[string]string{})

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Equal(t, expectedAnnotations, obj.GetAnnotations())
	})
	t.Run("fix different annotations", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(map[string]string{"kd": "vd"})

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Len(t, obj.GetAnnotations(), 3)
		assert.Equal(t, "va", obj.GetAnnotations()["ka"])
		assert.Equal(t, "vb", obj.GetAnnotations()["kb"])
		assert.Equal(t, "vd", obj.GetAnnotations()["kd"])
	})
	t.Run("fix partially matching annotations", func(t *testing.T) {
		// given
		obj := &corev1.ConfigMap{}
		obj.SetAnnotations(map[string]string{"ka": "va", "kb": "different", "kd": "vd"})

		// when
		obj = pred.FixToMatch(obj).(*corev1.ConfigMap)

		// then
		assert.Len(t, obj.GetAnnotations(), 3)
		assert.Equal(t, "va", obj.GetAnnotations()["ka"])
		assert.Equal(t, "vb", obj.GetAnnotations()["kb"])
		assert.Equal(t, "vd", obj.GetAnnotations()["kd"])
	})
}
