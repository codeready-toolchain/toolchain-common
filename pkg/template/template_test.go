package template_test

import (
	"embed"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
)

//go:embed testTemplates/*
var EFS embed.FS

//go:embed testTemplates/host/*
var hostFS embed.FS

func TestLoadObjectsFromEmbedFS(t *testing.T) {
	t.Run("loads objects recursively from all subdirectories", func(t *testing.T) {
		// when
		objects, err := template.LoadObjectsFromEmbedFS(&EFS, &template.TemplateVariables{Namespace: test.HostOperatorNs})
		// then
		require.NoError(t, err)
		require.NotNil(t, objects)
		// we expect to have 4 objects loaded from the test templates folders
		require.Equal(t, 4, len(objects), "invalid number of expected objects")
	})

	t.Run("variable substitution works", func(t *testing.T) {
		// when
		// we pass only the hostFS directory
		objects, err := template.LoadObjectsFromEmbedFS(&hostFS, &template.TemplateVariables{Namespace: test.HostOperatorNs})
		// then
		// object's in this folder should have variables to be replaced
		require.NoError(t, err)
		require.NotNil(t, objects)
		for _, obj := range objects {
			// for now we only support the Namespace variable.
			// let's make sure it's correctly set on all the objects
			require.Equal(t, test.HostOperatorNs, obj.GetNamespace())
		}
	})

	t.Run("error - when variables are not provided", func(t *testing.T) {
		// when
		// we do not pass required variables for the templates that requires variables
		objects, err := template.LoadObjectsFromEmbedFS(&hostFS, nil)
		// then
		// we should get back an error
		require.Error(t, err)
		require.Nil(t, objects)
	})
}
