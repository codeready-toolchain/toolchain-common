package banneduser

import (
	"context"
	"fmt"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	commonsignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewBannedUser(t *testing.T) {
	userSignup1 := commonsignup.NewUserSignup(commonsignup.WithName("johny"), commonsignup.WithEmail("jonhy@example.com"))
	userSignup1UserEmailHashLabelKey := userSignup1.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey]

	userSignup2 := commonsignup.NewUserSignup(commonsignup.WithName("bob"), commonsignup.WithEmail("bob@example.com"))
	userSignup2.Labels = map[string]string{}

	userSignup3 := commonsignup.NewUserSignup(commonsignup.WithName("oliver"), commonsignup.WithEmail("oliver@example.com"))
	userSignup3PhoneHashLabelKey := "fd276563a8232d16620da8ec85d0575f"
	userSignup3.Labels[toolchainv1alpha1.UserSignupUserPhoneHashLabelKey] = userSignup3PhoneHashLabelKey
	userSignup3UserEmailHashLabelKey := userSignup3.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey]

	tests := []struct {
		name               string
		userSignup         *toolchainv1alpha1.UserSignup
		bannedBy           string
		wantError          bool
		wantErrorMsg       string
		expectedBannedUser *toolchainv1alpha1.BannedUser
	}{
		{
			name:         "userSignup with email hash label",
			userSignup:   userSignup1,
			bannedBy:     "admin",
			wantError:    false,
			wantErrorMsg: "",
			expectedBannedUser: &toolchainv1alpha1.BannedUser{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: userSignup1.Namespace,
					Name:      fmt.Sprintf("banneduser-%s", userSignup1UserEmailHashLabelKey),
					Labels: map[string]string{
						toolchainv1alpha1.BannedUserEmailHashLabelKey: userSignup1UserEmailHashLabelKey,
						bannedByLabel: "admin",
					},
				},
				Spec: toolchainv1alpha1.BannedUserSpec{
					Email: userSignup1.Spec.IdentityClaims.Email,
				},
			},
		},
		{
			name:               "userSignup without email hash label and phone hash label",
			userSignup:         userSignup2,
			bannedBy:           "admin",
			wantError:          true,
			wantErrorMsg:       fmt.Sprintf("the UserSignup %s doesn't have the label '%s' set", userSignup2.Name, toolchainv1alpha1.UserSignupUserEmailHashLabelKey),
			expectedBannedUser: nil,
		},
		{
			name:         "userSignup with email hash label and phone hash label",
			userSignup:   userSignup3,
			bannedBy:     "admin",
			wantError:    false,
			wantErrorMsg: "",
			expectedBannedUser: &toolchainv1alpha1.BannedUser{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: userSignup3.Namespace,
					Name:      fmt.Sprintf("banneduser-%s", userSignup3UserEmailHashLabelKey),
					Labels: map[string]string{
						toolchainv1alpha1.BannedUserEmailHashLabelKey: userSignup3UserEmailHashLabelKey,
						bannedByLabel: "admin",
						toolchainv1alpha1.UserSignupUserPhoneHashLabelKey: userSignup3PhoneHashLabelKey,
					},
				},
				Spec: toolchainv1alpha1.BannedUserSpec{
					Email: userSignup3.Spec.IdentityClaims.Email,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBannedUser(tt.userSignup, tt.bannedBy)

			if tt.wantError {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrorMsg, err.Error())
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)

				assert.Equal(t, tt.expectedBannedUser.ObjectMeta.Namespace, got.ObjectMeta.Namespace)
				assert.Equal(t, tt.expectedBannedUser.ObjectMeta.Name, got.ObjectMeta.Name)
				assert.Equal(t, tt.expectedBannedUser.Spec.Email, got.Spec.Email)

				if tt.expectedBannedUser != nil && !compareMaps(tt.expectedBannedUser.Labels, got.Labels) {
					t.Errorf("compareMaps(%v, %v) = false, expected = true", tt.expectedBannedUser.Labels, got.Labels)
				}
			}
		})
	}
}

func TestIsAlreadyBanned(t *testing.T) {
	userSignup1 := commonsignup.NewUserSignup(commonsignup.WithName("johny"), commonsignup.WithEmail("johny@example.com"))
	userSignup2 := commonsignup.NewUserSignup(commonsignup.WithName("bob"), commonsignup.WithEmail("bob@example.com"))
	userSignup3 := commonsignup.NewUserSignup(commonsignup.WithName("oliver"), commonsignup.WithEmail("oliver@example.com"))
	bannedUser1, err := NewBannedUser(userSignup1, "admin")
	require.NoError(t, err)
	bannedUser2, err := NewBannedUser(userSignup2, "admin")
	require.NoError(t, err)
	bannedUser3, err := NewBannedUser(userSignup3, "admin")
	require.NoError(t, err)

	mockT := test.NewMockT()
	fakeClient := test.NewFakeClient(mockT, bannedUser1)
	ctx := context.TODO()

	tests := []struct {
		name       string
		toBan      *toolchainv1alpha1.BannedUser
		wantResult bool
		wantError  bool
		fakeClient *test.FakeClient
	}{
		{
			name:       "user is already banned",
			toBan:      bannedUser1,
			wantResult: true,
			wantError:  false,
			fakeClient: fakeClient,
		},
		{
			name:       "user is not banned",
			toBan:      bannedUser2,
			wantResult: false,
			wantError:  false,
			fakeClient: fakeClient,
		},
		{
			name:       "cannot list banned users because the client does have type v1alpha1.BannedUserList registered in the scheme",
			toBan:      bannedUser3,
			wantResult: false,
			wantError:  true,
			fakeClient: &test.FakeClient{Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(), T: t},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := IsAlreadyBanned(ctx, tt.toBan, tt.fakeClient, test.HostOperatorNs)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, gotResult)
			}
		})
	}
}

func compareMaps(map1, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, value1 := range map1 {
		value2, ok := map2[key]
		if !ok {
			return false
		}
		if value1 != value2 {
			return false
		}
	}

	return true
}
