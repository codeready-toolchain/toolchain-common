package status

import (
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	kubefed_v1beta1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var fakeKubefedReason = "AKubefedReason"
var fakeKubefedMsg = "AKubefedMsg"

func TestGetKubefedConditions(t *testing.T) {
	t.Run("test kubefed conditions", func(t *testing.T) {
		t.Run("condition ready", func(t *testing.T) {
			// given
			readyAttrs := KubefedAttributes{
				GetClusterFunc: newGetHostClusterReady(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
				Threshold:      3,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  "HostConnectionReady",
				Message: "",
			}

			// when
			conditions := GetKubefedConditions(readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			assert.NoError(t, err)
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster not ok", func(t *testing.T) {
			// given
			msg := "the cluster connection was not found"
			readyAttrs := KubefedAttributes{
				GetClusterFunc: newGetHostClusterNotOk(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
				Threshold:      3,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "KubefedNotFound",
				Message: msg,
			}

			// when
			conditions := GetKubefedConditions(readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			assert.Error(t, err)
			assert.Equal(t, msg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster ok but not ready", func(t *testing.T) {
			// given
			readyAttrs := KubefedAttributes{
				GetClusterFunc: newGetHostClusterOkButNotReady(&fakeKubefedMsg),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
				Threshold:      3,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "HostConnectionNotReady",
				Message: fakeKubefedMsg,
			}

			// when
			conditions := GetKubefedConditions(readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			assert.Error(t, err)
			assert.Equal(t, fakeKubefedMsg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster ok but not ready - no message", func(t *testing.T) {
			// given
			msg := "the cluster connection is not ready"
			readyAttrs := KubefedAttributes{
				GetClusterFunc: newGetHostClusterOkButNotReady(nil),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
				Threshold:      3,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "HostConnectionNotReady",
				Message: msg,
			}

			// when
			conditions := GetKubefedConditions(readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			assert.Error(t, err)
			assert.Equal(t, msg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster ok but no ready condition", func(t *testing.T) {
			// given
			msg := "the cluster connection is not ready"
			readyAttrs := KubefedAttributes{
				GetClusterFunc: newGetHostClusterOkWithClusterOfflineCondition(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
				Threshold:      3,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "HostConnectionNotReady",
				Message: msg,
			}

			// when
			conditions := GetKubefedConditions(readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			assert.Error(t, err)
			assert.Equal(t, msg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition last probe time exceeded", func(t *testing.T) {
			// given
			msg := "exceeded the maximum duration since the last probe: 39s"
			readyAttrs := KubefedAttributes{
				GetClusterFunc: newGetHostClusterLastProbeTimeExceeded(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
				Threshold:      3,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "KubefedLastProbeTimeExceeded",
				Message: msg,
			}

			// when
			conditions := GetKubefedConditions(readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), msg)
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})
	})
}

func newGetHostClusterReady() cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(true, common.ClusterReady, corev1.ConditionTrue, metav1.Now(), &fakeKubefedReason, nil)
}

func newGetHostClusterNotOk() cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(false, common.ClusterReady, corev1.ConditionFalse, metav1.Now(), &fakeKubefedReason, &fakeKubefedMsg)
}

func newGetHostClusterOkButNotReady(message *string) cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(true, common.ClusterReady, corev1.ConditionFalse, metav1.Now(), &fakeKubefedReason, message)
}

func newGetHostClusterOkWithClusterOfflineCondition() cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(true, common.ClusterOffline, corev1.ConditionFalse, metav1.Now(), &fakeKubefedReason, &fakeKubefedMsg)
}

func newGetHostClusterLastProbeTimeExceeded() cluster.GetHostClusterFunc {
	tenMinsAgo := metav1.Now().Add(time.Duration(-10) * time.Minute)
	return NewFakeGetHostCluster(true, common.ClusterReady, corev1.ConditionTrue, metav1.NewTime(tenMinsAgo), &fakeKubefedReason, &fakeKubefedMsg)
}

// NewGetHostCluster returns cluster.GetHostClusterFunc function. The cluster.FedCluster
// that is returned by the function then contains the given client and the given status and lastProbeTime.
// If ok == false, then the function returns nil for the cluster.
func NewFakeGetHostCluster(ok bool, conditionType common.ClusterConditionType, status corev1.ConditionStatus, lastProbeTime metav1.Time, reason, message *string) cluster.GetHostClusterFunc {
	if !ok {
		return func() (*cluster.FedCluster, bool) {
			return nil, false
		}
	}
	return func() (*cluster.FedCluster, bool) {
		fedClusterValue := &cluster.FedCluster{
			Type:              cluster.Host,
			OperatorNamespace: test.HostOperatorNs,
			OwnerClusterName:  test.MemberClusterName,
			ClusterStatus: &kubefed_v1beta1.KubeFedClusterStatus{
				Conditions: []kubefed_v1beta1.ClusterCondition{{
					Type:          conditionType,
					Reason:        reason,
					Status:        status,
					LastProbeTime: lastProbeTime,
				}},
			},
		}
		if message != nil && *message != "" {
			fedClusterValue.ClusterStatus.Conditions[0].Message = message
		}

		return fedClusterValue, true
	}
}
