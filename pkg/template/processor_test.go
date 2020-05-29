package template_test

import (
	"bytes"
	"fmt"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/codeready-toolchain/api/pkg/apis"
	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test"
	"k8s.io/apimachinery/pkg/api/meta"

	authv1 "github.com/openshift/api/authorization/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestProcess(t *testing.T) {

	user := getNameWithTimestamp("user")
	commit := getNameWithTimestamp("sha")
	defaultCommit := "123abc"

	s := addToScheme(t)
	codecFactory := serializer.NewCodecFactory(s)
	decoder := codecFactory.UniversalDeserializer()

	p := template.NewProcessor(s)

	t.Run("should process template successfully", func(t *testing.T) {
		// given
		values := map[string]string{
			"USERNAME": user,
			"COMMIT":   commit,
		}
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 2)
		// assert namespace
		assertObject(t, expectedObj{
			template: NamespaceObj,
			username: user,
			commit:   commit,
		}, objs[0])
		// assert rolebinding
		assertObject(t, expectedObj{
			template: RolebindingObj,
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
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 2)

		// assert namespace
		assertObject(t, expectedObj{
			template: NamespaceObj,
			username: user,
			commit:   commit,
		}, objs[0])
		// assert rolebinding
		assertObject(t, expectedObj{
			template: RolebindingObj,
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
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 1)
		// assert namespace
		assertObject(t, expectedObj{
			template: NamespaceObj,
			username: user,
			commit:   commit,
		}, objs[0])
	})

	t.Run("should process template with default commit parameter", func(t *testing.T) {
		// given
		values := map[string]string{
			"USERNAME": user,
		}
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
		require.NoError(t, err)

		// when
		objs, err := p.Process(tmpl, values)

		// then
		require.NoError(t, err)
		require.Len(t, objs, 2)
		// assert namespace
		assertObject(t, expectedObj{
			template: NamespaceObj,
			username: user,
			commit:   defaultCommit,
		}, objs[0])
		// assert rolebinding
		assertObject(t, expectedObj{
			template: RolebindingObj,
			username: user,
			commit:   defaultCommit,
		}, objs[1])
	})

	t.Run("should fail because of missing required parameter", func(t *testing.T) {
		// given
		values := make(map[string]string)
		tmpl, err := DecodeTemplate(decoder,
			CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParamWithoutValue, CommitParam)))
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
			tmpl, err := DecodeTemplate(decoder,
				CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
			require.NoError(t, err)

			// when
			objs, err := p.Process(tmpl, values, template.RetainNamespaces)

			// then
			require.NoError(t, err)
			require.Len(t, objs, 1)
			// assert rolebinding
			assertObject(t, expectedObj{
				template: NamespaceObj,
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
			tmpl, err := DecodeTemplate(decoder,
				CreateTemplate(WithObjects(Namespace, RoleBinding), WithParams(UsernameParam, CommitParam)))
			require.NoError(t, err)

			// when
			objs, err := p.Process(tmpl, values, template.RetainAllButNamespaces)

			// then
			require.NoError(t, err)
			require.Len(t, objs, 1)
			// assert namespace
			assertObject(t, expectedObj{
				template: RolebindingObj,
				username: user,
				commit:   commit,
			}, objs[0])
		})

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

func assertObject(t *testing.T, expectedObj expectedObj, actual client.ToolchainObject) {
	expected, err := newObject(string(expectedObj.template), expectedObj.username, expectedObj.commit)
	require.NoError(t, err, "failed to create object from template")
	expMeta, err := meta.Accessor(expected)
	require.NoError(t, err)

	assert.Equal(t, expected, actual.GetObject())
	assert.Equal(t, expected.GetObjectKind().GroupVersionKind(), actual.GetGvk())
	assert.Equal(t, expMeta.GetName(), actual.GetName())
	assert.Equal(t, expMeta.GetNamespace(), actual.GetNamespace())
}

type expectedObj struct {
	template TemplateObject
	username string
	commit   string
}

func newObject(template, username, commit string) (runtime.Unstructured, error) {
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
