package usersignup

import (
	"crypto/md5"// nolint:gosec
	"encoding/hex"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/gofrs/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Modifier func(*toolchainv1alpha1.UserSignup)

func NewUserSignup(modifiers ...Modifier) *toolchainv1alpha1.UserSignup {
	signup := &toolchainv1alpha1.UserSignup{
		ObjectMeta: NewUserSignupObjectMeta("", "foo@redhat.com"),
		Spec: toolchainv1alpha1.UserSignupSpec{
			Userid:   "UserID123",
			Username: "foo@redhat.com",
		},
	}
	for _, modify := range modifiers {
		modify(signup)
	}
	return signup
}

func NewUserSignupObjectMeta(name, email string) metav1.ObjectMeta {
	if name == "" {
		name = uuid.Must(uuid.NewV4()).String()
	}

	md5hash := md5.New() // nolint:gosec
	// Ignore the error, as this implementation cannot return one
	_, _ = md5hash.Write([]byte(email))
	emailHash := hex.EncodeToString(md5hash.Sum(nil))

	return metav1.ObjectMeta{
		Name:      name,
		Namespace: test.HostOperatorNs,
		Annotations: map[string]string{
			toolchainv1alpha1.UserSignupUserEmailAnnotationKey: email,
		},
		Labels: map[string]string{
			toolchainv1alpha1.UserSignupUserEmailHashLabelKey: emailHash,
		},
		CreationTimestamp: metav1.Now(),
	}
}