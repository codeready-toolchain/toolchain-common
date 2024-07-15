package banneduser

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	commonsignup "github.com/codeready-toolchain/toolchain-common/pkg/test/usersignup"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewBannedUser(t *testing.T) {
	userSignup1 := commonsignup.NewUserSignup(commonsignup.WithName("johny"))
	userSignup1UserEmailHashLabelKey := userSignup1.Labels[toolchainv1alpha1.UserSignupUserEmailHashLabelKey]

	userSignup2 := commonsignup.NewUserSignup(commonsignup.WithName("bob"))
	userSignup2.Labels = map[string]string{}

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
			name:               "userSignup without email hash label",
			userSignup:         userSignup2,
			bannedBy:           "admin",
			wantError:          true,
			wantErrorMsg:       fmt.Sprintf("the UserSignup %s doesn't have the label '%s' set", userSignup2.Name, toolchainv1alpha1.UserSignupUserEmailHashLabelKey),
			expectedBannedUser: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewBannedUser(test.userSignup, test.bannedBy)

			if test.wantError {
				require.Error(t, err)
				assert.Equal(t, test.wantErrorMsg, err.Error())
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)

				assert.Equal(t, test.expectedBannedUser.ObjectMeta.Namespace, got.ObjectMeta.Namespace)
				assert.Equal(t, test.expectedBannedUser.ObjectMeta.Name, got.ObjectMeta.Name)
				assert.Equal(t, test.expectedBannedUser.Spec.Email, got.Spec.Email)
				reflect.DeepEqual(test.expectedBannedUser.ObjectMeta.Labels, got.ObjectMeta.Labels)
			}
		})
	}
}

func TestIsAlreadyBanned(t *testing.T) {
	userSignup1 := commonsignup.NewUserSignup(commonsignup.WithName("johny"))
	userSignup2 := commonsignup.NewUserSignup(commonsignup.WithName("bob"), commonsignup.WithEmail("bob@example.com"))
	bannedUser1, err := NewBannedUser(userSignup1, "admin")
	require.NoError(t, err)
	bannedUser2, err := NewBannedUser(userSignup2, "admin")
	require.NoError(t, err)

	mockT := test.NewMockT()
	fakeClient := test.NewFakeClient(mockT, bannedUser1)
	ctx := context.TODO()

	tests := []struct {
		name       string
		toBan      *toolchainv1alpha1.BannedUser
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "user is already banned",
			toBan:      bannedUser1,
			wantResult: true,
		},
		{
			name:       "user is not banned",
			toBan:      bannedUser2,
			wantResult: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := IsAlreadyBanned(ctx, tt.toBan, fakeClient, test.HostOperatorNs)

			fmt.Println("gotResult", gotResult)

			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, gotResult)
		})
	}
}
