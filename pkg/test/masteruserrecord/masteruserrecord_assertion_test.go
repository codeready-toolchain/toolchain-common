package masteruserrecord_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	murtest "github.com/codeready-toolchain/toolchain-common/pkg/test/masteruserrecord"
	"github.com/stretchr/testify/assert"
)

func TestHasAnnotationWithValue(t *testing.T) {

	t.Run("should succeed when annotation with value exists", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john", murtest.WithAnnotation("toolchain/email", "john@example.com"))
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "john@example.com")

		// then: all good
		assert.False(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.False(t, mockT.CalledErrorf())
	})

	t.Run("should fail when there is no annotations", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john")
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "john@example.com")

		// then: all good
		assert.True(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.True(t, mockT.CalledErrorf())
	})

	t.Run("should fail when annotation does not exist", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john", murtest.WithAnnotation("other/stuff", "whatever"))
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "john@example.com")

		// then: all good
		assert.True(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.True(t, mockT.CalledErrorf())
	})

	t.Run("should fail when annotation exists with different value", func(t *testing.T) {
		// given
		mur := murtest.NewMasterUserRecord(t, "john", murtest.WithAnnotation("toolchain/email", "john@example.com"))
		mockT := test.NewMockT()
		client := test.NewFakeClient(mockT, mur)

		// when
		murtest.AssertThatMasterUserRecord(mockT, "john", client).HasAnnotationWithValue("toolchain/email", "other@example.com")

		// then: all good
		assert.False(t, mockT.CalledFailNow())
		assert.False(t, mockT.CalledFatalf())
		assert.True(t, mockT.CalledErrorf())
	})
}
