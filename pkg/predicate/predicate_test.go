package predicate

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var missingDataEvents = []event.UpdateEvent{
	{},
	{ObjectNew: &toolchainv1alpha1.UserAccount{}},
	{ObjectOld: &toolchainv1alpha1.UserAccount{}}}

func TestOnlyUpdateWhenGenerationNotChangedPredicate(t *testing.T) {
	var noGenChangedPred = EitherUpdateWhenGenerationNotChangedOrDelete{}

	t.Run("update event", func(t *testing.T) {

		t.Run("when missing data", func(t *testing.T) {
			// given
			for _, event := range missingDataEvents {
				// when
				ok := noGenChangedPred.Update(event)

				// then
				assert.False(t, ok)
			}
		})

		t.Run("when generation changed", func(t *testing.T) {
			// given
			updateEvent := event.UpdateEvent{
				ObjectNew: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789)}},
				ObjectOld: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(987654321)}}}

			// when
			ok := noGenChangedPred.Update(updateEvent)

			// then
			assert.False(t, ok)
		})

		t.Run("when generation not changed", func(t *testing.T) {
			// given
			updateEvent := event.UpdateEvent{
				ObjectNew: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789)}},
				ObjectOld: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789)}}}

			// when
			ok := noGenChangedPred.Update(updateEvent)

			// then
			assert.True(t, ok)
		})

	})

	t.Run("create event returns false", func(t *testing.T) {
		// given
		createEvent := event.CreateEvent{
			Object: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789)}}}

		// when
		ok := noGenChangedPred.Create(createEvent)

		// then
		assert.False(t, ok)
	})

	t.Run("delete event returns true", func(t *testing.T) {
		// given
		deleteEvent := event.DeleteEvent{
			Object: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789)}}}

		// when
		ok := noGenChangedPred.Delete(deleteEvent)

		// then
		assert.True(t, ok)
	})

	t.Run("generic event returns false", func(t *testing.T) {
		// given
		genericEvent := event.GenericEvent{
			Object: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789)}}}

		// when
		ok := noGenChangedPred.Generic(genericEvent)

		// then
		assert.False(t, ok)
	})
}

func TestLabelsAndGenerationPredicate(t *testing.T) {
	var labelsAndGenPred = LabelsAndGenerationPredicate{}

	t.Run("update event", func(t *testing.T) {

		t.Run("when missing data", func(t *testing.T) {
			// given
			for _, event := range missingDataEvents {
				// when
				ok := labelsAndGenPred.Update(event)

				// then
				assert.False(t, ok)
			}
		})

		t.Run("when no changes", func(t *testing.T) {
			// given
			updateEvent := event.UpdateEvent{
				ObjectNew: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789), Labels: map[string]string{"test": "label"}}},
				ObjectOld: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789), Labels: map[string]string{"test": "label"}}}}

			// when
			ok := labelsAndGenPred.Update(updateEvent)

			// then
			assert.False(t, ok)
		})

		t.Run("when generation changed", func(t *testing.T) {
			// given
			updateEvent := event.UpdateEvent{
				ObjectNew: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789), Labels: map[string]string{"test": "label"}}},
				ObjectOld: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(987654321), Labels: map[string]string{"test": "label"}}}}

			// when
			ok := labelsAndGenPred.Update(updateEvent)

			// then
			assert.True(t, ok)
		})

		t.Run("when labels changed", func(t *testing.T) {
			// given
			updateEvent := event.UpdateEvent{
				ObjectNew: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789), Labels: map[string]string{"test": "label"}}},
				ObjectOld: &toolchainv1alpha1.UserAccount{ObjectMeta: metav1.ObjectMeta{Generation: int64(123456789), Labels: map[string]string{"test": "label", "another": "newlabel"}}}}

			// when
			ok := labelsAndGenPred.Update(updateEvent)

			// then
			assert.True(t, ok)
		})
	})
}
