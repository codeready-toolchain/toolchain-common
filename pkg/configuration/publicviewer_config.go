package configuration

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
)

type PublicViewerConfig struct {
	Config toolchainv1alpha1.PublicViewerConfig
}

func (c PublicViewerConfig) Enabled() bool {
	return c.Config.Enabled
}

func (c PublicViewerConfig) Username() string {
	return c.Config.Username
}

func (c PublicViewerConfig) IsPublicViewer(username string) bool {
	if !c.Enabled() {
		return false
	}

	return c.Config.Username == username
}
