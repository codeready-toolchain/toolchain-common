package client_test

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSsaClient(t *testing.T) {
	t.Run("ApplyObject", func(t *testing.T) {
		t.Run("creates", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}

			// when
			require.NoError(t, acl.ApplyObject(context.TODO(), obj))

			// then
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))
		})
		t.Run("updates", func(t *testing.T) {
			// given
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
				Data: map[string]string{"a": "b"},
			}
			cl, acl := NewTestSsaApplyClient(t, obj)

			updated := obj.DeepCopy()
			updated.Data["a"] = "c"

			// when
			require.NoError(t, acl.ApplyObject(context.TODO(), updated))
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

			// then
			assert.Equal(t, "c", inCluster.Data["a"])
		})
		t.Run("SetOwnerReference", func(t *testing.T) {
			// given
			owner := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner",
					Namespace: "default",
				},
			}

			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owned",
					Namespace: "default",
				},
			}
			cl, acl := NewTestSsaApplyClient(t, owner, obj)

			// when
			require.NoError(t, acl.ApplyObject(context.TODO(), obj, client.SetOwnerReference(owner)))
			inCluster := &corev1.ConfigMap{}
			require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

			// then
			require.Len(t, inCluster.OwnerReferences, 1)
			assert.Equal(t, "ConfigMap", inCluster.OwnerReferences[0].Kind)
			assert.Equal(t, "owner", inCluster.OwnerReferences[0].Name)
		})
		t.Run("EnsureLabels", func(t *testing.T) {
			t.Run("merge with existing", func(t *testing.T) {
				// given
				obj := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "obj",
						Namespace: "default",
						Labels:    map[string]string{"k": "l", "m": "n"},
					},
				}
				cl, acl := NewTestSsaApplyClient(t, obj)

				// when
				require.NoError(t, acl.ApplyObject(context.TODO(), obj, client.EnsureLabels(map[string]string{"a": "b", "c": "d"})))
				inCluster := &corev1.ConfigMap{}
				require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

				// then
				require.NotNil(t, inCluster.Labels)
				require.Len(t, inCluster.Labels, 4)
				assert.Equal(t, "b", inCluster.Labels["a"])
				assert.Equal(t, "d", inCluster.Labels["c"])
				assert.Equal(t, "l", inCluster.Labels["k"])
				assert.Equal(t, "n", inCluster.Labels["m"])
			})
			t.Run("add to empty", func(t *testing.T) {
				// given
				obj := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "obj",
						Namespace: "default",
					},
				}
				cl, acl := NewTestSsaApplyClient(t, obj)

				// when
				require.NoError(t, acl.ApplyObject(context.TODO(), obj, client.EnsureLabels(map[string]string{"a": "b", "c": "d"})))
				inCluster := &corev1.ConfigMap{}
				require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))

				// then
				require.NotNil(t, inCluster.Labels)
				require.Len(t, inCluster.Labels, 2)
				assert.Equal(t, "b", inCluster.Labels["a"])
				assert.Equal(t, "d", inCluster.Labels["c"])
			})
		})
		t.Run("SkipIf", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}

			// when
			require.NoError(t, acl.ApplyObject(context.TODO(), obj, client.SkipIf(func(o runtimeclient.Object) bool {
				return true
			})))

			// then
			inCluster := &corev1.ConfigMap{}
			require.True(t, errors.IsNotFound(cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster)))
		})
		t.Run("MigrateSSA", func(t *testing.T) {
			for _, setup := range []struct {
				defaultMigrate    bool
				explicitMigrate   *bool
				migrationExpected bool
			}{
				{
					defaultMigrate:    false,
					explicitMigrate:   ptr.To(true),
					migrationExpected: true,
				},
				{
					defaultMigrate:    false,
					explicitMigrate:   ptr.To(false),
					migrationExpected: false,
				},
				{
					defaultMigrate:    false,
					explicitMigrate:   nil,
					migrationExpected: false,
				},
				{
					defaultMigrate:    true,
					explicitMigrate:   ptr.To(true),
					migrationExpected: true,
				},
				{
					defaultMigrate:    true,
					explicitMigrate:   ptr.To(false),
					migrationExpected: false,
				},
				{
					defaultMigrate:    true,
					explicitMigrate:   nil,
					migrationExpected: true,
				},
			} {
				testName := fmt.Sprintf("default: %v, explicit: %v", setup.defaultMigrate, setup.explicitMigrate)
				t.Run(testName, func(t *testing.T) {
					// given
					obj := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "obj",
							Namespace: "default",
							ManagedFields: []metav1.ManagedFieldsEntry{
								{
									FieldsType: "FieldsV1",
									FieldsV1:   &metav1.FieldsV1{Raw: []byte(`{"f:spec": {"f:selector": {}}}`)},
									Manager:    strings.Split(rest.DefaultKubernetesUserAgent(), "/")[0],
									Operation:  metav1.ManagedFieldsOperationUpdate,
								},
							},
						},
						Spec: corev1.ServiceSpec{},
					}
					toApply := obj.DeepCopy()
					toApply.SetManagedFields(nil)

					cl, acl := NewTestSsaApplyClient(t, obj)
					acl.MigrateSSAByDefault = setup.defaultMigrate

					// when
					var opts []client.SSAApplyObjectOption
					if setup.explicitMigrate != nil {
						opts = append(opts, client.MigrateSSA(*setup.explicitMigrate))
					}
					inCluster := &corev1.Service{}
					require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))
					require.NoError(t, acl.ApplyObject(context.TODO(), toApply, opts...))

					// then
					inCluster = &corev1.Service{}
					require.NoError(t, cl.Get(context.TODO(), runtimeclient.ObjectKeyFromObject(obj), inCluster))
					if setup.migrationExpected {
						assert.Len(t, inCluster.ManagedFields, 1)
						assert.Equal(t, "test-field-owner", inCluster.ManagedFields[0].Manager)
						assert.Equal(t, metav1.ManagedFieldsOperationApply, inCluster.ManagedFields[0].Operation)
					} else {
						assert.Len(t, inCluster.ManagedFields, 1)
						assert.NotEqual(t, "test-field-owner", inCluster.ManagedFields[0].Manager)
						assert.Equal(t, metav1.ManagedFieldsOperationUpdate, inCluster.ManagedFields[0].Operation)
					}
				})
			}
		})
		t.Run("propagates k8s errors", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			cl.MockPatch = func(ctx context.Context, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error {
				return errors.NewForbidden(schema.GroupResource{}, "asdf", stderrors.New("fabricated"))
			}
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj",
					Namespace: "default",
				},
			}

			// when
			err := acl.ApplyObject(context.TODO(), obj)

			// then
			assert.True(t, errors.IsForbidden(err))
		})
		t.Run("error message format", func(t *testing.T) {
			t.Run("on option application error", func(t *testing.T) {
				// given
				_, acl := NewTestSsaApplyClient(t)
				obj := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "obj",
						Namespace: "default",
					},
				}
				invalidOwner := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "obj",
						Namespace: "differentNamespace",
					},
				}

				// when
				err := acl.ApplyObject(context.TODO(), obj, client.SetOwnerReference(invalidOwner))

				// then
				require.Error(t, err)
				assert.Equal(t, "unable to patch '*v1.ConfigMap' called 'obj' in namespace 'default': failed to configure the apply function: cross-namespace owner references are disallowed, owner's namespace differentNamespace, obj's namespace default", err.Error())
			})
			t.Run("on SSA prep error", func(t *testing.T) {
				// given
				cl := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
				acl := client.NewSSAApplyClient(cl, "testOwner")

				obj := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "obj",
						Namespace: "default",
					},
				}

				// when
				err := acl.ApplyObject(context.TODO(), obj)

				// then
				require.Error(t, err)
				assert.Equal(t, "unable to patch '*v1.ConfigMap' called 'obj' in namespace 'default': failed to prepare the object for SSA: no kind is registered for the type v1.ConfigMap in scheme \"pkg/runtime/scheme.go:100\"", err.Error())
			})
			t.Run("on k8s error", func(t *testing.T) {
				// given
				cl, acl := NewTestSsaApplyClient(t)
				cl.MockPatch = func(ctx context.Context, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error {
					return errors.NewForbidden(schema.GroupResource{}, "asdf", stderrors.New("fabricated"))
				}
				obj := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "obj",
						Namespace: "default",
					},
				}

				// when
				err := acl.ApplyObject(context.TODO(), obj)

				// then
				assert.Equal(t, "unable to patch '/v1, Kind=ConfigMap' called 'obj' in namespace 'default': forbidden: fabricated", err.Error())
			})
		})
	})
	t.Run("Apply", func(t *testing.T) {
		t.Run("executes a for loop", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			obj1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj1",
					Namespace: "default",
				},
			}
			obj2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj2",
					Namespace: "default",
				},
			}

			// when
			require.NoError(t, acl.Apply(context.TODO(), []runtimeclient.Object{obj1, obj2}))

			// then
			inCluster := &corev1.ConfigMapList{}
			require.NoError(t, cl.List(context.TODO(), inCluster))
			assert.Len(t, inCluster.Items, 2)
		})
		t.Run("exits early", func(t *testing.T) {
			// given
			cl, acl := NewTestSsaApplyClient(t)
			obj1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj1",
					Namespace: "default",
				},
			}
			obj2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "obj2",
					Namespace: "default",
				},
			}
			counter := 0
			cl.MockPatch = func(ctx context.Context, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error {
				if counter == 0 {
					counter += 1
					return test.Patch(ctx, cl, obj, patch, opts...)
				}
				return fmt.Errorf("boom")
			}

			// when
			err := acl.Apply(context.TODO(), []runtimeclient.Object{obj1, obj2})

			// then
			require.Error(t, err)
			inCluster := &corev1.ConfigMapList{}
			require.NoError(t, cl.List(context.TODO(), inCluster))
			assert.Len(t, inCluster.Items, 1)
		})
	})
}

func TestEnsureGVK(t *testing.T) {
	emptyScheme := runtime.NewScheme()

	t.Run("scheme not consulted when GVK present", func(t *testing.T) {
		// given
		withGvk := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
			},
		}

		// when
		err := client.EnsureGVK(withGvk, emptyScheme)

		// then
		require.NoError(t, err)
	})

	t.Run("scheme consulted when no GVK present", func(t *testing.T) {
		withoutGvk := &corev1.ConfigMap{}

		// when
		err := client.EnsureGVK(withoutGvk, emptyScheme)

		// then
		require.Error(t, err)
	})
}

func NewTestSsaApplyClient(t *testing.T, initObjs ...runtimeclient.Object) (*test.FakeClient, *client.SSAApplyClient) {
	cl := test.NewFakeClient(t, initObjs...)

	return cl, &client.SSAApplyClient{
		Client:     cl,
		FieldOwner: "test-field-owner",
	}
}
