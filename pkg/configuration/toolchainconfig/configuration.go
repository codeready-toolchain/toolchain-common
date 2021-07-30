package toolchainconfig

import (
	"strings"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/configuration"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ToolchainStatusName = "toolchain-status"

	// NotificationDeliveryServiceMailgun is the notification delivery service to use during production
	NotificationDeliveryServiceMailgun = "mailgun"
)

var logger = logf.Log.WithName("configuration")

type ToolchainConfig struct {
	cfg     *toolchainv1alpha1.ToolchainConfigSpec
	secrets map[string]map[string]string
}

func NewToolchainConfig(cfg *toolchainv1alpha1.ToolchainConfigSpec, secrets map[string]map[string]string) ToolchainConfig {
	return ToolchainConfig{
		cfg:     cfg,
		secrets: secrets,
	}
}

func (c *ToolchainConfig) Print() {
	logger.Info("Toolchain configuration variables", "ToolchainConfigSpec", c.cfg)
}

func (c *ToolchainConfig) Environment() string {
	return configuration.GetString(c.cfg.Host.Environment, "prod")
}

func (c *ToolchainConfig) AutomaticApproval() AutoApprovalConfig {
	return AutoApprovalConfig{c.cfg.Host.AutomaticApproval}
}

func (c *ToolchainConfig) Deactivation() DeactivationConfig {
	return DeactivationConfig{c.cfg.Host.Deactivation}
}

func (c *ToolchainConfig) Metrics() MetricsConfig {
	return MetricsConfig{c.cfg.Host.Metrics}
}

func (c *ToolchainConfig) Notifications() NotificationsConfig {
	return NotificationsConfig{
		c:       c.cfg.Host.Notifications,
		secrets: c.secrets,
	}
}

func (c *ToolchainConfig) RegistrationService() RegistrationServiceConfig {
	return RegistrationServiceConfig{
		c:       c.cfg.Host.RegistrationService,
		secrets: c.secrets,
	}
}

func (c *ToolchainConfig) Tiers() TiersConfig {
	return TiersConfig{c.cfg.Host.Tiers}
}

func (c *ToolchainConfig) ToolchainStatus() ToolchainStatusConfig {
	return ToolchainStatusConfig{c.cfg.Host.ToolchainStatus}
}

func (c *ToolchainConfig) Users() UsersConfig {
	return UsersConfig{c.cfg.Host.Users}
}

type AutoApprovalConfig struct {
	approval toolchainv1alpha1.AutomaticApprovalConfig
}

func (a AutoApprovalConfig) IsEnabled() bool {
	return configuration.GetBool(a.approval.Enabled, false)
}

func (a AutoApprovalConfig) ResourceCapacityThresholdDefault() int {
	return configuration.GetInt(a.approval.ResourceCapacityThreshold.DefaultThreshold, 80)
}

func (a AutoApprovalConfig) ResourceCapacityThresholdSpecificPerMemberCluster() map[string]int {
	return a.approval.ResourceCapacityThreshold.SpecificPerMemberCluster
}

func (a AutoApprovalConfig) MaxNumberOfUsersOverall() int {
	return configuration.GetInt(a.approval.MaxNumberOfUsers.Overall, 1000)
}

func (a AutoApprovalConfig) MaxNumberOfUsersSpecificPerMemberCluster() map[string]int {
	return a.approval.MaxNumberOfUsers.SpecificPerMemberCluster
}

type DeactivationConfig struct {
	dctv toolchainv1alpha1.DeactivationConfig
}

func (d DeactivationConfig) DeactivatingNotificationDays() int {
	return configuration.GetInt(d.dctv.DeactivatingNotificationDays, 3)
}

func (d DeactivationConfig) DeactivationDomainsExcluded() []string {
	excluded := configuration.GetString(d.dctv.DeactivationDomainsExcluded, "")
	v := strings.FieldsFunc(excluded, func(c rune) bool {
		return c == ','
	})
	return v
}

func (d DeactivationConfig) UserSignupDeactivatedRetentionDays() int {
	return configuration.GetInt(d.dctv.UserSignupDeactivatedRetentionDays, 365)
}

func (d DeactivationConfig) UserSignupUnverifiedRetentionDays() int {
	return configuration.GetInt(d.dctv.UserSignupUnverifiedRetentionDays, 7)
}

type MetricsConfig struct {
	metrics toolchainv1alpha1.MetricsConfig
}

func (d MetricsConfig) ForceSynchronization() bool {
	return configuration.GetBool(d.metrics.ForceSynchronization, false)
}

type NotificationsConfig struct {
	c       toolchainv1alpha1.NotificationsConfig
	secrets map[string]map[string]string
}

func (n NotificationsConfig) notificationSecret(secretKey string) string {
	secret := configuration.GetString(n.c.Secret.Ref, "")
	return n.secrets[secret][secretKey]
}

func (n NotificationsConfig) NotificationDeliveryService() string {
	return configuration.GetString(n.c.NotificationDeliveryService, "mailgun")
}

func (n NotificationsConfig) DurationBeforeNotificationDeletion() time.Duration {
	v := configuration.GetString(n.c.DurationBeforeNotificationDeletion, "24h")
	duration, err := time.ParseDuration(v)
	if err != nil {
		duration = 24 * time.Hour
	}
	return duration
}

func (n NotificationsConfig) AdminEmail() string {
	return configuration.GetString(n.c.AdminEmail, "")
}

func (n NotificationsConfig) MailgunDomain() string {
	key := configuration.GetString(n.c.Secret.MailgunDomain, "")
	return n.notificationSecret(key)
}

func (n NotificationsConfig) MailgunAPIKey() string {
	key := configuration.GetString(n.c.Secret.MailgunAPIKey, "")
	return n.notificationSecret(key)
}

func (n NotificationsConfig) MailgunSenderEmail() string {
	key := configuration.GetString(n.c.Secret.MailgunSenderEmail, "")
	return n.notificationSecret(key)
}

func (n NotificationsConfig) MailgunReplyToEmail() string {
	key := configuration.GetString(n.c.Secret.MailgunReplyToEmail, "")
	return n.notificationSecret(key)
}

type RegistrationServiceConfig struct {
	c       toolchainv1alpha1.RegistrationServiceConfig
	secrets map[string]map[string]string
}

func (r RegistrationServiceConfig) Analytics() RegistrationServiceAnalyticsConfig {
	return RegistrationServiceAnalyticsConfig{r.c.Analytics}
}

func (r RegistrationServiceConfig) Auth() RegistrationServiceAuthConfig {
	return RegistrationServiceAuthConfig{r.c.Auth}
}

func (r RegistrationServiceConfig) Environment() string {
	return configuration.GetString(r.c.Environment, "prod")
}

func (r RegistrationServiceConfig) LogLevel() string {
	return configuration.GetString(r.c.LogLevel, "info")
}

func (r RegistrationServiceConfig) Namespace() string {
	return configuration.GetString(r.c.Namespace, "toolchain-host-operator")
}

func (r RegistrationServiceConfig) RegistrationServiceURL() string {
	return configuration.GetString(r.c.RegistrationServiceURL, "https://registration.crt-placeholder.com")
}

func (r RegistrationServiceConfig) Verification() RegistrationServiceVerificationConfig {
	return RegistrationServiceVerificationConfig{c: r.c.Verification, secrets: r.secrets}
}

type RegistrationServiceAnalyticsConfig struct {
	c toolchainv1alpha1.RegistrationServiceAnalyticsConfig
}

func (r RegistrationServiceAnalyticsConfig) WoopraDomain() string {
	return configuration.GetString(r.c.WoopraDomain, "")
}

func (r RegistrationServiceAnalyticsConfig) SegmentWriteKey() string {
	return configuration.GetString(r.c.SegmentWriteKey, "")
}

type RegistrationServiceAuthConfig struct {
	c toolchainv1alpha1.RegistrationServiceAuthConfig
}

func (r RegistrationServiceAuthConfig) AuthClientLibraryURL() string {
	return configuration.GetString(r.c.AuthClientLibraryURL, "https://sso.prod-preview.openshift.io/auth/js/keycloak.js")
}

func (r RegistrationServiceAuthConfig) AuthClientConfigContentType() string {
	return configuration.GetString(r.c.AuthClientConfigContentType, "application/json; charset=utf-8")
}

func (r RegistrationServiceAuthConfig) AuthClientConfigRaw() string {
	return configuration.GetString(r.c.AuthClientConfigRaw, `{"realm": "toolchain-public","auth-server-url": "https://sso.prod-preview.openshift.io/auth","ssl-required": "none","resource": "crt","clientId": "crt","public-client": true}`)
}

func (r RegistrationServiceAuthConfig) AuthClientPublicKeysURL() string {
	return configuration.GetString(r.c.AuthClientPublicKeysURL, "https://sso.prod-preview.openshift.io/auth/realms/toolchain-public/protocol/openid-connect/certs")
}

type RegistrationServiceVerificationConfig struct {
	c       toolchainv1alpha1.RegistrationServiceVerificationConfig
	secrets map[string]map[string]string
}

func (r RegistrationServiceVerificationConfig) registrationServiceSecret(secretKey string) string {
	secret := configuration.GetString(r.c.Secret.Ref, "")
	return r.secrets[secret][secretKey]
}

func (r RegistrationServiceVerificationConfig) Enabled() bool {
	return configuration.GetBool(r.c.Enabled, false)
}

func (r RegistrationServiceVerificationConfig) DailyLimit() int {
	return configuration.GetInt(r.c.DailyLimit, 5)
}

func (r RegistrationServiceVerificationConfig) AttemptsAllowed() int {
	return configuration.GetInt(r.c.AttemptsAllowed, 3)
}

func (r RegistrationServiceVerificationConfig) MessageTemplate() string {
	return configuration.GetString(r.c.MessageTemplate, "Developer Sandbox for Red Hat OpenShift: Your verification code is %s")
}

func (r RegistrationServiceVerificationConfig) ExcludedEmailDomains() []string {
	excluded := configuration.GetString(r.c.ExcludedEmailDomains, "")
	v := strings.FieldsFunc(excluded, func(c rune) bool {
		return c == ','
	})
	return v
}

func (r RegistrationServiceVerificationConfig) CodeExpiresInMin() int {
	return configuration.GetInt(r.c.CodeExpiresInMin, 5)
}

func (r RegistrationServiceVerificationConfig) TwilioAccountSID() string {
	key := configuration.GetString(r.c.Secret.TwilioAccountSID, "")
	return r.registrationServiceSecret(key)
}

func (r RegistrationServiceVerificationConfig) TwilioAuthToken() string {
	key := configuration.GetString(r.c.Secret.TwilioAuthToken, "")
	return r.registrationServiceSecret(key)
}

func (r RegistrationServiceVerificationConfig) TwilioFromNumber() string {
	key := configuration.GetString(r.c.Secret.TwilioFromNumber, "")
	return r.registrationServiceSecret(key)
}

type TiersConfig struct {
	tiers toolchainv1alpha1.TiersConfig
}

func (d TiersConfig) DefaultTier() string {
	return configuration.GetString(d.tiers.DefaultTier, "base")
}

func (d TiersConfig) DurationBeforeChangeTierRequestDeletion() time.Duration {
	v := configuration.GetString(d.tiers.DurationBeforeChangeTierRequestDeletion, "24h")
	duration, err := time.ParseDuration(v)
	if err != nil {
		duration = 24 * time.Hour
	}
	return duration
}

func (d TiersConfig) TemplateUpdateRequestMaxPoolSize() int {
	return configuration.GetInt(d.tiers.TemplateUpdateRequestMaxPoolSize, 5)
}

type ToolchainStatusConfig struct {
	t toolchainv1alpha1.ToolchainStatusConfig
}

func (d ToolchainStatusConfig) ToolchainStatusRefreshTime() time.Duration {
	v := configuration.GetString(d.t.ToolchainStatusRefreshTime, "5s")
	duration, err := time.ParseDuration(v)
	if err != nil {
		duration = 5 * time.Second
	}
	return duration
}

type UsersConfig struct {
	c toolchainv1alpha1.UsersConfig
}

func (d UsersConfig) MasterUserRecordUpdateFailureThreshold() int {
	return configuration.GetInt(d.c.MasterUserRecordUpdateFailureThreshold, 2) // default: allow 1 failure, try again and then give up if failed again
}

func (d UsersConfig) ForbiddenUsernamePrefixes() []string {
	prefixes := configuration.GetString(d.c.ForbiddenUsernamePrefixes, "openshift,kube,default,redhat,sandbox")
	v := strings.FieldsFunc(prefixes, func(c rune) bool {
		return c == ','
	})
	return v
}

func (d UsersConfig) ForbiddenUsernameSuffixes() []string {
	suffixes := configuration.GetString(d.c.ForbiddenUsernameSuffixes, "admin")
	v := strings.FieldsFunc(suffixes, func(c rune) bool {
		return c == ','
	})
	return v
}
