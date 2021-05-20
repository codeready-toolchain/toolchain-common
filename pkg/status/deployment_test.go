package status

import (
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetDeploymentStatusConditions(t *testing.T) {

	t.Run("test deployment status conditions", func(t *testing.T) {

		t.Run("deployment ready", func(t *testing.T) {
			fakeClient := test.NewFakeClient(t, fakeDeploymentReady())
			conditions := GetDeploymentStatusConditions(fakeClient, "test-deployment", test.HostOperatorNs)
			err := ValidateComponentConditionReady(conditions...)
			require.NoError(t, err)

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionTrue,
				Reason:  "DeploymentReady",
				Message: "",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("deployment does not exist", func(t *testing.T) {
			fakeClient := test.NewFakeClient(t)
			conditions := GetDeploymentStatusConditions(fakeClient, "test-deployment", test.HostOperatorNs)
			err := ValidateComponentConditionReady(conditions...)
			require.Error(t, err)

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "DeploymentNotFound",
				Message: "unable to get the deployment: deployments.apps \"test-deployment\" not found",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("deployment not available", func(t *testing.T) {
			fakeClient := test.NewFakeClient(t, fakeDeploymentNotAvailable())
			conditions := GetDeploymentStatusConditions(fakeClient, "test-deployment", test.HostOperatorNs)
			err := ValidateComponentConditionReady(conditions...)
			require.Error(t, err)

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "DeploymentNotReady",
				Message: "deployment has unready status conditions: Available",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})

		t.Run("deployment not progressing", func(t *testing.T) {
			fakeClient := test.NewFakeClient(t, fakeDeploymentNotProgressing())
			conditions := GetDeploymentStatusConditions(fakeClient, "test-deployment", test.HostOperatorNs)
			err := ValidateComponentConditionReady(conditions...)
			require.Error(t, err)

			expected := toolchainv1alpha1.Condition{
				Type:    toolchainv1alpha1.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "DeploymentNotReady",
				Message: "deployment has unready status conditions: Progressing",
			}
			test.AssertConditionsMatchAndRecentTimestamps(t, conditions, expected)
		})
	})
}

func fakeDeploymentNotAvailable() *appsv1.Deployment {
	return newFakeDeployment("test-deployment", test.HostOperatorNs, DeploymentNotAvailableCondition(), DeploymentProgressingCondition())
}

func fakeDeploymentNotProgressing() *appsv1.Deployment {
	return newFakeDeployment("test-deployment", test.HostOperatorNs, DeploymentAvailableCondition(), DeploymentNotProgressingCondition())
}

func fakeDeploymentReady() *appsv1.Deployment {
	return newFakeDeployment("test-deployment", test.HostOperatorNs, DeploymentAvailableCondition(), DeploymentProgressingCondition())
}

func newFakeDeployment(name, namespace string, deploymentConditions ...appsv1.DeploymentCondition) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			Conditions: deploymentConditions,
		},
	}
}
