package test

import (
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	conditions "github.com/codeready-toolchain/toolchain-common/pkg/condition"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// AssertConditionsMatch asserts that the specified list A of conditions is equal to specified
// list B of conditions ignoring the order of the elements. We can't use assert.ElementsMatch
// because the LastTransitionTime of the actual conditions can be modified but the conditions
// still should be treated as matched
func AssertConditionsMatch(t T, actual []toolchainv1alpha1.Condition, expected ...toolchainv1alpha1.Condition) {
	AssertThat(t, actual, Has(AllConditionsLike(expected...)))
}

// AssertContainsCondition asserts that the specified list of conditions contains the specified condition.
// LastTransitionTime is ignored.
func AssertContainsCondition(t T, conditions []toolchainv1alpha1.Condition, contains toolchainv1alpha1.Condition) {
	AssertThat(t, conditions, Has(SomeConditionThat(contains.Type, HasStatus(contains.Status), HasReason(contains.Reason), HasMessage(contains.Message))))
}

// AssertConditionsMatchAndRecentTimestamps asserts that the specified list of conditions match AND asserts that the timestamps are recent
func AssertConditionsMatchAndRecentTimestamps(t T, actual []toolchainv1alpha1.Condition, expected ...toolchainv1alpha1.Condition) {
	cutoff := time.Now().Add(-5 * time.Second)
	expectedMap := map[toolchainv1alpha1.ConditionType][]Predicate[toolchainv1alpha1.Condition]{}
	for _, c := range expected {
		expectedMap[c.Type] = []Predicate[toolchainv1alpha1.Condition]{IsLike(c), HasTransitionTimeLaterThan(cutoff), HasUpdateTimeLaterThan(cutoff)}
	}
	AssertThat(t, actual, Has(AllConditions(expectedMap)))
}

// ConditionsMatch returns true if the specified list A of conditions is equal to specified
// list B of conditions ignoring the order of the elements
func ConditionsMatch(actual []toolchainv1alpha1.Condition, expected ...toolchainv1alpha1.Condition) bool {
	return AllConditionsLike(expected...).Matches(actual)
}

// ContainsCondition returns true if the specified list of conditions contains the specified condition.
// LastTransitionTime is ignored.
func ContainsCondition(conditions []toolchainv1alpha1.Condition, contains toolchainv1alpha1.Condition) bool {
	return SomeConditionThat(contains.Type, IsLike(contains)).Matches(conditions)
}

func SomeConditionThat(conditionType toolchainv1alpha1.ConditionType, preds ...Predicate[toolchainv1alpha1.Condition]) Predicate[[]toolchainv1alpha1.Condition] {
	return &conditionsPredicate{conditionType: conditionType, predicates: preds}
}

func SomeConditionLike(expected toolchainv1alpha1.Condition) Predicate[[]toolchainv1alpha1.Condition] {
	return SomeConditionThat(expected.Type, IsLike(expected))
}

func AllConditions(conditions map[toolchainv1alpha1.ConditionType][]Predicate[toolchainv1alpha1.Condition]) Predicate[[]toolchainv1alpha1.Condition] {
	return &allConditionsLikePredicate{conditions: conditions}
}

func AllConditionsLike(expected ...toolchainv1alpha1.Condition) Predicate[[]toolchainv1alpha1.Condition] {
	expectedMap := map[toolchainv1alpha1.ConditionType][]Predicate[toolchainv1alpha1.Condition]{}
	for _, c := range expected {
		expectedMap[c.Type] = []Predicate[toolchainv1alpha1.Condition]{IsLike(c)}
	}
	return &allConditionsLikePredicate{conditions: expectedMap}
}

func BridgeToConditions[T client.Object](accessor func(T) *[]toolchainv1alpha1.Condition, pred Predicate[[]toolchainv1alpha1.Condition]) Predicate[T] {
	return &bridgePredicate[T]{accessor: accessor, pred: pred}
}

func IsLike(cond toolchainv1alpha1.Condition) Predicate[toolchainv1alpha1.Condition] {
	return &likePredicate{condition: cond}
}

func IsTrue() Predicate[toolchainv1alpha1.Condition] {
	return HasStatus(corev1.ConditionTrue)
}

func IsNotTrue() Predicate[toolchainv1alpha1.Condition] {
	return HasStatusDifferentFrom(corev1.ConditionTrue)
}

func IsFalse() Predicate[toolchainv1alpha1.Condition] {
	return HasStatus(corev1.ConditionFalse)
}

func IsNotFalse() Predicate[toolchainv1alpha1.Condition] {
	return HasStatusDifferentFrom(corev1.ConditionFalse)
}

func IsUnknown() Predicate[toolchainv1alpha1.Condition] {
	return HasStatus(corev1.ConditionUnknown)
}

func IsNotUnknown() Predicate[toolchainv1alpha1.Condition] {
	return HasStatusDifferentFrom(corev1.ConditionUnknown)
}

func HasStatus(status corev1.ConditionStatus) Predicate[toolchainv1alpha1.Condition] {
	return &conditionPredicate{expectedStatus: &status}
}

func HasStatusDifferentFrom(status corev1.ConditionStatus) Predicate[toolchainv1alpha1.Condition] {
	return &conditionPredicate{expectedStatus: &status, negate: true}
}

func HasReason(reason string) Predicate[toolchainv1alpha1.Condition] {
	return &conditionPredicate{expectedReason: pointer.String(reason)}
}

func HasMessage(reason string) Predicate[toolchainv1alpha1.Condition] {
	return &conditionPredicate{expectedMessage: pointer.String(reason)}
}

func HasRecentTransitionTime() Predicate[toolchainv1alpha1.Condition] {
	return HasTransitionTimeLaterThan(time.Now().Add(-5 * time.Second))
}

func HasRecentUpdateTime() Predicate[toolchainv1alpha1.Condition] {
	return HasUpdateTimeLaterThan(time.Now().Add(-5 * time.Second))
}

func HasTransitionTimeLaterThan(t time.Time) Predicate[toolchainv1alpha1.Condition] {
	return &recencyPredicate{oldestAllowed: t, transition: true}
}

func HasUpdateTimeLaterThan(t time.Time) Predicate[toolchainv1alpha1.Condition] {
	return &recencyPredicate{oldestAllowed: t, transition: false}
}

type conditionsPredicate struct {
	conditionType toolchainv1alpha1.ConditionType
	predicates    []Predicate[toolchainv1alpha1.Condition]
}

// Matches implements Predicate.
func (c *conditionsPredicate) Matches(conds []toolchainv1alpha1.Condition) bool {
	condition, found := conditions.FindConditionByType(conds, c.conditionType)
	if !found {
		return false
	}

	for _, predicate := range c.predicates {
		if !predicate.Matches(condition) {
			return false
		}
	}
	return true
}

func (c *conditionsPredicate) FixToMatch(conds []toolchainv1alpha1.Condition) []toolchainv1alpha1.Condition {
	var found bool
	var index int
	var condition toolchainv1alpha1.Condition

	if conds == nil {
		conds = []toolchainv1alpha1.Condition{}
	} else {
		copy := make([]toolchainv1alpha1.Condition, len(conds))
		for i, c := range conds {
			copy[i] = c
		}
		conds = copy
	}

	for i, cond := range conds {
		if cond.Type == c.conditionType {
			found = true
			index = i
			condition = cond
			break
		}
	}

	if !found {
		conds = append(conds, toolchainv1alpha1.Condition{})
		index = len(conds) - 1
		condition.Type = c.conditionType
	}

	for _, predicate := range c.predicates {
		if p, ok := predicate.(PredicateMatchFixer[toolchainv1alpha1.Condition]); ok {
			condition = p.FixToMatch(condition)
		}
	}

	conds[index] = condition

	return conds
}

type conditionPredicate struct {
	expectedStatus  *corev1.ConditionStatus
	expectedReason  *string
	expectedMessage *string
	expectedType    toolchainv1alpha1.ConditionType
	negate          bool
}

func (c *conditionPredicate) Matches(cond toolchainv1alpha1.Condition) bool {
	if c.expectedType != "" && cond.Type != c.expectedType {
		return c.negate
	}
	if c.expectedStatus != nil && cond.Status != *c.expectedStatus {
		return c.negate
	}
	if c.expectedReason != nil && cond.Reason != *c.expectedReason {
		return c.negate
	}
	if c.expectedMessage != nil && cond.Message != *c.expectedMessage {
		return c.negate
	}

	return !c.negate
}

func (c *conditionPredicate) FixToMatch(cond toolchainv1alpha1.Condition) toolchainv1alpha1.Condition {
	if c.expectedType != "" {
		if c.negate {
			cond.Type = "<different from: " + c.expectedType + ">"
		} else {
			cond.Type = c.expectedType
		}
	}
	if c.expectedStatus != nil {
		if c.negate {
			cond.Status = "<different from: " + *c.expectedStatus + ">"
		} else {
			cond.Status = *c.expectedStatus
		}
	}
	if c.expectedReason != nil {
		if c.negate {
			cond.Reason = "<different from: " + *c.expectedReason + ">"
		} else {
			cond.Reason = *c.expectedReason
		}
	}
	if c.expectedMessage != nil {
		if c.negate {
			cond.Message = "<different from: " + *c.expectedMessage + ">"
		} else {
			cond.Message = *c.expectedMessage
		}
	}

	return cond
}

type bridgePredicate[T client.Object] struct {
	accessor func(T) *[]toolchainv1alpha1.Condition
	pred     Predicate[[]toolchainv1alpha1.Condition]
}

func (c *bridgePredicate[T]) Matches(obj T) bool {
	conds := c.accessor(obj)
	return c.pred.Matches(*conds)
}

func (c *bridgePredicate[T]) FixToMatch(obj T) T {
	if p, ok := c.pred.(PredicateMatchFixer[[]toolchainv1alpha1.Condition]); ok {
		obj = obj.DeepCopyObject().(T)
		conds := c.accessor(obj)
		*conds = p.FixToMatch(*conds)
	}
	return obj
}

type recencyPredicate struct {
	oldestAllowed time.Time
	transition    bool
}

func (r *recencyPredicate) Matches(cond toolchainv1alpha1.Condition) bool {
	var condTime time.Time
	if r.transition {
		condTime = cond.LastTransitionTime.Time
	} else if cond.LastUpdatedTime != nil {
		condTime = cond.LastUpdatedTime.Time
	} else {
		// we're looking for a recent update time, but there was no update
		return false
	}

	return condTime.After(r.oldestAllowed)
}

func (r *recencyPredicate) FixToMatch(cond toolchainv1alpha1.Condition) toolchainv1alpha1.Condition {
	if r.transition {
		cond.LastTransitionTime.Time = r.oldestAllowed
	} else {
		cond.LastUpdatedTime = &metav1.Time{Time: r.oldestAllowed}
	}
	return cond
}

type likePredicate struct {
	condition toolchainv1alpha1.Condition
}

func (r *likePredicate) Matches(cond toolchainv1alpha1.Condition) bool {
	return r.condition.Type == cond.Type && r.condition.Status == cond.Status && r.condition.Reason == cond.Reason && r.condition.Message == cond.Message
}

func (r *likePredicate) FixToMatch(cond toolchainv1alpha1.Condition) toolchainv1alpha1.Condition {
	cond.Type = r.condition.Type
	cond.Status = r.condition.Status
	cond.Reason = r.condition.Reason
	cond.Message = r.condition.Message
	return cond
}

type allConditionsLikePredicate struct {
	conditions map[toolchainv1alpha1.ConditionType][]Predicate[toolchainv1alpha1.Condition]
}

func (r *allConditionsLikePredicate) Matches(conds []toolchainv1alpha1.Condition) bool {
	if len(conds) != len(r.conditions) {
		return false
	}

	for _, cond := range conds {
		preds, ok := r.conditions[cond.Type]
		if !ok {
			return false
		}

		for _, p := range preds {
			if !p.Matches(cond) {
				return false
			}
		}
	}
	return true
}

func (r *allConditionsLikePredicate) FixToMatch(conds []toolchainv1alpha1.Condition) []toolchainv1alpha1.Condition {
	remainingTypes := make(map[toolchainv1alpha1.ConditionType]bool, len(r.conditions))
	for t := range r.conditions {
		remainingTypes[t] = true
	}

	fixed := []toolchainv1alpha1.Condition{}

	fix := func(cond toolchainv1alpha1.Condition, preds []Predicate[toolchainv1alpha1.Condition]) toolchainv1alpha1.Condition {
		for _, p := range preds {
			if p, ok := p.(PredicateMatchFixer[toolchainv1alpha1.Condition]); ok {
				cond = p.FixToMatch(cond)
			}
		}
		return cond
	}

	for _, cond := range conds {
		preds, ok := r.conditions[cond.Type]
		if !ok {
			// we don't add the condition to the fixed ones, effectively removing it
			continue
		}
		cond = fix(cond, preds)
		fixed = append(fixed, cond)
		delete(remainingTypes, cond.Type)
	}

	for t := range remainingTypes {
		preds := r.conditions[t]
		cond := toolchainv1alpha1.Condition{}
		cond.Type = t
		cond = fix(cond, preds)
		fixed = append(fixed, cond)
	}

	return fixed
}

var (
	_ Predicate[[]toolchainv1alpha1.Condition]           = (*conditionsPredicate)(nil)
	_ PredicateMatchFixer[[]toolchainv1alpha1.Condition] = (*conditionsPredicate)(nil)
	_ Predicate[[]toolchainv1alpha1.Condition]           = (*allConditionsLikePredicate)(nil)
	_ PredicateMatchFixer[[]toolchainv1alpha1.Condition] = (*allConditionsLikePredicate)(nil)
	_ Predicate[toolchainv1alpha1.Condition]             = (*conditionPredicate)(nil)
	_ PredicateMatchFixer[toolchainv1alpha1.Condition]   = (*conditionPredicate)(nil)
	_ Predicate[toolchainv1alpha1.Condition]             = (*recencyPredicate)(nil)
	_ PredicateMatchFixer[toolchainv1alpha1.Condition]   = (*recencyPredicate)(nil)
	_ Predicate[toolchainv1alpha1.Condition]             = (*likePredicate)(nil)
	_ PredicateMatchFixer[toolchainv1alpha1.Condition]   = (*likePredicate)(nil)
)
