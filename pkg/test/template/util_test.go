package template

import (
	"testing"

	"github.com/codeready-toolchain/member-operator/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestDecodeTemplate(t *testing.T) {
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	codecFactory := serializer.NewCodecFactory(s)
	decoder := codecFactory.UniversalDeserializer()

	t.Run("valid_template", func(t *testing.T) {
		// test
		tmpl, err := DecodeTemplate(decoder, validTmplContent)

		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, "mytemplate", tmpl.GetName())
		assert.NotNil(t, tmpl.Objects)
		assert.Equal(t, 1, len(tmpl.Objects))
	})

	t.Run("invalid_template", func(t *testing.T) {
		// test
		tmpl, err := DecodeTemplate(decoder, invalidTmplContent)

		assert.Error(t, err)
		assert.Nil(t, tmpl)
		assert.Contains(t, err.Error(), "unable to decode template")
	})

	t.Run("nil_decoder", func(t *testing.T) {
		// test
		tmpl, err := DecodeTemplate(nil, invalidTmplContent)

		assert.Error(t, err)
		assert.Nil(t, tmpl)
		assert.Contains(t, err.Error(), "decoder cannot be nil")
	})

	t.Run("nil_template", func(t *testing.T) {
		// test
		tmpl, err := DecodeTemplate(decoder, nil)

		assert.Error(t, err)
		assert.Nil(t, tmpl)
		assert.Contains(t, err.Error(), "unable to decode template")
	})

}

var (
	validTmplContent = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: mytemplate
objects:
- apiVersion: v1
  kind: Namespace
  metadata:
    name: mynamespace`)

	invalidTmplContent = []byte(`apiVersion: v1
kind: Template
metadata:
  name: mytemplate`)
)
