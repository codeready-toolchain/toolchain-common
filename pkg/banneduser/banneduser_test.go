package banneduser

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	commonsignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewBannedUser(t *testing.T) {
	userSignup1 := commonsignup.NewUserSignup(commonsignup.WithName("johny"), commonsignup.WithEmail("jonhy@example.com"))
	userSignup1UserEmailHash := userSignup1.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey]

	userSignup2 := commonsignup.NewUserSignup(commonsignup.WithName("bob"), commonsignup.WithEmail("bob@example.com"))
	userSignup2.Labels = nil

	userSignup3 := commonsignup.NewUserSignup(commonsignup.WithName("oliver"), commonsignup.WithEmail("oliver@example.com"))
	userSignup3PhoneHash := "fd276563a8232d16620da8ec85d0575f"
	userSignup3.Labels[toolchainv1alpha1.UserSignupUserPhoneHashLabelKey] = userSignup3PhoneHash
	userSignup3EmailHash := userSignup3.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey]

	tests := []struct {
		name               string
		userSignup         *toolchainv1alpha1.UserSignup
		bannedBy           string
		banReason          string
		wantError          bool
		wantErrorMsg       string
		expectedBannedUser *toolchainv1alpha1.BannedUser
	}{
		{
			name:         "userSignup with email hash label",
			userSignup:   userSignup1,
			bannedBy:     "admin",
			banReason:    "ban reason 1",
			wantError:    false,
			wantErrorMsg: "",
			expectedBannedUser: &toolchainv1alpha1.BannedUser{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: userSignup1.Namespace,
					Name:      fmt.Sprintf("banneduser-%s", userSignup1UserEmailHash),
					Labels: map[string]string{
						toolchainv1alpha1.BannedUserEmailHashLabelKey: userSignup1UserEmailHash,
						toolchainv1alpha1.BannedByLabelKey:            "admin",
					},
				},
				Spec: toolchainv1alpha1.BannedUserSpec{
					Email:  userSignup1.Spec.IdentityClaims.Email,
					Reason: "ban reason 1",
				},
			},
		},
		{
			name:               "userSignup without email hash label and phone hash label",
			userSignup:         userSignup2,
			bannedBy:           "admin",
			banReason:          "ban reason 2",
			wantError:          true,
			wantErrorMsg:       fmt.Sprintf("the UserSignup %s doesn't have the label '%s' set", userSignup2.Name, toolchainv1alpha1.UserSignupUserEmailHashLabelKey),
			expectedBannedUser: nil,
		},
		{
			name:         "userSignup with email hash label and phone hash label",
			userSignup:   userSignup3,
			bannedBy:     "admin",
			banReason:    "ban reason 3",
			wantError:    false,
			wantErrorMsg: "",
			expectedBannedUser: &toolchainv1alpha1.BannedUser{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: userSignup3.Namespace,
					Name:      fmt.Sprintf("banneduser-%s", userSignup3EmailHash),
					Labels: map[string]string{
						toolchainv1alpha1.BannedUserEmailHashLabelKey:     userSignup3EmailHash,
						toolchainv1alpha1.BannedByLabelKey:                "admin",
						toolchainv1alpha1.UserSignupUserPhoneHashLabelKey: userSignup3PhoneHash,
					},
				},
				Spec: toolchainv1alpha1.BannedUserSpec{
					Email:  userSignup3.Spec.IdentityClaims.Email,
					Reason: "ban reason 3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBannedUser(tt.userSignup, tt.bannedBy, tt.banReason)

			if tt.wantError {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrorMsg, err.Error())
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)

				assert.Equal(t, tt.expectedBannedUser.Namespace, got.Namespace)
				assert.Equal(t, tt.expectedBannedUser.Name, got.Name)
				assert.Equal(t, tt.expectedBannedUser.Spec.Email, got.Spec.Email)
				assert.Equal(t, tt.expectedBannedUser.Spec.Reason, got.Spec.Reason)

				if tt.expectedBannedUser != nil {
					assert.True(t, reflect.DeepEqual(tt.expectedBannedUser.Labels, got.Labels))
				}
			}
		})
	}
}

func TestGetBannedUser(t *testing.T) {
	userSignup1 := commonsignup.NewUserSignup(commonsignup.WithName("johny"), commonsignup.WithEmail("johny@example.com"))
	userSignup2 := commonsignup.NewUserSignup(commonsignup.WithName("bob"), commonsignup.WithEmail("bob@example.com"))
	userSignup3 := commonsignup.NewUserSignup(commonsignup.WithName("oliver"), commonsignup.WithEmail("oliver@example.com"))
	bannedUser1, err := NewBannedUser(userSignup1, "admin", "")
	require.NoError(t, err)
	bannedUser2, err := NewBannedUser(userSignup2, "admin", "")
	require.NoError(t, err)
	bannedUser3, err := NewBannedUser(userSignup3, "admin", "")
	require.NoError(t, err)

	mockT := test.NewMockT()
	fakeClient := test.NewFakeClient(mockT, bannedUser1)
	ctx := context.TODO()

	tests := []struct {
		name       string
		toBan      *toolchainv1alpha1.BannedUser
		wantResult *toolchainv1alpha1.BannedUser
		wantError  bool
		fakeClient *test.FakeClient
	}{
		{
			name:       "user is already banned",
			toBan:      bannedUser1,
			wantResult: bannedUser1,
			wantError:  false,
			fakeClient: fakeClient,
		},
		{
			name:       "user is not banned",
			toBan:      bannedUser2,
			wantResult: nil,
			wantError:  false,
			fakeClient: fakeClient,
		},
		{
			name:       "cannot list banned users because the client does have type v1alpha1.BannedUserList registered in the scheme",
			toBan:      bannedUser3,
			wantResult: nil,
			wantError:  true,
			fakeClient: &test.FakeClient{Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(), T: t},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := GetBannedUser(ctx, tt.toBan.Labels[toolchainv1alpha1.BannedUserEmailHashLabelKey], tt.fakeClient, test.HostOperatorNs)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, gotResult)
			}
		})
	}
}
