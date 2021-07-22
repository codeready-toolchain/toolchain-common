package toolchainconfig

import (
	"testing"
	"time"

	testconfig "github.com/codeready-toolchain/toolchain-common/pkg/test/config"

	"github.com/stretchr/testify/assert"
)

func TestAutomaticApprovalConfig(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.False(t, toolchainCfg.AutomaticApproval().IsEnabled())
		assert.Equal(t, 1000, toolchainCfg.AutomaticApproval().MaxNumberOfUsersOverall())
		assert.Empty(t, toolchainCfg.AutomaticApproval().MaxNumberOfUsersSpecificPerMemberCluster())
		assert.Equal(t, 80, toolchainCfg.AutomaticApproval().ResourceCapacityThresholdDefault())
		assert.Empty(t, toolchainCfg.AutomaticApproval().ResourceCapacityThresholdSpecificPerMemberCluster())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.AutomaticApproval().Enabled(true).MaxNumberOfUsers(123, testconfig.PerMemberCluster("member1", 321)).ResourceCapacityThreshold(456, testconfig.PerMemberCluster("member1", 654)))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.True(t, toolchainCfg.AutomaticApproval().IsEnabled())
		assert.Equal(t, 123, toolchainCfg.AutomaticApproval().MaxNumberOfUsersOverall())
		assert.Equal(t, cfg.Spec.Host.AutomaticApproval.MaxNumberOfUsers.SpecificPerMemberCluster, toolchainCfg.AutomaticApproval().MaxNumberOfUsersSpecificPerMemberCluster())
		assert.Equal(t, 456, toolchainCfg.AutomaticApproval().ResourceCapacityThresholdDefault())
		assert.Equal(t, cfg.Spec.Host.AutomaticApproval.ResourceCapacityThreshold.SpecificPerMemberCluster, toolchainCfg.AutomaticApproval().ResourceCapacityThresholdSpecificPerMemberCluster())
	})
}

func TestDeactivationConfig(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 3, toolchainCfg.Deactivation().DeactivatingNotificationDays())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.Deactivation().DeactivatingNotificationDays(5))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 5, toolchainCfg.Deactivation().DeactivatingNotificationDays())
	})
}

func TestEnvironment(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, "prod", toolchainCfg.Environment())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.Environment(testconfig.E2E))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, "e2e-tests", toolchainCfg.Environment())
	})
}

func TestMetrics(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.False(t, toolchainCfg.Metrics().ForceSynchronization())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.Metrics().ForceSynchronization(true))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.True(t, toolchainCfg.Metrics().ForceSynchronization())
	})
}

func TestNotifications(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Empty(t, toolchainCfg.Notifications().AdminEmail())
		assert.Empty(t, toolchainCfg.Notifications().MailgunDomain())
		assert.Empty(t, toolchainCfg.Notifications().MailgunAPIKey())
		assert.Empty(t, toolchainCfg.Notifications().MailgunSenderEmail())
		assert.Empty(t, toolchainCfg.Notifications().MailgunReplyToEmail())
		assert.Equal(t, "mailgun", toolchainCfg.Notifications().NotificationDeliveryService())
		assert.Equal(t, 24*time.Hour, toolchainCfg.Notifications().DurationBeforeNotificationDeletion())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t,
			testconfig.Notifications().
				AdminEmail("joe.schmoe@redhat.com").
				DurationBeforeNotificationDeletion("48h").
				NotificationDeliveryService("mailknife").
				Secret().
				Ref("notifications").
				MailgunAPIKey("mailgunAPIKey").
				MailgunDomain("mailgunDomain").
				MailgunReplyToEmail("replyTo").
				MailgunSenderEmail("sender"))
		notificationSecretValues := make(map[string]string)
		notificationSecretValues["mailgunAPIKey"] = "abc123"
		notificationSecretValues["mailgunDomain"] = "domain.abc"
		notificationSecretValues["replyTo"] = "devsandbox_rulez@redhat.com"
		notificationSecretValues["sender"] = "devsandbox@redhat.com"
		secrets := make(map[string]map[string]string)
		secrets["notifications"] = notificationSecretValues

		toolchainCfg := NewToolchainConfig(&cfg.Spec, secrets)

		assert.Equal(t, "joe.schmoe@redhat.com", toolchainCfg.Notifications().AdminEmail())
		assert.Equal(t, "abc123", toolchainCfg.Notifications().MailgunAPIKey())
		assert.Equal(t, "domain.abc", toolchainCfg.Notifications().MailgunDomain())
		assert.Equal(t, "devsandbox_rulez@redhat.com", toolchainCfg.Notifications().MailgunReplyToEmail())
		assert.Equal(t, "devsandbox@redhat.com", toolchainCfg.Notifications().MailgunSenderEmail())
		assert.Equal(t, "mailknife", toolchainCfg.Notifications().NotificationDeliveryService())
		assert.Equal(t, 48*time.Hour, toolchainCfg.Notifications().DurationBeforeNotificationDeletion())
	})
}

func TestRegistrationService(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, "prod", toolchainCfg.RegistrationService().Environment())
		assert.Equal(t, "info", toolchainCfg.RegistrationService().LogLevel())
		assert.Equal(t, "toolchain-host-operator", toolchainCfg.RegistrationService().Namespace())
		assert.Equal(t, "https://registration.crt-placeholder.com", toolchainCfg.RegistrationService().RegistrationServiceURL())
		assert.Empty(t, toolchainCfg.RegistrationService().Analytics().SegmentWriteKey())
		assert.Empty(t, toolchainCfg.RegistrationService().Analytics().WoopraDomain())
		assert.Equal(t, "https://sso.prod-preview.openshift.io/auth/js/keycloak.js", toolchainCfg.RegistrationService().Auth().AuthClientLibraryURL())
		assert.Equal(t, "application/json; charset=utf-8", toolchainCfg.RegistrationService().Auth().AuthClientConfigContentType())
		assert.Equal(t, `{"realm": "toolchain-public","auth-server-url": "https://sso.prod-preview.openshift.io/auth","ssl-required": "none","resource": "crt","clientId": "crt","public-client": true}`,
			toolchainCfg.RegistrationService().Auth().AuthClientConfigRaw())
		assert.Equal(t, "https://sso.prod-preview.openshift.io/auth/realms/toolchain-public/protocol/openid-connect/certs", toolchainCfg.RegistrationService().Auth().AuthClientPublicKeysURL())
		assert.False(t, toolchainCfg.RegistrationService().Verification().Enabled())
		assert.Equal(t, 5, toolchainCfg.RegistrationService().Verification().DailyLimit())
		assert.Equal(t, 3, toolchainCfg.RegistrationService().Verification().AttemptsAllowed())
		assert.Equal(t, "Developer Sandbox for Red Hat OpenShift: Your verification code is %s", toolchainCfg.RegistrationService().Verification().MessageTemplate())
		assert.Empty(t, toolchainCfg.RegistrationService().Verification().ExcludedEmailDomains())
		assert.Equal(t, 5, toolchainCfg.RegistrationService().Verification().CodeExpiresInMin())
		assert.Empty(t, toolchainCfg.RegistrationService().Verification().TwilioAccountSID())
		assert.Empty(t, toolchainCfg.RegistrationService().Verification().TwilioAuthToken())
		assert.Empty(t, toolchainCfg.RegistrationService().Verification().TwilioFromNumber())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.RegistrationService().
			Environment("e2e-tests").
			LogLevel("debug").
			Namespace("another-namespace").
			RegistrationServiceURL("www.crtregservice.com").
			Analytics().SegmentWriteKey("keyabc").
			Analytics().WoopraDomain("woopra.com").
			Auth().AuthClientLibraryURL("https://sso.openshift.com/auth/js/keycloak.js").
			Auth().AuthClientConfigContentType("application/xml").
			Auth().AuthClientConfigRaw(`{"realm": "toolchain-private"}`).
			Auth().AuthClientPublicKeysURL("https://sso.openshift.com/certs").
			Verification().Enabled(true).
			Verification().DailyLimit(15).
			Verification().AttemptsAllowed(13).
			Verification().MessageTemplate("Developer Sandbox verification code: %s").
			Verification().ExcludedEmailDomains("redhat.com,ibm.com").
			Verification().CodeExpiresInMin(151).
			Verification().Secret().Ref("verification-secrets").TwilioAccountSID("twiolio.sid").TwilioAuthToken("twiolio.token").TwilioFromNumber("twiolio.fromnumber"))

		verificationSecretValues := make(map[string]string)
		verificationSecretValues["twiolio.sid"] = "def"
		verificationSecretValues["twiolio.token"] = "ghi"
		verificationSecretValues["twiolio.fromnumber"] = "jkl"
		secrets := make(map[string]map[string]string)
		secrets["verification-secrets"] = verificationSecretValues

		toolchainCfg := NewToolchainConfig(&cfg.Spec, secrets)

		assert.Equal(t, "e2e-tests", toolchainCfg.RegistrationService().Environment())
		assert.Equal(t, "debug", toolchainCfg.RegistrationService().LogLevel())
		assert.Equal(t, "another-namespace", toolchainCfg.RegistrationService().Namespace())
		assert.Equal(t, "www.crtregservice.com", toolchainCfg.RegistrationService().RegistrationServiceURL())
		assert.Equal(t, "keyabc", toolchainCfg.RegistrationService().Analytics().SegmentWriteKey())
		assert.Equal(t, "woopra.com", toolchainCfg.RegistrationService().Analytics().WoopraDomain())
		assert.Equal(t, "https://sso.openshift.com/auth/js/keycloak.js", toolchainCfg.RegistrationService().Auth().AuthClientLibraryURL())
		assert.Equal(t, "application/xml", toolchainCfg.RegistrationService().Auth().AuthClientConfigContentType())
		assert.Equal(t, `{"realm": "toolchain-private"}`, toolchainCfg.RegistrationService().Auth().AuthClientConfigRaw())
		assert.Equal(t, "https://sso.openshift.com/certs", toolchainCfg.RegistrationService().Auth().AuthClientPublicKeysURL())

		assert.True(t, toolchainCfg.RegistrationService().Verification().Enabled())
		assert.Equal(t, 15, toolchainCfg.RegistrationService().Verification().DailyLimit())
		assert.Equal(t, 13, toolchainCfg.RegistrationService().Verification().AttemptsAllowed())
		assert.Equal(t, "Developer Sandbox verification code: %s", toolchainCfg.RegistrationService().Verification().MessageTemplate())
		assert.Equal(t, "redhat.com,ibm.com", toolchainCfg.RegistrationService().Verification().ExcludedEmailDomains())
		assert.Equal(t, 151, toolchainCfg.RegistrationService().Verification().CodeExpiresInMin())
		assert.Equal(t, "def", toolchainCfg.RegistrationService().Verification().TwilioAccountSID())
		assert.Equal(t, "ghi", toolchainCfg.RegistrationService().Verification().TwilioAuthToken())
		assert.Equal(t, "jkl", toolchainCfg.RegistrationService().Verification().TwilioFromNumber())
	})
}

func TestTiers(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, "base", toolchainCfg.Tiers().DefaultTier())
		assert.Equal(t, 24*time.Hour, toolchainCfg.Tiers().DurationBeforeChangeTierRequestDeletion())
		assert.Equal(t, 5, toolchainCfg.Tiers().TemplateUpdateRequestMaxPoolSize())
	})
	t.Run("invalid", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.Tiers().DurationBeforeChangeTierRequestDeletion("rapid"))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 24*time.Hour, toolchainCfg.Tiers().DurationBeforeChangeTierRequestDeletion())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.Tiers().
			DefaultTier("advanced").
			DurationBeforeChangeTierRequestDeletion("48h").
			TemplateUpdateRequestMaxPoolSize(40))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, "advanced", toolchainCfg.Tiers().DefaultTier())
		assert.Equal(t, 48*time.Hour, toolchainCfg.Tiers().DurationBeforeChangeTierRequestDeletion())
		assert.Equal(t, 40, toolchainCfg.Tiers().TemplateUpdateRequestMaxPoolSize())
	})
}

func TestToolchainStatus(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 5*time.Second, toolchainCfg.ToolchainStatus().ToolchainStatusRefreshTime())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.ToolchainStatus().ToolchainStatusRefreshTime("10s"))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 10*time.Second, toolchainCfg.ToolchainStatus().ToolchainStatusRefreshTime())
	})
}

func TestUsers(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t)
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 2, toolchainCfg.Users().MasterUserRecordUpdateFailureThreshold())
		assert.Equal(t, []string{"openshift", "kube", "default", "redhat", "sandbox"}, toolchainCfg.Users().ForbiddenUsernamePrefixes())
		assert.Equal(t, []string{"admin"}, toolchainCfg.Users().ForbiddenUsernameSuffixes())
	})
	t.Run("non-default", func(t *testing.T) {
		cfg := NewToolchainConfigWithReset(t, testconfig.Users().MasterUserRecordUpdateFailureThreshold(10).ForbiddenUsernamePrefixes("bread,butter").ForbiddenUsernameSuffixes("sugar,cream"))
		toolchainCfg := NewToolchainConfig(&cfg.Spec, map[string]map[string]string{})

		assert.Equal(t, 10, toolchainCfg.Users().MasterUserRecordUpdateFailureThreshold())
		assert.Equal(t, []string{"bread", "butter"}, toolchainCfg.Users().ForbiddenUsernamePrefixes())
		assert.Equal(t, []string{"sugar", "cream"}, toolchainCfg.Users().ForbiddenUsernameSuffixes())
	})
}
