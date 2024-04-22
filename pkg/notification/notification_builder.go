package notification

import (
	"context"
	"fmt"
	"net/mail"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/gofrs/uuid/v5"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Option = func(notification *toolchainv1alpha1.Notification) error

type Builder interface {
	WithName(name string) Builder
	WithTemplate(template string) Builder
	WithSubjectAndContent(subject, content string) Builder
	WithNotificationType(notificationType string) Builder
	WithControllerReference(owner v1.Object, scheme *runtime.Scheme) Builder
	WithKeysAndValues(keysAndValues map[string]string) Builder
	WithUserContext(userSignup *toolchainv1alpha1.UserSignup) Builder
	Create(ctx context.Context, recipient string) (*toolchainv1alpha1.Notification, error)
}

func NewNotificationBuilder(client client.Client, namespace string) Builder {
	return &notificationBuilderImpl{
		client:    client,
		namespace: namespace,
		options:   []Option{},
	}
}

type notificationBuilderImpl struct {
	client    client.Client
	namespace string
	options   []Option
}

func (b *notificationBuilderImpl) Create(ctx context.Context, recipient string) (*toolchainv1alpha1.Notification, error) {

	if list, err := mail.ParseAddressList(recipient); err != nil || len(list) == 0 {
		return nil, errors.Wrap(err, fmt.Sprintf("The specified recipient [%s] is not a valid email address", recipient))
	}

	notification := &toolchainv1alpha1.Notification{
		ObjectMeta: v1.ObjectMeta{
			Namespace: b.namespace,
			Labels:    map[string]string{},
		},
		Spec: toolchainv1alpha1.NotificationSpec{
			Recipient: recipient,
			Context:   make(map[string]string),
		},
	}

	for _, opt := range b.options {
		err := opt(notification)
		if err != nil {
			return nil, err
		}
	}

	generateName(notification)

	return notification, b.client.Create(ctx, notification)
}

func generateName(notification *toolchainv1alpha1.Notification) {
	if notification.ObjectMeta.Name == "" {
		if username, found := notification.Spec.Context["UserName"]; found && username != "" {
			notificationType, found := notification.Labels[toolchainv1alpha1.NotificationTypeLabelKey]
			if found {
				notification.ObjectMeta.GenerateName = fmt.Sprintf("%s-%s-", username, notificationType)
				return
			}
			notification.ObjectMeta.GenerateName = fmt.Sprintf("%s-untyped", username)
			return
		}
		notification.ObjectMeta.GenerateName = fmt.Sprintf("%s-untyped", uuid.Must(uuid.NewV4()).String())
	}
}

func (b *notificationBuilderImpl) WithName(name string) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {
		n.ObjectMeta.Name = name
		return nil
	})
	return b
}

func (b *notificationBuilderImpl) WithTemplate(template string) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {
		n.Spec.Template = template
		return nil
	})
	return b
}

func (b *notificationBuilderImpl) WithSubjectAndContent(subject, content string) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {
		n.Spec.Subject = subject
		n.Spec.Content = content
		return nil
	})
	return b
}

func (b *notificationBuilderImpl) WithNotificationType(notificationType string) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {
		n.ObjectMeta.Labels[toolchainv1alpha1.NotificationTypeLabelKey] = notificationType
		return nil
	})
	return b
}

func (b *notificationBuilderImpl) WithControllerReference(owner v1.Object, scheme *runtime.Scheme) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {
		return controllerutil.SetControllerReference(owner, n, scheme)
	})
	return b
}

func (b *notificationBuilderImpl) WithKeysAndValues(keysAndValues map[string]string) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {
		for k, v := range keysAndValues {
			n.Spec.Context[k] = v
		}
		return nil
	})
	return b
}

func (b *notificationBuilderImpl) WithUserContext(userSignup *toolchainv1alpha1.UserSignup) Builder {
	b.options = append(b.options, func(n *toolchainv1alpha1.Notification) error {

		n.Spec.Context["Sub"] = userSignup.Spec.IdentityClaims.Sub
		n.Spec.Context["UserID"] = userSignup.Spec.IdentityClaims.Sub
		n.Spec.Context["UserName"] = userSignup.Status.CompliantUsername
		n.Spec.Context["FirstName"] = userSignup.Spec.IdentityClaims.GivenName
		n.Spec.Context["LastName"] = userSignup.Spec.IdentityClaims.FamilyName
		n.Spec.Context["CompanyName"] = userSignup.Spec.IdentityClaims.Company

		n.ObjectMeta.Labels[toolchainv1alpha1.NotificationUserNameLabelKey] = userSignup.Status.CompliantUsername

		if userSignup.Spec.IdentityClaims.Email != "" {
			n.Spec.Context["UserEmail"] = userSignup.Spec.IdentityClaims.Email
		}

		return nil
	})
	return b
}
