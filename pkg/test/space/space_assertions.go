package space

import (
	"context"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/hash"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"golang.org/x/exp/slices"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Assertion struct {
	space          *toolchainv1alpha1.Space
	client         runtimeclient.Client
	namespacedName types.NamespacedName
	t              test.T
	spaceRequest   *toolchainv1alpha1.SpaceRequest
	parentSpace    *toolchainv1alpha1.Space
}

func (a *Assertion) loadResource() error {
	if a.spaceRequest != nil && a.parentSpace != nil {
		// we are testing a spaceRequest scenario
		return a.loadSubSpace()
	}

	// default space test scenario
	space := &toolchainv1alpha1.Space{}
	err := a.client.Get(context.TODO(), a.namespacedName, space)
	a.space = space
	return err
}

// AssertThatSpace helper func to begin with the assertions on a Space
func AssertThatSpace(t test.T, namespace, name string, client runtimeclient.Client) *Assertion {
	return &Assertion{
		client:         client,
		namespacedName: test.NamespacedName(namespace, name),
		t:              t,
	}
}

func (a *Assertion) Get() *toolchainv1alpha1.Space {
	err := a.loadResource()
	require.NoError(a.t, err)
	return a.space
}

func (a *Assertion) Exists() *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	return a
}

func (a *Assertion) DoesNotExist() *Assertion {
	err := a.loadResource()
	require.Error(a.t, err)
	require.True(a.t, errors.IsNotFound(err))
	return a
}

func (a *Assertion) HasFinalizer() *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Contains(a.t, a.space.Finalizers, toolchainv1alpha1.FinalizerName)
	return a
}

func (a *Assertion) HasNoFinalizers() *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Empty(a.t, a.space.Finalizers)
	return a
}

func (a *Assertion) HasTier(tierName string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, tierName, a.space.Spec.TierName)
	return a
}

func (a *Assertion) HasDisableInheritance(disableInheritance bool) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, disableInheritance, a.space.Spec.DisableInheritance)
	return a
}

func (a *Assertion) HasParentSpace(parentSpaceName string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, parentSpaceName, a.space.Spec.ParentSpace)
	value, found := a.space.Labels[toolchainv1alpha1.ParentSpaceLabelKey]
	require.True(a.t, found)
	assert.Equal(a.t, parentSpaceName, value)
	return a
}

func (a *Assertion) HasLabelWithValue(key, value string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	require.NotNil(a.t, a.space.Labels)
	assert.Equal(a.t, value, a.space.Labels[key])
	return a
}

func (a *Assertion) HasAnnotationWithValue(key, value string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	require.NotNil(a.t, a.space.Annotations)
	assert.Equal(a.t, value, a.space.Annotations[key])
	return a
}

func (a *Assertion) DoesNotHaveAnnotation(key string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	require.NotContains(a.t, a.space.Annotations, key)
	return a
}

func (a *Assertion) HasMatchingTierLabelForTier(tier *toolchainv1alpha1.NSTemplateTier) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	key := hash.TemplateTierHashLabelKey(tier.Name)
	require.Contains(a.t, a.space.Labels, key)
	expectedHash, err := hash.ComputeHashForNSTemplateTier(tier)
	require.NoError(a.t, err)
	assert.Equal(a.t, expectedHash, a.space.Labels[key])
	return a
}

func (a *Assertion) HasStateLabel(stateValue string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	require.NotNil(a.t, a.space.Labels)
	assert.Equal(a.t, stateValue, a.space.Labels[toolchainv1alpha1.SpaceStateLabelKey])
	return a
}

func (a *Assertion) DoesNotHaveLabel(key string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	require.NotContains(a.t, a.space.Labels, key)
	return a
}

func (a *Assertion) HasNoSpecTargetCluster() *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Empty(a.t, a.space.Spec.TargetCluster)
	return a
}

func (a *Assertion) HasSpecTargetCluster(targetCluster string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, targetCluster, a.space.Spec.TargetCluster)
	return a
}

func (a *Assertion) HasSpecTargetClusterRoles(roles []string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, roles, a.space.Spec.TargetClusterRoles)
	return a
}

func (a *Assertion) HasNoStatusTargetCluster() *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Empty(a.t, a.space.Status.TargetCluster)
	return a
}

func (a *Assertion) HasStatusTargetCluster(targetCluster string) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, targetCluster, a.space.Status.TargetCluster)
	return a
}

func (a *Assertion) HasStatusProvisionedNamespaces(provisionedNamespaces []toolchainv1alpha1.SpaceNamespace) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Equal(a.t, provisionedNamespaces, a.space.Status.ProvisionedNamespaces)
	return a
}

func (a *Assertion) HasConditions(expected ...toolchainv1alpha1.Condition) *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	test.AssertConditionsMatch(a.t, a.space.Status.Conditions, expected...)
	return a
}

func (a *Assertion) HasNoConditions() *Assertion {
	err := a.loadResource()
	require.NoError(a.t, err)
	assert.Empty(a.t, a.space.Status.Conditions)
	return a
}

func Provisioning() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   toolchainv1alpha1.ConditionReady,
		Status: corev1.ConditionFalse,
		Reason: toolchainv1alpha1.SpaceProvisioningReason,
	}
}

func ProvisioningPending(msg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.SpaceProvisioningPendingReason,
		Message: msg,
	}
}

func ProvisioningFailed(msg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.SpaceProvisioningFailedReason,
		Message: msg,
	}
}

func Retargeting() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   toolchainv1alpha1.ConditionReady,
		Status: corev1.ConditionFalse,
		Reason: toolchainv1alpha1.SpaceRetargetingReason,
	}
}

func RetargetingFailed(msg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.SpaceRetargetingFailedReason,
		Message: msg,
	}
}

func Updating() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   toolchainv1alpha1.ConditionReady,
		Status: corev1.ConditionFalse,
		Reason: toolchainv1alpha1.SpaceUpdatingReason,
	}
}

func UnableToCreateNSTemplateSet(msg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.SpaceUnableToCreateNSTemplateSetReason,
		Message: msg,
	}
}

func UnableToUpdateNSTemplateSet(msg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.SpaceUnableToUpdateNSTemplateSetReason,
		Message: msg,
	}
}

func Ready() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   toolchainv1alpha1.ConditionReady,
		Status: corev1.ConditionTrue,
		Reason: toolchainv1alpha1.SpaceProvisionedReason,
	}
}

func Terminating() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   toolchainv1alpha1.ConditionReady,
		Status: corev1.ConditionFalse,
		Reason: toolchainv1alpha1.SpaceTerminatingReason,
	}
}

func TerminatingFailed(msg string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  toolchainv1alpha1.SpaceTerminatingFailedReason,
		Message: msg,
	}
}

// Assertions on multiple Spaces at once
type SpacesAssertion struct {
	spaces    *toolchainv1alpha1.SpaceList
	client    runtimeclient.Client
	namespace string
	t         test.T
}

func AssertThatSpaces(t test.T, client runtimeclient.Client) *SpacesAssertion {
	return &SpacesAssertion{
		client:    client,
		namespace: test.HostOperatorNs,
		t:         t,
	}
}

func (a *SpacesAssertion) loadSpaces() error {
	spaces := &toolchainv1alpha1.SpaceList{}
	err := a.client.List(context.TODO(), spaces, runtimeclient.InNamespace(a.namespace))
	a.spaces = spaces
	return err
}

func (a *SpacesAssertion) HaveCount(count int) *SpacesAssertion {
	err := a.loadSpaces()
	require.NoError(a.t, err)
	require.Len(a.t, a.spaces.Items, count)
	return a
}

func AssertThatSubSpace(t test.T, client runtimeclient.Client, spaceRequest *toolchainv1alpha1.SpaceRequest, parentSpace *toolchainv1alpha1.Space) *Assertion {
	return &Assertion{
		t:            t,
		client:       client,
		spaceRequest: spaceRequest,
		parentSpace:  parentSpace,
	}
}

func (a *Assertion) loadSubSpace() error {
	spaces := &toolchainv1alpha1.SpaceList{}
	spaceRequestLabel := runtimeclient.MatchingLabels{
		toolchainv1alpha1.SpaceRequestLabelKey:          a.spaceRequest.GetName(),
		toolchainv1alpha1.SpaceRequestNamespaceLabelKey: a.spaceRequest.GetNamespace(),
		toolchainv1alpha1.ParentSpaceLabelKey:           a.parentSpace.GetName(),
	}
	err := a.client.List(context.TODO(), spaces, spaceRequestLabel, runtimeclient.InNamespace(a.parentSpace.GetNamespace()))
	if len(spaces.Items) > 0 {
		a.space = &spaces.Items[0]
	}
	return err
}

///////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////
/////// Example conversion to predicates //////////////////////////
///////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////

func Finalizer() test.Predicate[*toolchainv1alpha1.Space] {
	return &hasFinalizer{}
}

func NoFinalizer() test.Predicate[*toolchainv1alpha1.Space] {
	return &hasFinalizer{negate: true}
}

func Tier(tierName string) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasTier{name: tierName}
}

func DisableInheritance(disabled bool) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasInheritance{disabled: disabled}
}

func ParentSpace(parentSpaceName string) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasParentSpace{name: parentSpaceName}
}

// predicates on labels and annotations are implemented generically, so we don't have
// to redefine and reimplement them here...

func MatchingTierLabelForTier(tier *toolchainv1alpha1.NSTemplateTier) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasMatchingTierLabel{tier: tier}
}

func StateLabel(value string) test.Predicate[client.Object] {
	return test.Labels(map[string]string{toolchainv1alpha1.SpaceStateLabelKey: value})
}

func TargetCluster(cluster string) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasTargetCluster{cluster: cluster}
}

func TargetClusterRoles(roles []string) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasTargetClusterRoles{roles: roles}
}

func StatusTargetCluster(cluster string) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasStatusTargetCluster{cluster: cluster}
}

func ProvisionedNamespaces(provisionedNamespaces []toolchainv1alpha1.SpaceNamespace) test.Predicate[*toolchainv1alpha1.Space] {
	return &hasProvisionedNamespaces{namespaces: provisionedNamespaces}
}

// condition predicates are done generically

func Conditions(pred test.Predicate[[]toolchainv1alpha1.Condition]) test.Predicate[*toolchainv1alpha1.Space] {
	return test.BridgeToConditions(func(s *toolchainv1alpha1.Space) *[]toolchainv1alpha1.Condition {
		return &s.Status.Conditions
	}, pred)
}

// impls

type (
	hasFinalizer         struct{ negate bool }
	hasTier              struct{ name string }
	hasInheritance       struct{ disabled bool }
	hasParentSpace       struct{ name string }
	hasMatchingTierLabel struct {
		tier *toolchainv1alpha1.NSTemplateTier
	}
	hasTargetCluster         struct{ cluster string }
	hasTargetClusterRoles    struct{ roles []string }
	hasStatusTargetCluster   struct{ cluster string }
	hasProvisionedNamespaces struct {
		namespaces []toolchainv1alpha1.SpaceNamespace
	}
)

func (p *hasFinalizer) Matches(s *toolchainv1alpha1.Space) bool {
	return slices.Contains(s.Finalizers, toolchainv1alpha1.FinalizerName)
}

func (p *hasFinalizer) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	if slices.Contains(s.Finalizers, toolchainv1alpha1.FinalizerName) != p.negate {
		s = s.DeepCopy()
		if p.negate {
			s.Finalizers = slices.DeleteFunc(s.Finalizers, func(e string) bool {
				return e == toolchainv1alpha1.FinalizerName
			})
		} else {
			s.Finalizers = append(s.Finalizers, toolchainv1alpha1.FinalizerName)
		}
	}
	return s
}

func (p *hasTier) Matches(s *toolchainv1alpha1.Space) bool {
	return s.Spec.TierName == p.name
}

func (p *hasTier) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Spec.TierName = p.name
	return s
}

func (p *hasInheritance) Matches(s *toolchainv1alpha1.Space) bool {
	return s.Spec.DisableInheritance == p.disabled
}

func (p *hasInheritance) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Spec.DisableInheritance = p.disabled
	return s
}

func (p *hasParentSpace) Matches(s *toolchainv1alpha1.Space) bool {
	return s.Spec.ParentSpace == p.name
}

func (p *hasParentSpace) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Spec.ParentSpace = p.name
	return s
}

func (p *hasMatchingTierLabel) Matches(s *toolchainv1alpha1.Space) bool {
	key, hash := p.labelKeyAndHash()
	return s.Labels[key] == hash
}

func (p *hasMatchingTierLabel) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	key, hash := p.labelKeyAndHash()
	if s.Labels == nil {
		s.Labels = map[string]string{}
	}
	s.Labels[key] = hash
	return s
}

func (p *hasMatchingTierLabel) labelKeyAndHash() (string, string) {
	key := hash.TemplateTierHashLabelKey(p.tier.Name)
	// TODO: ignoring the error is not ideal - maybe we could support failable predicates?
	hash, _ := hash.ComputeHashForNSTemplateTier(p.tier)
	return key, hash
}

func (p *hasTargetCluster) Matches(s *toolchainv1alpha1.Space) bool {
	return s.Spec.TargetCluster == p.cluster
}

func (p *hasTargetCluster) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Spec.TargetCluster = p.cluster
	return s
}

func (p *hasTargetClusterRoles) Matches(s *toolchainv1alpha1.Space) bool {
	return slices.Equal(s.Spec.TargetClusterRoles, p.roles)
}

func (p *hasTargetClusterRoles) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Spec.TargetClusterRoles = p.roles
	return s
}

func (p *hasStatusTargetCluster) Matches(s *toolchainv1alpha1.Space) bool {
	return s.Status.TargetCluster == p.cluster
}

func (p *hasStatusTargetCluster) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Status.TargetCluster = p.cluster
	return s
}

func (p *hasProvisionedNamespaces) Matches(s *toolchainv1alpha1.Space) bool {
	return slices.Equal(s.Status.ProvisionedNamespaces, p.namespaces)
}

func (p *hasProvisionedNamespaces) FixToMatch(s *toolchainv1alpha1.Space) *toolchainv1alpha1.Space {
	s = s.DeepCopy()
	s.Status.ProvisionedNamespaces = p.namespaces
	return s
}
