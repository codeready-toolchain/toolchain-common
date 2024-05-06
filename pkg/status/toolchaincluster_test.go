package status

import (
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var fakeToolchainClusterReason = "AToolchainClusterReason"
var fakeToolchainClusterMsg = "AToolchainClusterMsg"

var log = logf.Log.WithName("toolchaincluster_test")

func TestGetToolchainClusterConditions(t *testing.T) {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))
	t.Run("test ToolchainCluster conditions", func(t *testing.T) {
		t.Run("condition ready", func(t *testing.T) {
			// given
			readyAttrs := ToolchainClusterAttributes{
				GetClusterFunc: newGetHostClusterReady(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  "HostConnectionReady",
				Message: "",
			}

			// when
			conditions := GetToolchainClusterConditions(log, readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			require.NoError(t, err)
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster not ok", func(t *testing.T) {
			// given
			msg := "the cluster connection was not found"
			readyAttrs := ToolchainClusterAttributes{
				GetClusterFunc: newGetHostClusterNotOk(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "ToolchainClusterNotFound",
				Message: msg,
			}

			// when
			conditions := GetToolchainClusterConditions(log, readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			require.Error(t, err)
			assert.Equal(t, msg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster ok but not ready", func(t *testing.T) {
			// given
			readyAttrs := ToolchainClusterAttributes{
				GetClusterFunc: newGetHostClusterOkButNotReady(fakeToolchainClusterMsg),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "HostConnectionNotReady",
				Message: fakeToolchainClusterMsg,
			}

			// when
			conditions := GetToolchainClusterConditions(log, readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			require.Error(t, err)
			assert.Equal(t, fakeToolchainClusterMsg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster ok but not ready - no message", func(t *testing.T) {
			// given
			msg := "the cluster connection is not ready"
			readyAttrs := ToolchainClusterAttributes{
				GetClusterFunc: newGetHostClusterOkButNotReady(""),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "HostConnectionNotReady",
				Message: msg,
			}

			// when
			conditions := GetToolchainClusterConditions(log, readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			require.Error(t, err)
			assert.Equal(t, msg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition cluster ok but no ready condition", func(t *testing.T) {
			// given
			msg := "the cluster connection is not ready"
			readyAttrs := ToolchainClusterAttributes{
				GetClusterFunc: newGetHostClusterOkWithClusterOfflineCondition(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "HostConnectionNotReady",
				Message: msg,
			}

			// when
			conditions := GetToolchainClusterConditions(log, readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			require.Error(t, err)
			assert.Equal(t, msg, err.Error())
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("condition last probe time exceeded", func(t *testing.T) {
			// given
			msg := "exceeded the maximum duration since the last probe: 13s"
			readyAttrs := ToolchainClusterAttributes{
				GetClusterFunc: newGetHostClusterLastProbeTimeExceeded(),
				Period:         10 * time.Second,
				Timeout:        3 * time.Second,
			}
			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "ToolchainClusterLastProbeTimeExceeded",
				Message: msg,
			}

			// when
			conditions := GetToolchainClusterConditions(log, readyAttrs)
			err := ValidateComponentConditionReady(conditions...)

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), msg)
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})
	})
}

func newGetHostClusterReady() cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(true, toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue, metav1.Now(), fakeToolchainClusterReason, "")
}

func newGetHostClusterNotOk() cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(false, toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionFalse, metav1.Now(), fakeToolchainClusterReason, fakeToolchainClusterMsg)
}

func newGetHostClusterOkButNotReady(message string) cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(true, toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionFalse, metav1.Now(), fakeToolchainClusterReason, message)
}

func newGetHostClusterOkWithClusterOfflineCondition() cluster.GetHostClusterFunc {
	return NewFakeGetHostCluster(true, toolchainv1alpha1.ToolchainClusterOffline, corev1.ConditionFalse, metav1.Now(), fakeToolchainClusterReason, fakeToolchainClusterMsg)
}

func newGetHostClusterLastProbeTimeExceeded() cluster.GetHostClusterFunc {
	tenMinsAgo := metav1.Now().Add(time.Duration(-10) * time.Minute)
	return NewFakeGetHostCluster(true, toolchainv1alpha1.ToolchainClusterReady, corev1.ConditionTrue, metav1.NewTime(tenMinsAgo), fakeToolchainClusterReason, fakeToolchainClusterMsg)
}

// NewGetHostCluster returns cluster.GetHostClusterFunc function. The cluster.CachedToolchainCluster
// that is returned by the function then contains the given client and the given status and lastProbeTime.
// If ok == false, then the function returns nil for the cluster.
func NewFakeGetHostCluster(ok bool, conditionType toolchainv1alpha1.ToolchainClusterConditionType, status corev1.ConditionStatus, lastProbeTime metav1.Time, reason, message string) cluster.GetHostClusterFunc {
	if !ok {
		return func() (*cluster.CachedToolchainCluster, bool) {
			return nil, false
		}
	}
	return func() (*cluster.CachedToolchainCluster, bool) {
		toolchainClusterValue := &cluster.CachedToolchainCluster{
			Config: &cluster.Config{
				OperatorNamespace: test.HostOperatorNs,
				OwnerClusterName:  test.MemberClusterName,
			},
			ClusterStatus: &toolchainv1alpha1.ToolchainClusterStatus{
				Conditions: []toolchainv1alpha1.ToolchainClusterCondition{{
					Type:            conditionType,
					Reason:          reason,
					Status:          status,
					LastProbeTime:   lastProbeTime,
					LastUpdatedTime: &lastProbeTime,
				}},
			},
		}
		if message != "" {
			toolchainClusterValue.ClusterStatus.Conditions[0].Message = message
		}

		return toolchainClusterValue, true
	}
}
