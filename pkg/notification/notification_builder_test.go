package notification

import (
	"strings"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testusersignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationBuilder(t *testing.T) {
	// given
	client := test.NewFakeClient(t)
	logger := zap.New(zap.UseDevMode(true))

	t.Run("success with no options", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).Create(logger, "foo@acme.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "foo@acme.com", notification.Spec.Recipient)
	})

	t.Run("fail with empty email address", func(t *testing.T) {
		// when
		_, err := NewNotificationBuilder(client, test.HostOperatorNs).Create(logger, "")

		// then
		require.Error(t, err)
		assert.Equal(t, "The specified recipient [] is not a valid email address", err.Error())
	})

	t.Run("fail with invalid email address", func(t *testing.T) {
		// when
		_, err := NewNotificationBuilder(client, test.HostOperatorNs).Create(logger, "foo")

		// then
		require.Error(t, err)
		assert.Equal(t, "The specified recipient [foo] is not a valid email address", err.Error())
	})

	t.Run("success with multiple valid email addresses", func(t *testing.T) {
		// given
		emailsToTest := []string{
			"john.wick@subdomain.domain.com",
			"john-Wick@domain.com",
		}

		for _, email := range emailsToTest {

			// when
			_, err := NewNotificationBuilder(client, test.HostOperatorNs).Create(logger, email)

			// then
			require.NoError(t, err)
		}
	})

	t.Run("success with user context", func(t *testing.T) {
		// given
		userSignup := testusersignup.NewUserSignup()
		userSignup.Spec.GivenName = "John"
		userSignup.Spec.FamilyName = "Smith"
		userSignup.Spec.Company = "ACME Corp"
		userSignup.Status = toolchainv1alpha1.UserSignupStatus{
			CompliantUsername: "jsmith",
		}

		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithUserContext(userSignup).
			Create(logger, userSignup.Annotations[toolchainv1alpha1.UserSignupUserEmailAnnotationKey])

		// then
		require.NoError(t, err)
		assert.Equal(t, userSignup.Annotations[toolchainv1alpha1.UserSignupUserEmailAnnotationKey], notification.Spec.Recipient)
		assert.Equal(t, userSignup.Annotations[toolchainv1alpha1.UserSignupUserEmailAnnotationKey], notification.Spec.Context["UserEmail"])
		assert.Equal(t, userSignup.Spec.GivenName, notification.Spec.Context["FirstName"])
		assert.Equal(t, userSignup.Spec.FamilyName, notification.Spec.Context["LastName"])
		assert.Equal(t, userSignup.Spec.Company, notification.Spec.Context["CompanyName"])
		assert.Equal(t, userSignup.Spec.Userid, notification.Spec.Context["UserID"])
		assert.Equal(t, userSignup.Status.CompliantUsername, notification.Spec.Context["UserName"])
	})

	t.Run("success with hard coded notification name", func(t *testing.T) {
		// given
		name := uuid.Must(uuid.NewV4()).String()
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithName(name).
			Create(logger, "foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, name, notification.Name)
	})

	t.Run("success with template", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithTemplate("default").
			Create(logger, "foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "default", notification.Spec.Template)
	})

	t.Run("success with subject and content", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithSubjectAndContent("This is a test subject", "This is some test content").
			Create(logger, "foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "This is a test subject", notification.Spec.Subject)
		assert.Equal(t, "This is some test content", notification.Spec.Content)
	})

	t.Run("success with keys and values", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithKeysAndValues(map[string]string{"foo": "bar"}).
			Create(logger, "foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "bar", notification.Spec.Context["foo"])
	})

	t.Run("success with notification type", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithNotificationType("TestNotificationType").
			Create(logger, "foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "TestNotificationType", notification.Labels[toolchainv1alpha1.NotificationTypeLabelKey])
	})

	t.Run("success with empty compliant username", func(t *testing.T) {
		// given
		userSignup := testusersignup.NewUserSignup()
		userSignup.Spec.GivenName = "John"
		userSignup.Spec.FamilyName = "Smith"
		userSignup.Spec.Company = "ACME Corp"
		userSignup.Status = toolchainv1alpha1.UserSignupStatus{
			CompliantUsername: "",
		}

		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithNotificationType("TestNotificationType").
			WithUserContext(userSignup).
			Create(logger, "foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "TestNotificationType", notification.Labels[toolchainv1alpha1.NotificationTypeLabelKey])
		assert.False(t, strings.HasPrefix(notification.Name, "-"))
	})
}
