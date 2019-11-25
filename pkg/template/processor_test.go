package template_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"k8s.io/api/apps/v1"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/codeready-toolchain/api/pkg/apis"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	authv1 "github.com/openshift/api/authorization/v1"
	templatev1 "github.com/openshift/api/template/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestProcess(t *testing.T) {

	user := getNameWithTimestamp("user")
	commit := getNameWithTimestamp("sha")
	defaultCommit := "123abc"

	s := addToScheme(t)
	codecFactory := serializer.NewCodecFactory(s)
	decoder := codecFactory.UniversalDeserializer()

	cl := test.NewFakeClient(t)
	p := template.NewProcessor(cl, s)

	t.Run("should process template successfully", func(t *testing.T) {
		// given
		values := map[string]string{
			"USERNAME": user,
			"COMMIT":   commit,
		}
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 2)
		// assert namespace
		assertObject(t, expectedObj{
			template: namespaceObj,
			username: user,
			commit:   commit,
		}, objs[0])
		// assert rolebinding
		assertObject(t, expectedObj{
			template: rolebindingObj,
			username: user,
			commit:   commit,
		}, objs[1])

	})

	t.Run("should overwrite default value of commit parameter", func(t *testing.T) {
		// given
		values := map[string]string{
			"USERNAME": user,
			"COMMIT":   commit,
		}
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 2)

		// assert namespace
		assertObject(t, expectedObj{
			template: namespaceObj,
			username: user,
			commit:   commit,
		}, objs[0])
		// assert rolebinding
		assertObject(t, expectedObj{
			template: rolebindingObj,
			username: user,
			commit:   commit,
		}, objs[1])
	})

	t.Run("should not fail for random extra param", func(t *testing.T) {
		// given
		random := getNameWithTimestamp("random")
		values := map[string]string{
			"USERNAME": user,
			"COMMIT":   commit,
			"random":   random, // extra, unused param
		}
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace), withParams(usernameParam, commitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 1)
		// assert namespace
		assertObject(t, expectedObj{
			template: namespaceObj,
			username: user,
			commit:   commit,
		}, objs[0])
	})

	t.Run("should process template with default commit parameter", func(t *testing.T) {
		// given
		values := map[string]string{
			"USERNAME": user,
		}
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 2)
		// assert namespace
		assertObject(t, expectedObj{
			template: namespaceObj,
			username: user,
			commit:   defaultCommit,
		}, objs[0])
		// assert rolebinding
		assertObject(t, expectedObj{
			template: rolebindingObj,
			username: user,
			commit:   defaultCommit,
		}, objs[1])
	})

	t.Run("should fail because of missing required parameter", func(t *testing.T) {
		// given
		values := make(map[string]string)
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebinding), withParams(usernameParamWithoutValue, commitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.Error(t, err, "fail to process as not providing required param USERNAME")
		assert.Nil(t, objs)
	})

	t.Run("filter results", func(t *testing.T) {

		t.Run("return namespace", func(t *testing.T) {
			// given
			values := map[string]string{
				"USERNAME": user,
			}
			tmpl, err := decodeTemplate(decoder,
				createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
			require.NoError(t, err)

			// when
			objs, err := p.Process(tmpl, values, template.RetainNamespaces)

			// then
			require.NoError(t, err)
			require.Len(t, objs, 1)
			// assert rolebinding
			assertObject(t, expectedObj{
				template: namespaceObj,
				username: user,
				commit:   defaultCommit,
			}, objs[0])
		})

		t.Run("return other resources", func(t *testing.T) {
			// given
			values := map[string]string{
				"USERNAME": user,
				"COMMIT":   commit,
			}
			tmpl, err := decodeTemplate(decoder,
				createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
			require.NoError(t, err)

			// when
			objs, err := p.Process(tmpl, values, template.RetainAllButNamespaces)

			// then
			require.NoError(t, err)
			require.Len(t, objs, 1)
			// assert namespace
			assertObject(t, expectedObj{
				template: rolebindingObj,
				username: user,
				commit:   commit,
			}, objs[0])
		})

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
		cl := test.NewFakeClient(t)
		p := template.NewProcessor(cl, s)
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace), withParams(usernameParam, commitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)

		// when
		err = p.Apply(objs)

		// then
		require.NoError(t, err)
		assertNamespaceExists(t, cl, user)
	})

	t.Run("should create role binding alone", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t)
		p := template.NewProcessor(cl, s)
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)

		// when
		err = p.Apply(objs)

		// then
		require.NoError(t, err)
		assertRoleBindingExists(t, cl, user)
	})

	t.Run("should create namespace and role binding", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t)
		p := template.NewProcessor(cl, s)
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)

		// when
		err = p.Apply(objs)

		// then
		require.NoError(t, err)
		assertNamespaceExists(t, cl, user)
		assertRoleBindingExists(t, cl, user)

	})

	t.Run("should update existing role binding", func(t *testing.T) {
		// given
		cl := test.NewFakeClient(t)
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			if rb, ok := obj.(*unstructured.Unstructured); ok {
				rb.SetResourceVersion("1")
				return cl.Client.Create(ctx, obj, opts...)
			}
			return cl.Client.Create(ctx, obj, opts...)
		}
		cl.MockUpdate = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
			if u, ok := obj.(*unstructured.Unstructured); ok {
				t.Logf("updating resource of kind %s with version %s\n", u.GetKind(), u.GetResourceVersion())
				if u.GetKind() == "RoleBinding" && u.GetResourceVersion() != "1" {
					return fmt.Errorf("invalid resource version: %q", u.GetResourceVersion())
				}
			}
			return cl.Client.Update(ctx, obj, opts...)
		}
		p := template.NewProcessor(cl, s)
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)
		err = p.Apply(objs)
		require.NoError(t, err)
		assertRoleBindingExists(t, cl, user)

		// when rolebinding changes
		tmpl, err = decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebindingWithExtraUser), withParams(usernameParam, commitParam)))
		require.NoError(t, err)
		objs, err = p.Process(tmpl, values)
		require.NoError(t, err)
		err = p.Apply(objs)

		// then
		require.NoError(t, err)
		binding := assertRoleBindingExists(t, cl, user)
		require.Len(t, binding.Subjects, 2)
		assert.Equal(t, "User", binding.Subjects[0].Kind)
		assert.Equal(t, user, binding.Subjects[0].Name)
		assert.Equal(t, "User", binding.Subjects[1].Kind)
		assert.Equal(t, "extraUser", binding.Subjects[1].Name)
	})

	t.Run("failures", func(t *testing.T) {

		t.Run("should fail to create template object", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)
			p := template.NewProcessor(cl, s)
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				return errors.New("failed to create resource")
			}
			tmpl, err := decodeTemplate(decoder,
				createTemplate(withObjects(rolebinding), withParams(usernameParam, commitParam)))
			require.NoError(t, err)

			// when
			objs, err := p.Process(tmpl, values)
			require.NoError(t, err)
			err = p.Apply(objs)

			// then
			require.Error(t, err)
		})

		t.Run("should fail to update template object", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)
			p := template.NewProcessor(cl, s)
			cl.MockUpdate = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
				return errors.New("failed to update resource")
			}
			tmpl, err := decodeTemplate(decoder,
				createTemplate(withObjects(rolebinding), withParams(usernameParam, commitParam)))
			require.NoError(t, err)
			objs, err := p.Process(tmpl, values)
			require.NoError(t, err)
			err = p.Apply(objs)
			require.NoError(t, err)

			// when
			tmpl, err = decodeTemplate(decoder,
				createTemplate(withObjects(rolebindingWithExtraUser), withParams(usernameParam, commitParam)))
			require.NoError(t, err)
			objs, err = p.Process(tmpl, values)
			require.NoError(t, err)
			err = p.Apply(objs)

			// then
			assert.Error(t, err)
		})
	})

	t.Run("should create with extra labels and ownerref", func(t *testing.T) {

		// given
		values := map[string]string{
			"USERNAME": user,
			"COMMIT":   commit,
		}
		cl := test.NewFakeClient(t)
		p := template.NewProcessor(cl, s)
		tmpl, err := decodeTemplate(decoder,
			createTemplate(withObjects(namespace, rolebinding), withParams(usernameParam, commitParam)))
		require.NoError(t, err)
		objs, err := p.Process(tmpl, values)
		require.NoError(t, err)

		// when adding labels and an owner reference
		obj := objs[0]
		nsObj, ok := obj.Object.(*unstructured.Unstructured)
		require.True(t, ok)
		nsObj.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: "crt/v1",
				Kind:       "NSTemplateSet",
				Name:       "foo",
			},
		})
		nsObj.SetLabels(map[string]string{
			"provider": "codeready-toolchain",
			"version":  commit,
			"extra":    "foo",
		})
		err = p.Apply(objs)

		// then
		require.NoError(t, err)
		ns := assertNamespaceExists(t, cl, user)
		// verify labels
		assert.Equal(t, map[string]string{
			"provider": "codeready-toolchain",
			"version":  commit,
			"extra":    "foo",
		}, ns.Labels)
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

func TestApplySingle(t *testing.T) {
	// given
	s := addToScheme(t)
	codecFactory := serializer.NewCodecFactory(s)
	decoder := codecFactory.UniversalDeserializer()

	// create default objects with default params
	cl := test.NewFakeClient(t)
	p := template.NewProcessor(cl, s)
	tmpl, err := decodeTemplate(decoder,
		createTemplate(withObjects(service, configMap), withParams(namespaceParam, servSelectorParam)))
	require.NoError(t, err)
	defaultObjs, err := p.Process(tmpl, map[string]string{})
	require.NoError(t, err)
	defaultObject := defaultObjs[0].Object

	// create objects containing modified params
	tmpl, err = decodeTemplate(decoder,
		createTemplate(withObjects(service, configMap), withParams(namespaceParam, servSelectorParam)))
	require.NoError(t, err)
	values := map[string]string{
		"NAMESPACE":        "toolchain-host-operator",
		"SERVICE_SELECTOR": "all-services",
	}
	modifiedObjs, err := p.Process(tmpl, values)

	t.Run("updates of service object", func(t *testing.T) {
		// given
		namespacedName := types.NamespacedName{Namespace: "toolchain-host-operator", Name: "registration-service"}

		t.Run("when using forceUpdate=true, it should update even when spec is same", func(t *testing.T) {
			// given
			cli := test.NewFakeClient(t)
			processor := template.NewProcessor(cli, s)
			_, err := processor.ApplySingle(defaultObject.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := processor.ApplySingle(modifiedObjs[0].Object, true, nil)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
		})

		t.Run("when using forceUpdate=false, it should update when spec is different", func(t *testing.T) {
			// given
			cli := test.NewFakeClient(t)
			processor := template.NewProcessor(cli, s)
			_, err := processor.ApplySingle(defaultObject.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := processor.ApplySingle(modifiedObjs[0].Object, false, nil)

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
			cli := test.NewFakeClient(t)
			processor := template.NewProcessor(cli, s)
			_, err := processor.ApplySingle(defaultObject.DeepCopyObject(), true, nil)
			require.NoError(t, err)

			// when
			createdOrChanged, err := processor.ApplySingle(defaultObject, false, nil)

			// then
			require.NoError(t, err)
			assert.False(t, createdOrChanged)
		})

		t.Run("when object is missing, it should create it no matter what is set as forceUpdate", func(t *testing.T) {
			// given
			cli := test.NewFakeClient(t)
			processor := template.NewProcessor(cli, s)
			require.NoError(t, err)
			deployment := &v1.Deployment{}

			// when
			createdOrChanged, err := processor.ApplySingle(modifiedObjs[0].Object, false, deployment)

			// then
			require.NoError(t, err)
			assert.True(t, createdOrChanged)
			service := &corev1.Service{}
			err = cli.Get(context.TODO(), namespacedName, service)
			require.NoError(t, err)
			assert.Equal(t, "all-services", service.Spec.Selector["run"])
			assert.NotEmpty(t, service.OwnerReferences)
		})
	})

	t.Run("when using forceUpdate=false, it should update ConfigMap when data field is different", func(t *testing.T) {
		// given
		cli := test.NewFakeClient(t)
		processor := template.NewProcessor(cli, s)
		_, err := processor.ApplySingle(defaultObjs[1].Object, true, nil)
		require.NoError(t, err)

		// when
		createdOrChanged, err := processor.ApplySingle(modifiedObjs[1].Object, false, nil)

		// then
		require.NoError(t, err)
		assert.True(t, createdOrChanged)
		configMap := &corev1.ConfigMap{}
		namespacedName := types.NamespacedName{Namespace: "toolchain-host-operator", Name: "registration-service"}
		err = cli.Get(context.TODO(), namespacedName, configMap)
		require.NoError(t, err)
		assert.Equal(t, "all-services", configMap.Data["service-selector"])
	})
}

func addToScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := authv1.Install(s)
	require.NoError(t, err)
	err = apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}

func getNameWithTimestamp(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func assertObject(t *testing.T, expectedObj expectedObj, actual runtime.RawExtension) {
	// objJson, err := actual.Marshal()
	// require.NoError(t, err, "failed to marshal json from unstructured object")
	expected, err := newObject(expectedObj.template, expectedObj.username, expectedObj.commit)
	require.NoError(t, err, "failed to create object from template")
	assert.Equal(t, expected, actual.Object)
}

type expectedObj struct {
	template string
	username string
	commit   string
}

func newObject(template, username, commit string) (*unstructured.Unstructured, error) {
	tmpl := texttemplate.New("")
	tmpl, err := tmpl.Parse(template)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, struct {
		Username string
		Commit   string
	}{
		Username: username,
		Commit:   commit,
	})
	if err != nil {
		return nil, err
	}
	result := &unstructured.Unstructured{}
	err = result.UnmarshalJSON(buf.Bytes())
	return result, err
}

func assertNamespaceExists(t *testing.T, c client.Client, nsName string) corev1.Namespace {
	// check that the namespace was created
	var ns corev1.Namespace
	err := c.Get(context.TODO(), types.NamespacedName{Name: nsName, Namespace: ""}, &ns) // assert namespace is cluster-scoped
	require.NoError(t, err)
	return ns
}

func assertRoleBindingExists(t *testing.T, c client.Client, ns string) authv1.RoleBinding {
	// check that the rolebinding is created in the namespace
	// (the fake client just records the request but does not perform any consistency check)
	var rb authv1.RoleBinding
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: ns,
		Name:      fmt.Sprintf("%s-edit", ns),
	}, &rb)

	require.NoError(t, err)
	return rb
}

func decodeTemplate(decoder runtime.Decoder, tmplContent string) (*templatev1.Template, error) {
	tmpl := &templatev1.Template{}
	_, _, err := decoder.Decode([]byte(tmplContent), nil, tmpl)
	if err != nil {
		return nil, err
	}
	return tmpl, err
}

func createTemplate(innerObjects templateObject, params templateParams) string {
	return fmt.Sprintf(templateSkeleton, innerObjects, params)
}

type templateParams string
type templateObject string

func withParams(parameters ...string) templateParams {
	params := ""
	for _, param := range parameters {
		params += param
	}
	return templateParams(params)
}

func withObjects(innerObjects ...string) templateObject {
	objects := ""
	for _, object := range innerObjects {
		objects += object
	}
	return templateObject(objects)
}

const (
	templateSkeleton = `apiVersion: template.openshift.io/v1
kind: Template
metadata:
  labels:
    provider: codeready-toolchain
    version: ${COMMIT}
  name: basic-tier-template
objects:
%s
parameters:
%s
`
	usernameParam = `
- name: USERNAME
  value: toolchain-dev
  required: true`

	usernameParamWithoutValue = `
- name: USERNAME
  required: true`

	commitParam = `
- name: COMMIT
  value: 123abc
  required: true`

	namespaceParam = `
- name: NAMESPACE
  value: toolchain-host-operator`

	servSelectorParam = `
- name: SERVICE_SELECTOR
  value: registration-service`

	namespace = `
- apiVersion: v1
  kind: Namespace
  metadata:
    annotations:
      openshift.io/description: ${USERNAME}-user
      openshift.io/display-name: ${USERNAME}
      openshift.io/requester: ${USERNAME}
    labels:
      provider: codeready-toolchain
      version: ${COMMIT}
    name: ${USERNAME}`

	rolebinding = `
- apiVersion: authorization.openshift.io/v1
  kind: RoleBinding
  metadata:
    name: ${USERNAME}-edit
    namespace: ${USERNAME}
  roleRef:
    kind: ClusterRole
    name: edit
  subjects:
  - kind: User
    name: ${USERNAME}`

	rolebindingWithExtraUser = `
- apiVersion: authorization.openshift.io/v1
  kind: RoleBinding
  metadata:
    name: ${USERNAME}-edit
    namespace: ${USERNAME}
  roleRef:
    kind: ClusterRole
    name: edit
  subjects:
  - kind: User
    name: ${USERNAME}
  - kind: User
    name: extraUser`

	service = `
- kind: Service
  apiVersion: v1
  metadata:
    name: registration-service
    namespace: ${NAMESPACE}
    labels:
      provider: codeready-toolchain
      run: registration-service
  spec:
    selector:
      run: ${SERVICE_SELECTOR}`

	configMap = `
- kind: ConfigMap
  apiVersion: v1
  metadata:
    labels:
      provider: codeready-toolchain
    name: registration-service
    namespace: ${NAMESPACE}
  type: Opaque
  data:
    service-selector: ${SERVICE_SELECTOR}`

	namespaceObj = `{ 
	"apiVersion": "v1",
	"kind": "Namespace",
	"metadata": {
		"annotations": {
			"openshift.io/description": "{{ .Username }}-user",
			"openshift.io/display-name": "{{ .Username }}",
			"openshift.io/requester": "{{ .Username }}"
		},
		"labels": {
			"provider": "codeready-toolchain",
			"version": "{{ .Commit }}"
		},
		"name": "{{ .Username }}"
	}
}`

	rolebindingObj = `{
	"apiVersion": "authorization.openshift.io/v1",
	"kind": "RoleBinding",
	"metadata": {
		"name": "{{ .Username }}-edit",
    	"namespace": "{{ .Username }}"
	},
	"roleRef": {
		"kind": "ClusterRole",
		"name": "edit"
	},
	"subjects": [
		{
			"kind": "User",
			"name": "{{ .Username }}"
		}
	]
}`
)
