package useraccount_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	murtest "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"
	uatest "github.com/codeready-toolchain/toolchain-common/pkg/test/useraccount"

	"github.com/stretchr/testify/assert"
)

func TestUserAccountFromMur(t *testing.T) {
	t.Run("UserAccount from MUR should have its own NSTemplateSet", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john")
		userAcc := uatest.NewUserAccountFromMur(mur)

		// when
		murtest.ModifyUaInMur(mur, test.MemberClusterName, murtest.UserAccountTierName("admin"))

		// then
		assert.Equal(t, "admin", mur.Spec.UserAccounts[0].Spec.NSTemplateSet.TierName) // modified
		assert.Equal(t, "basic", userAcc.Spec.NSTemplateSet.TierName)                  // unmodified
	})
}
