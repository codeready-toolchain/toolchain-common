package idler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/codeready-toolchain/toolchain-common/pkg/owners"
	testcommon "github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/fake"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	fakescale "k8s.io/client-go/scale/fake"
	clienttest "k8s.io/client-go/testing"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const apiEndpoint = "https://api.openshift.com:6443"

var customListKinds = map[schema.GroupVersionResource]string{
	{Group: "serving.kserve.io", Version: "v1beta1", Resource: "inferenceservices"}: "InferenceServiceList",
}

func TestIdleOwnerUnsupportedKind(t *testing.T) {
	idler, _ := newTestIdler(t)
	owner := &owners.ObjectWithGVR{
		Object: &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "cm",
					"namespace": "ns",
				},
			},
		},
		GVR: &schema.GroupVersionResource{Version: "v1", Resource: "configmaps"},
	}

	err := idler.IdleOwner(context.TODO(), owner, Options{})
	require.ErrorIs(t, err, ErrUnsupportedKind)
}

func TestIdleOwnerKindMatrix(t *testing.T) {
	replicas := int32(3)
	ns := "test-ns"

	tests := []struct {
		name       string
		kind       string
		prepare    func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, scalesClient *fakescale.FakeScaleClient) *owners.ObjectWithGVR
		opts       Options
		assertIdle func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, scalesClient *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR)
	}{
		{
			name: "Deployment",
			kind: "Deployment",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
					Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
				}, appsv1.SchemeGroupVersion.WithResource("deployments"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				assertScaledToZero(t, dynamicClient, *owner.GVR, ns, owner.Object.GetName())
			},
		},
		{
			name: "ReplicaSet",
			kind: "ReplicaSet",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: ns},
					Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
				}, appsv1.SchemeGroupVersion.WithResource("replicasets"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				assertScaledToZero(t, dynamicClient, *owner.GVR, ns, owner.Object.GetName())
			},
		},
		{
			name: "StatefulSet",
			kind: "StatefulSet",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: "sts", Namespace: ns},
					Spec:       appsv1.StatefulSetSpec{Replicas: &replicas},
				}, appsv1.SchemeGroupVersion.WithResource("statefulsets"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				assertScaledToZero(t, dynamicClient, *owner.GVR, ns, owner.Object.GetName())
			},
		},
		{
			name: "ReplicationController",
			kind: "ReplicationController",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &corev1.ReplicationController{
					ObjectMeta: metav1.ObjectMeta{Name: "rc", Namespace: ns},
					Spec:       corev1.ReplicationControllerSpec{Replicas: &replicas},
				}, corev1.SchemeGroupVersion.WithResource("replicationcontrollers"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				assertScaledToZero(t, dynamicClient, *owner.GVR, ns, owner.Object.GetName())
			},
		},
		{
			name: "Integration",
			kind: "Integration",
			prepare: func(t *testing.T, _ *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "camel.apache.org/v1",
					"kind":       "Integration",
					"metadata":   map[string]any{"name": "integ", "namespace": ns},
					"spec":       map[string]any{"replicas": int64(3)},
				}}
				gvr := schema.GroupVersionResource{Group: "camel.apache.org", Version: "v1", Resource: "integrations"}
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, _ *fakedynamic.FakeDynamicClient, scalesClient *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				assertScalePatched(t, scalesClient, owner.Object.GetName(), owner.Object.GetNamespace())
			},
		},
		{
			name: "KameletBinding",
			kind: "KameletBinding",
			prepare: func(t *testing.T, _ *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "camel.apache.org/v1alpha1",
					"kind":       "KameletBinding",
					"metadata":   map[string]any{"name": "kb", "namespace": ns},
					"spec":       map[string]any{"replicas": int64(3)},
				}}
				gvr := schema.GroupVersionResource{Group: "camel.apache.org", Version: "v1alpha1", Resource: "kameletbindings"}
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, _ *fakedynamic.FakeDynamicClient, scalesClient *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				assertScalePatched(t, scalesClient, owner.Object.GetName(), owner.Object.GetNamespace())
			},
		},
		{
			name: "DeploymentConfig",
			kind: "DeploymentConfig",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "apps.openshift.io/v1",
					"kind":       "DeploymentConfig",
					"metadata":   map[string]any{"name": "dc", "namespace": ns},
					"spec":       map[string]any{"replicas": int64(3), "paused": true},
				}}
				gvr := schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
				_, err := dynamicClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, metav1.CreateOptions{})
				require.NoError(t, err)
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				got, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.NoError(t, err)
				replicasVal, _, err := unstructured.NestedInt64(got.Object, "spec", "replicas")
				require.NoError(t, err)
				assert.Equal(t, int64(0), replicasVal)
				paused, _, err := unstructured.NestedBool(got.Object, "spec", "paused")
				require.NoError(t, err)
				assert.False(t, paused)
			},
		},
		{
			name: "DaemonSet",
			kind: "DaemonSet",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{Name: "ds", Namespace: ns},
				}, appsv1.SchemeGroupVersion.WithResource("daemonsets"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				_, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err), "expected NotFound, got %v", err)
			},
		},
		{
			name: "Job",
			kind: "Job",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{Name: "job", Namespace: ns},
				}, batchv1.SchemeGroupVersion.WithResource("jobs"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				_, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err), "expected NotFound, got %v", err)
			},
		},
		{
			name: "PersistentVolumeClaim",
			kind: "PersistentVolumeClaim",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				return createAndGetOwner(t, dynamicClient, &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: "pvc", Namespace: ns},
				}, corev1.SchemeGroupVersion.WithResource("persistentvolumeclaims"))
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				_, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err), "expected NotFound, got %v", err)
			},
		},
		{
			name: "DataVolume",
			kind: "DataVolume",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "cdi.kubevirt.io/v1beta1",
					"kind":       "DataVolume",
					"metadata":   map[string]any{"name": "dv", "namespace": ns},
				}}
				gvr := schema.GroupVersionResource{Group: "cdi.kubevirt.io", Version: "v1beta1", Resource: "datavolumes"}
				_, err := dynamicClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, metav1.CreateOptions{})
				require.NoError(t, err)
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				_, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.True(t, apierrors.IsNotFound(err), "expected NotFound, got %v", err)
			},
		},
		{
			name: "AnsibleAutomationPlatform",
			kind: "AnsibleAutomationPlatform",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "aap.ansible.com/v1alpha1",
					"kind":       "AnsibleAutomationPlatform",
					"metadata":   map[string]any{"name": "aap", "namespace": ns},
					"spec":       map[string]any{"idle_aap": false},
				}}
				gvr := schema.GroupVersionResource{Group: "aap.ansible.com", Version: "v1alpha1", Resource: "ansibleautomationplatforms"}
				_, err := dynamicClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, metav1.CreateOptions{})
				require.NoError(t, err)
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				got, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.NoError(t, err)
				idled, found, err := unstructured.NestedBool(got.Object, "spec", "idle_aap")
				require.NoError(t, err)
				require.True(t, found)
				assert.True(t, idled)
			},
		},
		{
			name: "Claw",
			kind: "Claw",
			prepare: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "claw.sandbox.redhat.com/v1alpha1",
					"kind":       "Claw",
					"metadata":   map[string]any{"name": "claw", "namespace": ns},
					"spec":       map[string]any{"idle": false},
				}}
				gvr := schema.GroupVersionResource{Group: "claw.sandbox.redhat.com", Version: "v1alpha1", Resource: "claws"}
				_, err := dynamicClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, metav1.CreateOptions{})
				require.NoError(t, err)
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, owner *owners.ObjectWithGVR) {
				got, err := dynamicClient.Resource(*owner.GVR).Namespace(ns).Get(context.TODO(), owner.Object.GetName(), metav1.GetOptions{})
				require.NoError(t, err)
				idled, found, err := unstructured.NestedBool(got.Object, "spec", "idle")
				require.NoError(t, err)
				require.True(t, found)
				assert.True(t, idled)
			},
		},
		{
			name: "VirtualMachine",
			kind: "VirtualMachine",
			prepare: func(t *testing.T, _ *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient) *owners.ObjectWithGVR {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "kubevirt.io/v1",
					"kind":       "VirtualMachine",
					"metadata":   map[string]any{"name": "vm", "namespace": ns},
				}}
				gvr := schema.GroupVersionResource{Group: "kubevirt.io", Version: "v1", Resource: "virtualmachines"}
				return &owners.ObjectWithGVR{Object: obj, GVR: &gvr}
			},
			assertIdle: func(t *testing.T, _ *fakedynamic.FakeDynamicClient, _ *fakescale.FakeScaleClient, _ *owners.ObjectWithGVR) {
				// stop call asserted via no error + gock expectation
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idler, clients := newTestIdler(t)
			if tc.kind == "VirtualMachine" {
				gock.New(apiEndpoint).
					Put("/apis/subresources.kubevirt.io/v1/namespaces/test-ns/virtualmachines/vm/stop").
					Reply(http.StatusOK)
			}
			owner := tc.prepare(t, clients.dynamicClient, clients.scalesClient)
			err := idler.IdleOwner(context.TODO(), owner, tc.opts)
			require.NoError(t, err)
			tc.assertIdle(t, clients.dynamicClient, clients.scalesClient, owner)
		})
	}
}

func TestIdleServingRuntimeCutoff(t *testing.T) {
	ns := "test-ns"
	srGVR := schema.GroupVersionResource{Group: "serving.kserve.io", Version: "v1alpha1", Resource: "servingruntimes"}
	isvcGVR := schema.GroupVersionResource{Group: "serving.kserve.io", Version: "v1beta1", Resource: "inferenceservices"}

	newServingRuntimeOwner := func() *owners.ObjectWithGVR {
		return &owners.ObjectWithGVR{
			Object: &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "serving.kserve.io/v1alpha1",
				"kind":       "ServingRuntime",
				"metadata":   map[string]any{"name": "sr", "namespace": ns},
			}},
			GVR: &srGVR,
		}
	}

	createInferenceService := func(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, name string, age time.Duration) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "serving.kserve.io/v1beta1",
			"kind":       "InferenceService",
			"metadata": map[string]any{
				"name":      name,
				"namespace": ns,
			},
		}}
		obj.SetName(name)
		obj.SetNamespace(ns)
		obj.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-age)))
		obj.SetAPIVersion("serving.kserve.io/v1beta1")
		obj.SetKind("InferenceService")
		_, err := dynamicClient.Resource(isvcGVR).Namespace(ns).Create(context.TODO(), obj, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	t.Run("timeout 0 deletes nothing", func(t *testing.T) {
		idler, clients := newTestIdler(t)
		createInferenceService(t, clients.dynamicClient, "old-isvc", 2*time.Hour)
		err := idler.IdleOwner(context.TODO(), newServingRuntimeOwner(), Options{TimeoutSeconds: 0})
		require.NoError(t, err)
		_, err = clients.dynamicClient.Resource(isvcGVR).Namespace(ns).Get(context.TODO(), "old-isvc", metav1.GetOptions{})
		require.NoError(t, err)
	})

	t.Run("positive timeout deletes only old InferenceServices", func(t *testing.T) {
		idler, clients := newTestIdler(t)
		createInferenceService(t, clients.dynamicClient, "old-isvc", 2*time.Hour)
		createInferenceService(t, clients.dynamicClient, "new-isvc", 1*time.Minute)
		err := idler.IdleOwner(context.TODO(), newServingRuntimeOwner(), Options{TimeoutSeconds: 3600})
		require.NoError(t, err)
		_, err = clients.dynamicClient.Resource(isvcGVR).Namespace(ns).Get(context.TODO(), "old-isvc", metav1.GetOptions{})
		require.Error(t, err)
		_, err = clients.dynamicClient.Resource(isvcGVR).Namespace(ns).Get(context.TODO(), "new-isvc", metav1.GetOptions{})
		require.NoError(t, err)
	})
}

func TestIdleFromPod(t *testing.T) {
	ns := "test-ns"
	replicas := int32(3)

	t.Run("single known owner", func(t *testing.T) {
		idler, clients := newTestIdler(t)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		}
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "app-rs", Namespace: ns},
			Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: ns}}
		require.NoError(t, controllerruntime.SetControllerReference(deployment, rs, scheme.Scheme))
		require.NoError(t, controllerruntime.SetControllerReference(rs, pod, scheme.Scheme))
		createTyped(t, clients.dynamicClient, deployment)
		createTyped(t, clients.dynamicClient, rs)

		kind, name, err := idler.IdleFromPod(context.TODO(), pod, Options{})
		require.NoError(t, err)
		assert.Equal(t, "Deployment", kind)
		assert.Equal(t, "app", name)
		assertScaledToZero(t, clients.dynamicClient, appsv1.SchemeGroupVersion.WithResource("deployments"), ns, "app")
		// second owner (ReplicaSet) is also idled by on-demand path when present
		assertScaledToZero(t, clients.dynamicClient, appsv1.SchemeGroupVersion.WithResource("replicasets"), ns, "app-rs")
	})

	t.Run("unknown-only chain returns empty", func(t *testing.T) {
		idler, clients := newTestIdler(t)
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm-owner", Namespace: ns}}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: ns}}
		require.NoError(t, controllerruntime.SetControllerReference(cm, pod, scheme.Scheme))
		createTyped(t, clients.dynamicClient, cm)

		kind, name, err := idler.IdleFromPod(context.TODO(), pod, Options{})
		require.NoError(t, err)
		assert.Empty(t, kind)
		assert.Empty(t, name)
	})

	t.Run("joins errors from two known owners", func(t *testing.T) {
		idler, clients := newTestIdler(t)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		}
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "app-rs", Namespace: ns},
			Spec:       appsv1.ReplicaSetSpec{Replicas: &replicas},
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: ns}}
		require.NoError(t, controllerruntime.SetControllerReference(deployment, rs, scheme.Scheme))
		require.NoError(t, controllerruntime.SetControllerReference(rs, pod, scheme.Scheme))
		createTyped(t, clients.dynamicClient, deployment)
		createTyped(t, clients.dynamicClient, rs)

		clients.dynamicClient.PrependReactor("patch", "deployments", func(action clienttest.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("deploy patch failed")
		})
		clients.dynamicClient.PrependReactor("patch", "replicasets", func(action clienttest.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("rs patch failed")
		})

		kind, name, err := idler.IdleFromPod(context.TODO(), pod, Options{})
		require.EqualError(t, err, "deploy patch failed\nrs patch failed")
		assert.Equal(t, "Deployment", kind)
		assert.Equal(t, "app", name)
	})

	t.Run("returns GetOwners error when no owner was attempted", func(t *testing.T) {
		dynamicClient := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, customListKinds)
		scalesClient := &fakescale.FakeScaleClient{}
		restClient, err := testcommon.NewRESTClient("dummy-token", apiEndpoint)
		require.NoError(t, err)
		// Discovery has no apps/v1 resources, so owner walk fails for Deployment.
		fakeDiscovery := newFakeDiscoveryClient()
		idler := New(fakeDiscovery, dynamicClient, scalesClient, restClient)

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: ns}}
		require.NoError(t, controllerruntime.SetControllerReference(deployment, pod, scheme.Scheme))

		kind, name, err := idler.IdleFromPod(context.TODO(), pod, Options{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no resource found for kind Deployment")
		assert.Empty(t, kind)
		assert.Empty(t, name)
	})
}

func TestIdleAAPAndClawAlreadyIdled(t *testing.T) {
	ns := "test-ns"
	idler, clients := newTestIdler(t)

	aap := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "aap.ansible.com/v1alpha1",
		"kind":       "AnsibleAutomationPlatform",
		"metadata":   map[string]any{"name": "aap", "namespace": ns},
		"spec":       map[string]any{"idle_aap": true},
	}}
	aapGVR := schema.GroupVersionResource{Group: "aap.ansible.com", Version: "v1alpha1", Resource: "ansibleautomationplatforms"}
	_, err := clients.dynamicClient.Resource(aapGVR).Namespace(ns).Create(context.TODO(), aap, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, idler.IdleOwner(context.TODO(), &owners.ObjectWithGVR{Object: aap, GVR: &aapGVR}, Options{}))

	claw := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "claw.sandbox.redhat.com/v1alpha1",
		"kind":       "Claw",
		"metadata":   map[string]any{"name": "claw", "namespace": ns},
		"spec":       map[string]any{"idle": true},
	}}
	clawGVR := schema.GroupVersionResource{Group: "claw.sandbox.redhat.com", Version: "v1alpha1", Resource: "claws"}
	_, err = clients.dynamicClient.Resource(clawGVR).Namespace(ns).Create(context.TODO(), claw, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, idler.IdleOwner(context.TODO(), &owners.ObjectWithGVR{Object: claw, GVR: &clawGVR}, Options{}))
}

type testClients struct {
	dynamicClient *fakedynamic.FakeDynamicClient
	scalesClient  *fakescale.FakeScaleClient
}

func newTestIdler(t *testing.T) (*Idler, *testClients) {
	t.Helper()
	dynamicClient := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, customListKinds)
	scalesClient := &fakescale.FakeScaleClient{}
	restClient, err := testcommon.NewRESTClient("dummy-token", apiEndpoint)
	require.NoError(t, err)
	restClient.Client.Transport = gock.DefaultTransport
	t.Cleanup(func() {
		gock.OffAll()
	})
	fakeDiscovery := newFakeDiscoveryClient(allResourcesList(t)...)
	return New(fakeDiscovery, dynamicClient, scalesClient, restClient), &testClients{
		dynamicClient: dynamicClient,
		scalesClient:  scalesClient,
	}
}

func createAndGetOwner(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, obj runtime.Object, gvr schema.GroupVersionResource) *owners.ObjectWithGVR {
	t.Helper()
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	u := &unstructured.Unstructured{Object: unstructuredObj}
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		gvks, _, err := scheme.Scheme.ObjectKinds(obj)
		require.NoError(t, err)
		require.NotEmpty(t, gvks)
		gvk = gvks[0]
	}
	u.SetGroupVersionKind(gvk)
	created, err := dynamicClient.Resource(gvr).Namespace(u.GetNamespace()).Create(context.TODO(), u, metav1.CreateOptions{})
	require.NoError(t, err)
	return &owners.ObjectWithGVR{Object: created, GVR: &gvr}
}

func createTyped(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, obj runtime.Object) {
	t.Helper()
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	u := &unstructured.Unstructured{Object: unstructuredObj}
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	require.NoError(t, err)
	require.NotEmpty(t, gvks)
	u.SetGroupVersionKind(gvks[0])
	gvr, _ := meta.UnsafeGuessKindToResource(gvks[0])
	_, err = dynamicClient.Resource(gvr).Namespace(u.GetNamespace()).Create(context.TODO(), u, metav1.CreateOptions{})
	require.NoError(t, err)
}

func assertScaledToZero(t *testing.T, dynamicClient *fakedynamic.FakeDynamicClient, gvr schema.GroupVersionResource, namespace, name string) {
	t.Helper()
	got, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.NoError(t, err)
	replicasVal, found, err := unstructured.NestedInt64(got.Object, "spec", "replicas")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(0), replicasVal)
}

func assertScalePatched(t *testing.T, scalesClient *fakescale.FakeScaleClient, name, namespace string) {
	t.Helper()
	for _, action := range scalesClient.Actions() {
		if action.GetVerb() == "patch" {
			patchAction := action.(clienttest.PatchActionImpl)
			if patchAction.GetName() == name && patchAction.GetNamespace() == namespace {
				assert.JSONEq(t, `{"spec":{"replicas":0}}`, string(patchAction.GetPatch()))
				return
			}
		}
	}
	require.Failf(t, "expected scale patch", "for %s/%s", namespace, name)
}

type fakeDiscoveryClient struct {
	*fake.FakeDiscovery
}

func newFakeDiscoveryClient(resources ...*metav1.APIResourceList) *fakeDiscoveryClient {
	fakeDiscovery := fakeclientset.NewSimpleClientset().Discovery().(*fake.FakeDiscovery)
	fakeDiscovery.Resources = resources
	return &fakeDiscoveryClient{FakeDiscovery: fakeDiscovery}
}

func (c *fakeDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return c.Resources, nil
}

func allResourcesList(t *testing.T) []*metav1.APIResourceList {
	t.Helper()
	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "kubevirt.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "virtualmachineinstances", Namespaced: true, Kind: "VirtualMachineInstance"},
				{Name: "virtualmachines", Namespaced: true, Kind: "VirtualMachine"},
			},
		},
		{
			GroupVersion: "cdi.kubevirt.io/v1beta1",
			APIResources: []metav1.APIResource{
				{Name: "datavolumes", Namespaced: true, Kind: "DataVolume"},
			},
		},
		{
			GroupVersion: "aap.ansible.com/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "ansibleautomationplatforms", Namespaced: true, Kind: "AnsibleAutomationPlatform"},
			},
		},
		{
			GroupVersion: "serving.kserve.io/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "servingruntimes", Namespaced: true, Kind: "ServingRuntime"},
			},
		},
		{
			GroupVersion: "serving.kserve.io/v1beta1",
			APIResources: []metav1.APIResource{
				{Name: "inferenceservices", Namespaced: true, Kind: "InferenceService"},
			},
		},
		{
			GroupVersion: "claw.sandbox.redhat.com/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "claws", Namespaced: true, Kind: "Claw"},
			},
		},
		{
			GroupVersion: "apps.openshift.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "deploymentconfigs", Namespaced: true, Kind: "DeploymentConfig"},
			},
		},
	}
	for gvk := range scheme.Scheme.AllKnownTypes() {
		resource, _ := meta.UnsafeGuessKindToResource(gvk)
		resources = append(resources, &metav1.APIResourceList{
			GroupVersion: gvk.GroupVersion().String(),
			APIResources: []metav1.APIResource{
				{Name: resource.Resource, Namespaced: true, Kind: gvk.Kind},
			},
		})
	}
	for gvk, gvr := range SupportedScaleResources {
		resources = append(resources, &metav1.APIResourceList{
			GroupVersion: gvr.GroupVersion().String(),
			APIResources: []metav1.APIResource{
				{Name: gvr.Resource, Namespaced: true, Kind: gvk.Kind},
			},
		})
	}
	return resources
}
