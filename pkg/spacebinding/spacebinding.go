package spacebinding

import (
	"fmt"
	"hash/crc32"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultSpaceRole = "admin"

func NewSpaceBinding(mur *toolchainv1alpha1.MasterUserRecord, space *toolchainv1alpha1.Space, creator string, options ...Option) *toolchainv1alpha1.SpaceBinding {
	labels := map[string]string{
		toolchainv1alpha1.SpaceCreatorLabelKey:                 creator,
		toolchainv1alpha1.SpaceBindingMasterUserRecordLabelKey: mur.Name,
		toolchainv1alpha1.SpaceBindingSpaceLabelKey:            space.Name,
	}

	sb := &toolchainv1alpha1.SpaceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mur.Namespace,
			Name:      spaceBindingName(space.Name, mur.Name),
			Labels:    labels,
		},
		Spec: toolchainv1alpha1.SpaceBindingSpec{
			MasterUserRecord: mur.Name,
			Space:            space.Name,
			SpaceRole:        defaultSpaceRole,
		},
	}

	for _, option := range options {
		option(sb)
	}

	return sb
}

// spaceBindingName generates a unique name for the SpaceBinding resource to create,
// based on the name of the Space and the name of the associated MasterUserRecord
func spaceBindingName(spaceName, murName string) string {
	c := crc32.Checksum([]byte(fmt.Sprintf("%s-%s", spaceName, murName)), crc32.IEEETable)
	if len(spaceName) > 50 {
		return fmt.Sprintf("%s-%x", spaceName[:50], c)
	}
	return fmt.Sprintf("%s-%x", spaceName, c)
}

type Option func(spacebinding *toolchainv1alpha1.SpaceBinding)

func WithRole(role string) Option {
	return func(sb *toolchainv1alpha1.SpaceBinding) {
		sb.Spec.SpaceRole = role
	}
}
