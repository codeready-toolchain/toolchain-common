package configuration

import (
	"fmt"
	"strings"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
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

// GetToolchainConfig returns a ToolchainConfig using the cache, or if the cache was not initialized
// then retrieves the latest config using the provided client and updates the cache
func GetToolchainConfig(cl client.Client) (ToolchainConfig, error) {
	config, secrets, err := getConfig(cl, &toolchainv1alpha1.ToolchainConfig{})
	if err != nil {
		// return default config
		logger.Error(err, "failed to retrieve ToolchainConfig")
		return ToolchainConfig{cfg: &toolchainv1alpha1.ToolchainConfigSpec{}}, err
	}
	return newToolchainConfig(config, secrets), nil
}

// GetCachedToolchainConfig returns a ToolchainConfig directly from the cache
func GetCachedToolchainConfig() ToolchainConfig {
	config, secrets := getCachedConfig()
	return newToolchainConfig(config, secrets)
}

// ForceLoadToolchainConfig updates the cache using the provided client and returns the latest ToolchainConfig
func ForceLoadToolchainConfig(cl client.Client) (ToolchainConfig, error) {
	config, secrets, err := loadLatest(cl, &toolchainv1alpha1.ToolchainConfig{})
	if err != nil {
		// return default config
		logger.Error(err, "failed to force load ToolchainConfig")
		return ToolchainConfig{cfg: &toolchainv1alpha1.ToolchainConfigSpec{}}, err
	}
	return newToolchainConfig(config, secrets), nil
}

func newToolchainConfig(config runtime.Object, secrets map[string]map[string]string) ToolchainConfig {
	if config == nil {
		// return default config if there's no config resource
		return ToolchainConfig{cfg: &toolchainv1alpha1.ToolchainConfigSpec{}}
	}

	toolchaincfg, ok := config.(*toolchainv1alpha1.ToolchainConfig)
	if !ok {
		// return default config
		logger.Error(fmt.Errorf("cache does not contain toolchainconfig resource type"), "failed to get ToolchainConfig from resource, using default configuration")
		return ToolchainConfig{cfg: &toolchainv1alpha1.ToolchainConfigSpec{}}
	}
	return ToolchainConfig{cfg: &toolchaincfg.Spec, secrets: secrets}
}

func (c *ToolchainConfig) Print() {
	logger.Info("Toolchain configuration variables", "ToolchainConfigSpec", c.cfg)
}

func (c *ToolchainConfig) Environment() string {
	return GetString(c.cfg.Host.Environment, "prod")
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
	return GetBool(a.approval.Enabled, false)
}

func (a AutoApprovalConfig) ResourceCapacityThresholdDefault() int {
	return GetInt(a.approval.ResourceCapacityThreshold.DefaultThreshold, 80)
}

func (a AutoApprovalConfig) ResourceCapacityThresholdSpecificPerMemberCluster() map[string]int {
	return a.approval.ResourceCapacityThreshold.SpecificPerMemberCluster
}

func (a AutoApprovalConfig) MaxNumberOfUsersOverall() int {
	return GetInt(a.approval.MaxNumberOfUsers.Overall, 1000)
}

func (a AutoApprovalConfig) MaxNumberOfUsersSpecificPerMemberCluster() map[string]int {
	return a.approval.MaxNumberOfUsers.SpecificPerMemberCluster
}

type DeactivationConfig struct {
	dctv toolchainv1alpha1.DeactivationConfig
}

func (d DeactivationConfig) DeactivatingNotificationDays() int {
	return GetInt(d.dctv.DeactivatingNotificationDays, 3)
}

func (d DeactivationConfig) DeactivationDomainsExcluded() []string {
	excluded := GetString(d.dctv.DeactivationDomainsExcluded, "")
	v := strings.FieldsFunc(excluded, func(c rune) bool {
		return c == ','
	})
	return v
}

func (d DeactivationConfig) UserSignupDeactivatedRetentionDays() int {
	return GetInt(d.dctv.UserSignupDeactivatedRetentionDays, 365)
}

func (d DeactivationConfig) UserSignupUnverifiedRetentionDays() int {
	return GetInt(d.dctv.UserSignupUnverifiedRetentionDays, 7)
}

type MetricsConfig struct {
	metrics toolchainv1alpha1.MetricsConfig
}

func (d MetricsConfig) ForceSynchronization() bool {
	return GetBool(d.metrics.ForceSynchronization, false)
}

type NotificationsConfig struct {
	c       toolchainv1alpha1.NotificationsConfig
	secrets map[string]map[string]string
}

func (n NotificationsConfig) notificationSecret(secretKey string) string {
	secret := GetString(n.c.Secret.Ref, "")
	return n.secrets[secret][secretKey]
}

func (n NotificationsConfig) NotificationDeliveryService() string {
	return GetString(n.c.NotificationDeliveryService, "mailgun")
}

func (n NotificationsConfig) DurationBeforeNotificationDeletion() time.Duration {
	v := GetString(n.c.DurationBeforeNotificationDeletion, "24h")
	duration, err := time.ParseDuration(v)
	if err != nil {
		duration = 24 * time.Hour
	}
	return duration
}

func (n NotificationsConfig) AdminEmail() string {
	return GetString(n.c.AdminEmail, "")
}

func (n NotificationsConfig) MailgunDomain() string {
	key := GetString(n.c.Secret.MailgunDomain, "")
	return n.notificationSecret(key)
}

func (n NotificationsConfig) MailgunAPIKey() string {
	key := GetString(n.c.Secret.MailgunAPIKey, "")
	return n.notificationSecret(key)
}

func (n NotificationsConfig) MailgunSenderEmail() string {
	key := GetString(n.c.Secret.MailgunSenderEmail, "")
	return n.notificationSecret(key)
}

func (n NotificationsConfig) MailgunReplyToEmail() string {
	key := GetString(n.c.Secret.MailgunReplyToEmail, "")
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
	return GetString(r.c.Environment, "prod")
}

func (r RegistrationServiceConfig) LogLevel() string {
	return GetString(r.c.LogLevel, "info")
}

func (r RegistrationServiceConfig) Namespace() string {
	return GetString(r.c.Namespace, "toolchain-host-operator")
}

func (r RegistrationServiceConfig) RegistrationServiceURL() string {
	return GetString(r.c.RegistrationServiceURL, "https://registration.crt-placeholder.com")
}

func (r RegistrationServiceConfig) Verification() RegistrationServiceVerificationConfig {
	return RegistrationServiceVerificationConfig{c: r.c.Verification, secrets: r.secrets}
}

type RegistrationServiceAnalyticsConfig struct {
	c toolchainv1alpha1.RegistrationServiceAnalyticsConfig
}

func (r RegistrationServiceAnalyticsConfig) WoopraDomain() string {
	return GetString(r.c.WoopraDomain, "")
}

func (r RegistrationServiceAnalyticsConfig) SegmentWriteKey() string {
	return GetString(r.c.SegmentWriteKey, "")
}

type RegistrationServiceAuthConfig struct {
	c toolchainv1alpha1.RegistrationServiceAuthConfig
}

func (r RegistrationServiceAuthConfig) AuthClientLibraryURL() string {
	return GetString(r.c.AuthClientLibraryURL, "https://sso.prod-preview.openshift.io/auth/js/keycloak.js")
}

func (r RegistrationServiceAuthConfig) AuthClientConfigContentType() string {
	return GetString(r.c.AuthClientConfigContentType, "application/json; charset=utf-8")
}

func (r RegistrationServiceAuthConfig) AuthClientConfigRaw() string {
	return GetString(r.c.AuthClientConfigRaw, `{"realm": "toolchain-public","auth-server-url": "https://sso.prod-preview.openshift.io/auth","ssl-required": "none","resource": "crt","clientId": "crt","public-client": true}`)
}

func (r RegistrationServiceAuthConfig) AuthClientPublicKeysURL() string {
	return GetString(r.c.AuthClientPublicKeysURL, "https://sso.prod-preview.openshift.io/auth/realms/toolchain-public/protocol/openid-connect/certs")
}

type RegistrationServiceVerificationConfig struct {
	c       toolchainv1alpha1.RegistrationServiceVerificationConfig
	secrets map[string]map[string]string
}

func (r RegistrationServiceVerificationConfig) registrationServiceSecret(secretKey string) string {
	secret := GetString(r.c.Secret.Ref, "")
	return r.secrets[secret][secretKey]
}

func (r RegistrationServiceVerificationConfig) Enabled() bool {
	return GetBool(r.c.Enabled, false)
}

func (r RegistrationServiceVerificationConfig) DailyLimit() int {
	return GetInt(r.c.DailyLimit, 5)
}

func (r RegistrationServiceVerificationConfig) AttemptsAllowed() int {
	return GetInt(r.c.AttemptsAllowed, 3)
}

func (r RegistrationServiceVerificationConfig) MessageTemplate() string {
	return GetString(r.c.MessageTemplate, "Developer Sandbox for Red Hat OpenShift: Your verification code is %s")
}

func (r RegistrationServiceVerificationConfig) ExcludedEmailDomains() []string {
	excluded := GetString(r.c.ExcludedEmailDomains, "")
	v := strings.FieldsFunc(excluded, func(c rune) bool {
		return c == ','
	})
	return v
}

func (r RegistrationServiceVerificationConfig) CodeExpiresInMin() int {
	return GetInt(r.c.CodeExpiresInMin, 5)
}

func (r RegistrationServiceVerificationConfig) TwilioAccountSID() string {
	key := GetString(r.c.Secret.TwilioAccountSID, "")
	return r.registrationServiceSecret(key)
}

func (r RegistrationServiceVerificationConfig) TwilioAuthToken() string {
	key := GetString(r.c.Secret.TwilioAuthToken, "")
	return r.registrationServiceSecret(key)
}

func (r RegistrationServiceVerificationConfig) TwilioFromNumber() string {
	key := GetString(r.c.Secret.TwilioFromNumber, "")
	return r.registrationServiceSecret(key)
}

type TiersConfig struct {
	tiers toolchainv1alpha1.TiersConfig
}

func (d TiersConfig) DefaultTier() string {
	return GetString(d.tiers.DefaultTier, "base")
}

func (d TiersConfig) DurationBeforeChangeTierRequestDeletion() time.Duration {
	v := GetString(d.tiers.DurationBeforeChangeTierRequestDeletion, "24h")
	duration, err := time.ParseDuration(v)
	if err != nil {
		duration = 24 * time.Hour
	}
	return duration
}

func (d TiersConfig) TemplateUpdateRequestMaxPoolSize() int {
	return GetInt(d.tiers.TemplateUpdateRequestMaxPoolSize, 5)
}

type ToolchainStatusConfig struct {
	t toolchainv1alpha1.ToolchainStatusConfig
}

func (d ToolchainStatusConfig) ToolchainStatusRefreshTime() time.Duration {
	v := GetString(d.t.ToolchainStatusRefreshTime, "5s")
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
	return GetInt(d.c.MasterUserRecordUpdateFailureThreshold, 2) // default: allow 1 failure, try again and then give up if failed again
}

func (d UsersConfig) ForbiddenUsernamePrefixes() []string {
	prefixes := GetString(d.c.ForbiddenUsernamePrefixes, "openshift,kube,default,redhat,sandbox")
	v := strings.FieldsFunc(prefixes, func(c rune) bool {
		return c == ','
	})
	return v
}

func (d UsersConfig) ForbiddenUsernameSuffixes() []string {
	suffixes := GetString(d.c.ForbiddenUsernameSuffixes, "admin")
	v := strings.FieldsFunc(suffixes, func(c rune) bool {
		return c == ','
	})
	return v
}
