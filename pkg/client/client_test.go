package client_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/codeready-toolchain/api/pkg/apis"
	applyCl "github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	authv1 "github.com/openshift/api/authorization/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestApplySingle(t *testing.T) {
	// given
	s := addToScheme(t)

	defaultService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registration-service",
			Namespace: "toolchain-host-operator",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"run": "registration-service",
			},
		},
	}

	modifiedService := defaultService.DeepCopyObject().(*corev1.Service)
	modifiedService.Spec.Selector["run"] = "all-services"

	defaultCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registration-service",
			Namespace: "toolchain-host-operator",
		},
		Data: map[string]string{
			"first-param": "first-value",
		},
	}

	modifiedCm := defaultCm.DeepCopyObject().(*corev1.ConfigMap)
	modifiedCm.Data["first-param"] = "second-value"

	t.Run("updates of service object", func(t *testing.T) {
		// given
		namespacedName := types.NamespacedName{Namespace: "toolchain-host-operator", Name: "registration-service"}

		t.Run("when using forceUpdate=true, it should not update when specs are same", func(t *testing.T) {
			// given
			cl, _ := newClient(t, s)
			obj := defaultService.DeepCopy()
			_, err := cl.CreateOrUpdateObject(obj, true, nil)
			require.NoError(t, err)
			originalGeneration := obj.GetGeneration()

			// when updating with the same obj again
			createdOrChanged, err := cl.CreateOrUpdateObject(obj, true, nil)

			// then
			require.NoError(t, err)
			assert.False(t, createdOrChanged) // resource was not updated on the server, so returned value is `false`
			updateGeneration := obj.GetGeneration()
			assert.Equal(t, originalGeneration, updateGeneration)
		})

		t.Run("when using forceUpdate=true, it should update when specs are different", func(t *testing.T) {
			// given
			cl, _ := newClient(t, s)
			obj := defaultService.DeepCopy()
			_, err := cl.CreateOrUpdateObject(obj, true, nil)
			require.NoError(t, err)
			originalGeneration := obj.GetGeneration()

			// when updating with the modified obj
			modifiedObj := modifiedService.DeepCopy()
			modifiedObj.ObjectMeta.Generation = obj.GetGeneration()
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedObj, true, nil)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged) // resource was updated on the server, so returned value if `true`
			updateGeneration := modifiedObj.GetGeneration()
			assert.Equal(t, originalGeneration+1, updateGeneration)
		})

		t.Run("when using forceUpdate=false, it should update when spec is different", func(t *testing.T) {
			// given
			cl, cli := newClient(t, s)
			_, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), false, nil)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
			service := &corev1.Service{}
			err = cli.Get(context.TODO(), namespacedName, service)
			require.NoError(t, err)
			assert.Equal(t, "all-services", service.Spec.Selector["run"])
		})

		t.Run("when using forceUpdate=false, it should NOT update when using same object", func(t *testing.T) {
			// given
			cl, _ := newClient(t, s)
			_, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(defaultService.DeepCopyObject(), false, nil)

			// then
			require.NoError(t, err)
			assert.False(t, createdOrChanged)
		})

		t.Run("when object is missing, it should create it no matter what is set as forceUpdate", func(t *testing.T) {
			// given
			cl, cli := newClient(t, s)
			deployment := &v1.Deployment{}

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), false, deployment)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
			service := &corev1.Service{}
			err = cli.Get(context.TODO(), namespacedName, service)
			require.NoError(t, err)
			assert.Equal(t, "all-services", service.Spec.Selector["run"])
			assert.NotEmpty(t, service.OwnerReferences)
		})

		t.Run("when object cannot be retrieved because of any error, then it should fail", func(t *testing.T) {
			// given
			cl, cli := newClient(t, s)
			cli.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
				return fmt.Errorf("unable to get")
			}

			// when
			createdOrChanged, err := cl.CreateOrUpdateObject(modifiedService.DeepCopyObject(), false, nil)

			// then
			require.Error(t, err)
			assert.False(t, createdOrChanged)
			assert.Contains(t, err.Error(), "unable to get the resource")
		})
	})

	t.Run("when using forceUpdate=false, it should update ConfigMap when data field is different", func(t *testing.T) {
		// given
		cl, cli := newClient(t, s)
		_, err := cl.CreateOrUpdateObject(defaultCm.DeepCopyObject(), true, nil)
		require.NoError(t, err)

		// when
		createdOrChanged, err := cl.CreateOrUpdateObject(modifiedCm.DeepCopyObject(), false, nil)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrChanged)
		configMap := &corev1.ConfigMap{}
		namespacedName := types.NamespacedName{Namespace: "toolchain-host-operator", Name: "registration-service"}
		err = cli.Get(context.TODO(), namespacedName, configMap)
		require.NoError(t, err)
		assert.Equal(t, "second-value", configMap.Data["first-param"])
	})
}

func TestProcessAndApply(t *testing.T) {

	commit := getNameWithTimestamp("sha")
	user := getNameWithTimestamp("user")

	s := addToScheme(t)
	codecFactory := serializer.NewCodecFactory(s)
	decoder := codecFactory.UniversalDeserializer()

	values := map[string]string{
		"USERNAME": user,
		"COMMIT":   commit,
	}

	t.Run("should create namespace alone", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		p := template.NewProcessor(s)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)
		labels := newLabels("", "john", "")

		// when
		createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrUpdated)
		assertNamespaceExists(t, cl, user, labels, commit)
	})

	t.Run("should create role binding alone", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		p := template.NewProcessor(s)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)
		labels := newLabels("basic", "john", "dev")

		// when
		createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrUpdated)
		assertRoleBindingExists(t, cl, user, labels)
	})

	t.Run("should create namespace and role binding", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		p := template.NewProcessor(s)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)
		labels := newLabels("", "john", "dev")

		// when
		createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrUpdated)
		assertNamespaceExists(t, cl, user, labels, commit)
		assertRoleBindingExists(t, cl, user, labels)
	})

	t.Run("should update existing role binding", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			meta, err := meta.Accessor(obj)
			require.NoError(t, err)
			meta.SetResourceVersion("1")
			return cl.Client.Create(ctx, obj, opts...)
		}
		cl.MockUpdate = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
			meta, err := meta.Accessor(obj)
			require.NoError(t, err)
			t.Logf("updating resource of kind %s with version %s\n", obj.GetObjectKind().GroupVersionKind().Kind, meta.GetResourceVersion())
			if obj.GetObjectKind().GroupVersionKind().Kind == "RoleBinding" && meta.GetResourceVersion() != "1" {
				return fmt.Errorf("invalid resource version: %q", meta.GetResourceVersion())
			}
			return cl.Client.Update(ctx, obj, opts...)
		}
		p := template.NewProcessor(s)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)
		witoutType := newLabels("basic", "john", "")

		createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, witoutType)
		require.NoError(t, err)
		assert.True(t, createdOrUpdated)
		assertRoleBindingExists(t, cl, user, witoutType)

		// when rolebinding changes
		tmpl, err = DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBindingWithExtraUser), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err = p.Process(tmpl, values)
		require.NoError(t, err)
		complete := newLabels("advanced", "john", "dev")
		createdOrUpdated, err = applyCl.NewApplyClient(cl, s).Apply(objs, complete)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrUpdated)
		binding := assertRoleBindingExists(t, cl, user, complete)
		require.Len(t, binding.Subjects, 2)
		assert.Equal(t, "User", binding.Subjects[0].Kind)
		assert.Equal(t, user, binding.Subjects[0].Name)
		assert.Equal(t, "User", binding.Subjects[1].Kind)
		assert.Equal(t, "extraUser", binding.Subjects[1].Name)
	})

	t.Run("should not create or update existing namespace and role binding", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		p := template.NewProcessor(s)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)
		labels := newLabels("basic", "john", "dev")
		created, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)
		require.NoError(t, err)
		assert.True(t, created)
		assertNamespaceExists(t, cl, user, labels, commit)
		assertRoleBindingExists(t, cl, user, labels)

		// when apply the same template again
		updated, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)

		// then
		require.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("failures", func(t *testing.T) {

		t.Run("should fail to create template object", func(t *testing.T) {
			// given
			cl := NewFakeClient(t)
			p := template.NewProcessor(s)
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				return errors.New("failed to create resource")
			}
			tmpl, err := DecodeTemplate(decoder,
				CreateTemplate(WithObjects(RoleBinding), WithParams(UsernameParam, CommitParam)))
			require.NoError(t, err)

			// when
			objs, err := p.Process(tmpl, values)
			require.NoError(t, err)
			createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, newLabels("", "", ""))

			// then
			require.Error(t, err)
			assert.False(t, createdOrUpdated)
		})

		t.Run("should fail to update template object", func(t *testing.T) {
			// given
			cl := NewFakeClient(t)
			p := template.NewProcessor(s)
			cl.MockUpdate = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
				return errors.New("failed to update resource")
			}
			tmpl, err := DecodeTemplate(decoder,
				CreateTemplate(WithObjects(RoleBinding), WithParams(UsernameParam, CommitParam)))
			require.NoError(t, err)
			objs, err := p.Process(tmpl, values)
			require.NoError(t, err)
			labels := newLabels("", "", "")
			createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)
			require.NoError(t, err)
			assert.True(t, createdOrUpdated)

			// when
			tmpl, err = DecodeTemplate(decoder,
				CreateTemplate(WithObjects(RoleBindingWithExtraUser), WithParams(UsernameParam, CommitParam)))
			require.NoError(t, err)
			objs, err = p.Process(tmpl, values)
			require.NoError(t, err)
			createdOrUpdated, err = applyCl.NewApplyClient(cl, s).Apply(objs, newLabels("advanced", "john", "dev"))

			// then
			assert.Error(t, err)
			assert.False(t, createdOrUpdated)
		})
	})

	t.Run("should create with extra newLabels and ownerref", func(t *testing.T) {

		// given
		values := map[string]string{
			"USERNAME": user,
			"COMMIT":   commit,
		}
		cl := NewFakeClient(t)
		p := template.NewProcessor(s)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)

		// when adding newLabels and an owner reference
		obj := objs[0]
		meta, err := meta.Accessor(obj.Object)
		require.NoError(t, err)
		meta.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: "crt/v1",
				Kind:       "NSTemplateSet",
				Name:       "foo",
			},
		})
		labels := newLabels("basic", "john", "dev")
		createdOrUpdated, err := applyCl.NewApplyClient(cl, s).Apply(objs, labels)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrUpdated)
		ns := assertNamespaceExists(t, cl, user, labels, commit)
		// verify owner refs
		assert.Equal(t, []metav1.OwnerReference{
			{
				APIVersion: "crt/v1",
				Kind:       "NSTemplateSet",
				Name:       "foo",
			},
		}, ns.OwnerReferences)
	})
}

func assertNamespaceExists(t *testing.T, c client.Client, nsName string, labels map[string]string, version string) corev1.Namespace {
	// check that the namespace was created
	var ns corev1.Namespace
	err := c.Get(context.TODO(), types.NamespacedName{Name: nsName, Namespace: ""}, &ns) // assert namespace is cluster-scoped
	require.NoError(t, err)
	assert.Equal(t, expectedLabels(labels, version), ns.GetLabels())
	return ns
}

func expectedLabels(labels map[string]string, version string) map[string]string {
	expLabels := map[string]string{
		"extra": "something-extra",
	}
	if version != "" {
		expLabels["version"] = version
	}
	for k, v := range labels {
		expLabels[k] = v
	}
	return expLabels
}

func assertRoleBindingExists(t *testing.T, c client.Client, ns string, labels map[string]string) authv1.RoleBinding {
	// check that the rolebinding is created in the namespace
	// (the fake client just records the request but does not perform any consistency check)
	var rb authv1.RoleBinding
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: ns,
		Name:      fmt.Sprintf("%s-edit", ns),
	}, &rb)

	require.NoError(t, err)
	assert.Equal(t, expectedLabels(labels, ""), rb.GetLabels())
	return rb
}

func newLabels(tier, username, nsType string) map[string]string {
	labels := map[string]string{
		"toolchain.dev.openshift.com/provider": "codeready-toolchain",
	}
	if tier != "" {
		labels["toolchain.dev.openshift.com/tier"] = tier
	}
	if username != "" {
		labels["toolchain.dev.openshift.com/owner"] = username
	}
	if nsType != "" {
		labels["toolchain.dev.openshift.com/type"] = nsType
	}
	return labels
}

func getNameWithTimestamp(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func newClient(t *testing.T, s *runtime.Scheme) (*applyCl.ApplyClient, *test.FakeClient) {
	cli := NewFakeClient(t)
	return applyCl.NewApplyClient(cli, s), cli
}

func addToScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := authv1.Install(s)
	require.NoError(t, err)
	err = apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}
