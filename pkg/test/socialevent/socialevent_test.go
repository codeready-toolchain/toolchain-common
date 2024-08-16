package socialevent_test

import (
	"testing"
	"time"

	"github.com/codeready-toolchain/toolchain-common/pkg/socialevent"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testsocialevent "github.com/codeready-toolchain/toolchain-common/pkg/test/socialevent"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSocialEvent(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName())
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier) // default
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)     // default
		assert.Equal(t, 10, e.Spec.MaxAttendees)         // default
		assert.Equal(t, 0, e.Status.ActivationCount)     // default
	})

	t.Run("with custom user tier", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithUserTier("deactivate80"))
		// then
		assert.Equal(t, "deactivate80", e.Spec.UserTier) // custom
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)     // default
		assert.Equal(t, 10, e.Spec.MaxAttendees)         // default
		assert.Equal(t, 0, e.Status.ActivationCount)     // default
	})

	t.Run("with custom space tier", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithSpaceTier("base1ns6didler"))
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier)    // default
		assert.Equal(t, "base1ns6didler", e.Spec.SpaceTier) // custom
		assert.Equal(t, 10, e.Spec.MaxAttendees)            // default
		assert.Equal(t, 0, e.Status.ActivationCount)        // default
	})

	t.Run("with custom max attendees", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithMaxAttendees(5))
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier) // default
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)     // default
		assert.Equal(t, 5, e.Spec.MaxAttendees)          // custom
		assert.Equal(t, 0, e.Status.ActivationCount)     // default
	})

	t.Run("with custom activation count", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithActivationCount(1))
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier) // default
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)     // default
		assert.Equal(t, 10, e.Spec.MaxAttendees)         // default
		assert.Equal(t, 1, e.Status.ActivationCount)     // custom
	})

	t.Run("with custom start time", func(t *testing.T) {
		// given
		start := time.Now().Add(-10 * time.Hour)
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithStartTime(start))
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier)         // default
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)             // default
		assert.Equal(t, 10, e.Spec.MaxAttendees)                 // default
		assert.Equal(t, 0, e.Status.ActivationCount)             // default
		assert.Equal(t, metav1.NewTime(start), e.Spec.StartTime) // custom
	})

	t.Run("with custom end time", func(t *testing.T) {
		// given
		end := time.Now().Add(10 * time.Hour)
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithEndTime(end))
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier)     // default
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)         // default
		assert.Equal(t, 10, e.Spec.MaxAttendees)             // default
		assert.Equal(t, 0, e.Status.ActivationCount)         // default
		assert.Equal(t, metav1.NewTime(end), e.Spec.EndTime) // custom
	})

	t.Run("with custom target cluster", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithTargetCluster("member-1"))
		// then
		assert.Equal(t, "deactivate30", e.Spec.UserTier)  // default
		assert.Equal(t, "base1ns", e.Spec.SpaceTier)      // default
		assert.Equal(t, 10, e.Spec.MaxAttendees)          // default
		assert.Equal(t, 0, e.Status.ActivationCount)      // default
		assert.Equal(t, "member-1", e.Spec.TargetCluster) // custom
	})
}
