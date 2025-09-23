package memberoperatorconfig

import (
	"testing"
	"time"

	commonconfig "github.com/codeready-toolchain/toolchain-common/pkg/configuration"
	testconfig "github.com/codeready-toolchain/toolchain-common/pkg/test/config"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
		memberOperatorCfg := Configuration{cfg: &cfg.Spec}

		assert.Equal(t, "rhd", memberOperatorCfg.Auth().Idp())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Auth().Idp("another"))
		memberOperatorCfg := Configuration{cfg: &cfg.Spec}

		assert.Equal(t, "another", memberOperatorCfg.Auth().Idp())
	})
}

func TestAutoscaler(t *testing.T) {
	t.Run("deploy", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.True(t, memberOperatorCfg.Autoscaler().Deploy())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Autoscaler().Deploy(false))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.False(t, memberOperatorCfg.Autoscaler().Deploy())
		})
	})
	t.Run("buffer memory", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "50Mi", memberOperatorCfg.Autoscaler().BufferMemory())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Autoscaler().BufferMemory("5GiB"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "5GiB", memberOperatorCfg.Autoscaler().BufferMemory())
		})
	})
	t.Run("buffer cpu", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "50m", memberOperatorCfg.Autoscaler().BufferCPU())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Autoscaler().BufferCPU("2000m"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "2000m", memberOperatorCfg.Autoscaler().BufferCPU())
		})
	})
	t.Run("buffer replicas", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 2, memberOperatorCfg.Autoscaler().BufferReplicas())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Autoscaler().BufferReplicas(2))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 2, memberOperatorCfg.Autoscaler().BufferReplicas())
		})
	})
}

func TestGitHubSecret(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
		memberOperatorCfg := Configuration{cfg: &cfg.Spec}

		assert.Empty(t, memberOperatorCfg.GitHubSecret().AccessTokenKey())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.MemberStatus().
			GitHubSecretRef("github").
			GitHubSecretAccessTokenKey("accessToken"))

		gitHubSecretValues := make(map[string]string)
		gitHubSecretValues["accessToken"] = "abc123"
		secrets := make(map[string]map[string]string)
		secrets["github"] = gitHubSecretValues
		memberOperatorCfg := Configuration{cfg: &cfg.Spec, secrets: secrets}

		assert.Equal(t, "abc123", memberOperatorCfg.GitHubSecret().AccessTokenKey())
	})
}

func TestConsole(t *testing.T) {
	t.Run("console namespace", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "openshift-console", memberOperatorCfg.Console().Namespace())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Console().Namespace("another-namespace"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "another-namespace", memberOperatorCfg.Console().Namespace())
		})
	})
	t.Run("console route", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "console", memberOperatorCfg.Console().RouteName())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Console().RouteName("another-route"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, "another-route", memberOperatorCfg.Console().RouteName())
		})
	})
}

func TestMemberStatus(t *testing.T) {
	t.Run("member status refresh period", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 5*time.Second, memberOperatorCfg.MemberStatus().RefreshPeriod())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.MemberStatus().RefreshPeriod("10s"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 10*time.Second, memberOperatorCfg.MemberStatus().RefreshPeriod())
		})
		t.Run("non-default invalid value", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.MemberStatus().RefreshPeriod("10ABC"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 5*time.Second, memberOperatorCfg.MemberStatus().RefreshPeriod())
		})
	})
}

func TestSkipUserCreation(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
		memberOperatorCfg := Configuration{cfg: &cfg.Spec}

		assert.False(t, memberOperatorCfg.SkipUserCreation())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.SkipUserCreation(true))
		memberOperatorCfg := Configuration{cfg: &cfg.Spec}

		assert.True(t, memberOperatorCfg.SkipUserCreation())
	})
}

func TestToolchainCluster(t *testing.T) {
	t.Run("health check period", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 10*time.Second, memberOperatorCfg.ToolchainCluster().HealthCheckPeriod())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.ToolchainCluster().HealthCheckPeriod("3s"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 3*time.Second, memberOperatorCfg.ToolchainCluster().HealthCheckPeriod())
		})
		t.Run("non-default invalid value", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.ToolchainCluster().HealthCheckPeriod("3ABC"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 10*time.Second, memberOperatorCfg.ToolchainCluster().HealthCheckPeriod())
		})
	})
	t.Run("health check timeout", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 3*time.Second, memberOperatorCfg.ToolchainCluster().HealthCheckTimeout())
		})
		t.Run("non-default", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.ToolchainCluster().HealthCheckTimeout("11s"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 11*time.Second, memberOperatorCfg.ToolchainCluster().HealthCheckTimeout())
		})
		t.Run("non-default invalid value", func(t *testing.T) {
			cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.ToolchainCluster().HealthCheckTimeout("11ABC"))
			memberOperatorCfg := Configuration{cfg: &cfg.Spec}

			assert.Equal(t, 3*time.Second, memberOperatorCfg.ToolchainCluster().HealthCheckTimeout())
		})
	})
}

func TestWebhook(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t)
		memberOperatorCfg := Configuration{cfg: &cfg.Spec}

		assert.True(t, memberOperatorCfg.Webhook().Deploy())
		assert.Empty(t, memberOperatorCfg.Webhook().VMSSHKey())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := commonconfig.NewMemberOperatorConfigWithReset(t, testconfig.Webhook().
			Deploy(false).
			WebhookSecretRef("webhook").
			VMSSHKey("vmKey"))

		webhookSecretValues := make(map[string]string)
		webhookSecretValues["vmKey"] = "ssh-rsa abc-123"
		secrets := make(map[string]map[string]string)
		secrets["webhook"] = webhookSecretValues
		memberOperatorCfg := Configuration{cfg: &cfg.Spec, secrets: secrets}

		assert.False(t, memberOperatorCfg.Webhook().Deploy())
		assert.Equal(t, "ssh-rsa abc-123", memberOperatorCfg.Webhook().VMSSHKey())
	})
}
