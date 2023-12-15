package notification

import (
	"context"
	"strings"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testusersignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationBuilder(t *testing.T) {
	// given
	client := test.NewFakeClient(t)

	t.Run("success with no options", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).Create("foo@acme.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "foo@acme.com", notification.Spec.Recipient)
	})

	t.Run("fail with empty email address", func(t *testing.T) {
		// when
		_, err := NewNotificationBuilder(client, test.HostOperatorNs).Create("")

		// then
		require.Error(t, err)
		assert.Equal(t, "The specified recipient [] is not a valid email address: mail: no address", err.Error())
	})

	t.Run("fail with invalid email address", func(t *testing.T) {
		// when
		_, err := NewNotificationBuilder(client, test.HostOperatorNs).Create("foo")

		// then
		require.Error(t, err)
		assert.Equal(t, "The specified recipient [foo] is not a valid email address: mail: missing '@' or angle-addr", err.Error())
	})

	t.Run("success", func(t *testing.T) {
		// given
		userSignup := testusersignup.NewUserSignup()
		userSignup.Spec.IdentityClaims.GivenName = "John"
		userSignup.Spec.IdentityClaims.FamilyName = "Smith"
		userSignup.Spec.IdentityClaims.Company = "ACME Corp"
		userSignup.Status = toolchainv1alpha1.UserSignupStatus{
			CompliantUsername: "jsmith",
		}
		emailsToTest := []string{
			userSignup.Annotations[toolchainv1alpha1.UserSignupUserEmailAnnotationKey],
			"john.wick@subdomain.domain.com",
			"john-Wick@domain.com",
			"john@domain.com,another-john@some.com",
			"john@domain.com, with-comma@some.com, ",
			"john@domain.com,another-john@some.com, with-space@john.com",
			"Alice <alice@example.com>",
			"Alice <alice@example.com>, Bob <bob@example.com>, Eve <eve@example.com>",
		}

		for _, email := range emailsToTest {
			t.Run("with email "+email, func(t *testing.T) {
				// given
				client := test.NewFakeClient(t)

				// and when
				not, err := NewNotificationBuilder(client, test.HostOperatorNs).
					WithUserContext(userSignup).
					Create(email)

				// then
				require.NoError(t, err)
				notifications := &toolchainv1alpha1.NotificationList{}
				require.NoError(t, client.List(context.TODO(), notifications, runtimeclient.InNamespace(test.HostOperatorNs)))
				require.Len(t, notifications.Items, 1)
				notification := notifications.Items[0]

				assert.Equal(t, email, notification.Spec.Recipient)
				assert.Equal(t, userSignup.Annotations[toolchainv1alpha1.UserSignupUserEmailAnnotationKey], notification.Spec.Context["UserEmail"])
				assert.Equal(t, userSignup.Spec.IdentityClaims.GivenName, notification.Spec.Context["FirstName"])
				assert.Equal(t, userSignup.Spec.IdentityClaims.FamilyName, notification.Spec.Context["LastName"])
				assert.Equal(t, userSignup.Spec.IdentityClaims.Company, notification.Spec.Context["CompanyName"])
				assert.Equal(t, userSignup.Spec.IdentityClaims.Sub, notification.Spec.Context["Sub"])
				assert.Equal(t, userSignup.Status.CompliantUsername, notification.Spec.Context["UserName"])
				assert.Equal(t, userSignup.Status.CompliantUsername, notification.Labels[toolchainv1alpha1.NotificationUserNameLabelKey])

				assert.Equal(t, notification.Annotations, not.Annotations)
				assert.Equal(t, notification.Labels, not.Labels)
				assert.Equal(t, notification.Spec, not.Spec)
			})
		}
	})

	t.Run("success with hard coded notification name", func(t *testing.T) {
		// given
		name := uuid.Must(uuid.NewV4()).String()
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithName(name).
			Create("foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, name, notification.Name)
	})

	t.Run("success with template", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithTemplate("default").
			Create("foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "default", notification.Spec.Template)
	})

	t.Run("success with subject and content", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithSubjectAndContent("This is a test subject", "This is some test content").
			Create("foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "This is a test subject", notification.Spec.Subject)
		assert.Equal(t, "This is some test content", notification.Spec.Content)
	})

	t.Run("success with keys and values", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithKeysAndValues(map[string]string{"foo": "bar"}).
			Create("foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "bar", notification.Spec.Context["foo"])
	})

	t.Run("success with notification type", func(t *testing.T) {
		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithNotificationType("TestNotificationType").
			Create("foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "TestNotificationType", notification.Labels[toolchainv1alpha1.NotificationTypeLabelKey])
	})

	t.Run("success with empty compliant username", func(t *testing.T) {
		// given
		userSignup := testusersignup.NewUserSignup()
		userSignup.Spec.IdentityClaims.GivenName = "John"
		userSignup.Spec.IdentityClaims.FamilyName = "Smith"
		userSignup.Spec.IdentityClaims.Company = "ACME Corp"
		userSignup.Status = toolchainv1alpha1.UserSignupStatus{
			CompliantUsername: "",
		}

		// when
		notification, err := NewNotificationBuilder(client, test.HostOperatorNs).
			WithNotificationType("TestNotificationType").
			WithUserContext(userSignup).
			Create("foo@bar.com")

		// then
		require.NoError(t, err)
		assert.Equal(t, "TestNotificationType", notification.Labels[toolchainv1alpha1.NotificationTypeLabelKey])
		assert.False(t, strings.HasPrefix(notification.Name, "-"))
	})
}
