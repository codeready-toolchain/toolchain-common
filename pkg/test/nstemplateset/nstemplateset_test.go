package nstemplateset_test

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test/nstemplateset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewNSTemplateSet(t *testing.T) {

	tier := &toolchainv1alpha1.NSTemplateTier{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "toolchain-host-operator",
			Name:      "base1ns",
		},
		Spec: toolchainv1alpha1.NSTemplateTierSpec{
			ClusterResources: &toolchainv1alpha1.NSTemplateTierClusterResources{
				TemplateRef: "basic-clusterresources-123456new",
			},
			Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
				{
					TemplateRef: "basic-dev-123456new",
				},
				{
					TemplateRef: "basic-stage-123456new",
				},
			},
			SpaceRoles: map[string]toolchainv1alpha1.NSTemplateTierSpaceRole{
				"admin": {
					TemplateRef: "basic-admin-123456new",
				},
				"viewer": {
					TemplateRef: "basic-viewer-123456new",
				},
			},
		},
		Status: toolchainv1alpha1.NSTemplateTierStatus{
			Revisions: map[string]string{
				"basic-clusterresources-123456new": "basic-clusterresources-123456new-ttr",
				"basic-dev-123456new":              "basic-dev-123456new-ttr",
				"basic-stage-123456new":            "basic-stage-123456new-ttr",
				"basic-admin-123456new":            "basic-admin-123456new-ttr",
				"basic-viewer-123456new":           "basic-viewer-123456new-ttr",
			},
		},
	}

	t.Run("with default NSTemplateTier", func(t *testing.T) {
		// when
		result := nstemplateset.NewNSTemplateSet("foo")
		// then
		require.NotNil(t, result.Spec.ClusterResources)
		assert.Equal(t, "basic-clusterresources-abcde00", result.Spec.ClusterResources.TemplateRef)
		assert.ElementsMatch(t, []toolchainv1alpha1.NSTemplateSetNamespace{
			{
				TemplateRef: "basic-dev-abcde11",
			},
			{
				TemplateRef: "basic-code-abcde21",
			},
		}, result.Spec.Namespaces)
		assert.Empty(t, result.Spec.SpaceRoles)

	})

	t.Run("with custom NSTemplateTier but no spaceroles", func(t *testing.T) {
		// when
		result := nstemplateset.NewNSTemplateSet("foo", nstemplateset.WithReferencesFor(tier))
		// then
		require.NotNil(t, result.Spec.ClusterResources)
		assert.Equal(t, "basic-clusterresources-123456new-ttr", result.Spec.ClusterResources.TemplateRef)
		assert.ElementsMatch(t, []toolchainv1alpha1.NSTemplateSetNamespace{
			{
				TemplateRef: "basic-dev-123456new-ttr",
			},
			{
				TemplateRef: "basic-stage-123456new-ttr",
			},
		}, result.Spec.Namespaces)
		assert.Empty(t, result.Spec.SpaceRoles)
	})

	t.Run("with custom NSTemplateTier and two users with same role", func(t *testing.T) {
		// when
		result := nstemplateset.NewNSTemplateSet("foo",
			nstemplateset.WithReferencesFor(tier,
				nstemplateset.WithSpaceRole("admin", "john"),
				nstemplateset.WithSpaceRole("admin", "jack"),
			),
		)
		// then
		require.NotNil(t, result.Spec.ClusterResources)
		assert.Equal(t, "basic-clusterresources-123456new-ttr", result.Spec.ClusterResources.TemplateRef)
		assert.ElementsMatch(t, []toolchainv1alpha1.NSTemplateSetNamespace{
			{
				TemplateRef: "basic-dev-123456new-ttr",
			},
			{
				TemplateRef: "basic-stage-123456new-ttr",
			},
		}, result.Spec.Namespaces)
		assert.ElementsMatch(t, []toolchainv1alpha1.NSTemplateSetSpaceRole{
			{
				TemplateRef: "basic-admin-123456new-ttr",
				Usernames:   []string{"john", "jack"},
			},
		}, result.Spec.SpaceRoles)
	})

	t.Run("with custom NSTemplateTier and two users with different roles", func(t *testing.T) {
		// when
		result := nstemplateset.NewNSTemplateSet("foo",
			nstemplateset.WithReferencesFor(tier,
				nstemplateset.WithSpaceRole("admin", "john"),
				nstemplateset.WithSpaceRole("viewer", "jack"),
			),
		)
		// then
		require.NotNil(t, result.Spec.ClusterResources)
		assert.Equal(t, "basic-clusterresources-123456new-ttr", result.Spec.ClusterResources.TemplateRef)
		assert.ElementsMatch(t, []toolchainv1alpha1.NSTemplateSetNamespace{
			{
				TemplateRef: "basic-dev-123456new-ttr",
			},
			{
				TemplateRef: "basic-stage-123456new-ttr",
			},
		}, result.Spec.Namespaces)
		assert.ElementsMatch(t, []toolchainv1alpha1.NSTemplateSetSpaceRole{
			{
				TemplateRef: "basic-admin-123456new-ttr",
				Usernames:   []string{"john"},
			},
			{
				TemplateRef: "basic-viewer-123456new-ttr",
				Usernames:   []string{"jack"},
			},
		}, result.Spec.SpaceRoles)
	})
}
