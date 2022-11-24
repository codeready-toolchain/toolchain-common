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
		assert.Equal(t, e.Spec.UserTier, "deactivate30") // default
		assert.Equal(t, e.Spec.SpaceTier, "base1ns")     // default
		assert.Equal(t, e.Spec.MaxAttendees, 10)         // default
		assert.Equal(t, e.Status.ActivationCount, 0)     // default
	})

	t.Run("with custom user tier", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithUserTier("deactivate80"))
		// then
		assert.Equal(t, e.Spec.UserTier, "deactivate80") // custom
		assert.Equal(t, e.Spec.SpaceTier, "base1ns")     // default
		assert.Equal(t, e.Spec.MaxAttendees, 10)         // default
		assert.Equal(t, e.Status.ActivationCount, 0)     // default
	})

	t.Run("with custom space tier", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithSpaceTier("base1ns6didler"))
		// then
		assert.Equal(t, e.Spec.UserTier, "deactivate30")    // default
		assert.Equal(t, e.Spec.SpaceTier, "base1ns6didler") // custom
		assert.Equal(t, e.Spec.MaxAttendees, 10)            // default
		assert.Equal(t, e.Status.ActivationCount, 0)        // default
	})

	t.Run("with custom max attendees", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithMaxAttendees(5))
		// then
		assert.Equal(t, e.Spec.UserTier, "deactivate30") // default
		assert.Equal(t, e.Spec.SpaceTier, "base1ns")     // default
		assert.Equal(t, e.Spec.MaxAttendees, 5)          // custom
		assert.Equal(t, e.Status.ActivationCount, 0)     // default
	})

	t.Run("with custom activation count", func(t *testing.T) {
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithActivationCount(1))
		// then
		assert.Equal(t, e.Spec.UserTier, "deactivate30") // default
		assert.Equal(t, e.Spec.SpaceTier, "base1ns")     // default
		assert.Equal(t, e.Spec.MaxAttendees, 10)         // default
		assert.Equal(t, e.Status.ActivationCount, 1)     // custom
	})

	t.Run("with custom start time", func(t *testing.T) {
		// given
		start := time.Now().Add(-10 * time.Hour)
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithStartTime(start))
		// then
		assert.Equal(t, e.Spec.UserTier, "deactivate30")         // default
		assert.Equal(t, e.Spec.SpaceTier, "base1ns")             // default
		assert.Equal(t, e.Spec.MaxAttendees, 10)                 // default
		assert.Equal(t, e.Status.ActivationCount, 0)             // default
		assert.Equal(t, e.Spec.StartTime, metav1.NewTime(start)) // custom
	})

	t.Run("with custom end time", func(t *testing.T) {
		// given
		end := time.Now().Add(10 * time.Hour)
		// when
		e := testsocialevent.NewSocialEvent(test.HostOperatorNs, socialevent.NewName(), testsocialevent.WithEndTime(end))
		// then
		assert.Equal(t, e.Spec.UserTier, "deactivate30")     // default
		assert.Equal(t, e.Spec.SpaceTier, "base1ns")         // default
		assert.Equal(t, e.Spec.MaxAttendees, 10)             // default
		assert.Equal(t, e.Status.ActivationCount, 0)         // default
		assert.Equal(t, e.Spec.EndTime, metav1.NewTime(end)) // custom
	})

}
