package finalizers

import (
	"context"
	"errors"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/finalizer"
)

// Finalizers is a simple wrapper around finalizer.Finalizers that can also perform the actual update
// of the resource based on the finalization result.
type Finalizers struct {
	// finalizers is the implementation of the finalizer control logic implemented in the controller runtime.
	finalizers finalizer.Finalizers
}

// FinalizerFunc is a functional implementation of the finalizer.Finalizer interface.
type FinalizerFunc func(context.Context, client.Object) (finalizer.Result, error)

// RegisterWithStandardName registers the provided finalizer with the standard finalizer name defined in
// toolchainv1alpha1.
func (fs *Finalizers) RegisterWithStandardName(f finalizer.Finalizer) error {
	return fs.Register(toolchainv1alpha1.FinalizerName, f)
}

// Register registers the finalizer so that FinalizeAndUpdate will call it. Note that the finalizer MUST
// return an error if the condition for removing it from the object is not satisfied.
func (fs *Finalizers) Register(key string, f finalizer.Finalizer) error {
	fs.ensureInitialized()
	return fs.finalizers.Register(key, f)
}

// FinalizeAndUpdate runs the registered finalizers on the object and reports true if the object or its status
// has been updated in the cluster using the provided client.
//
// The result of calling this method on an object that is not being deleted is that all the registered finalizers
// are added to the set of the finalizers on the object and the object is updated in the cluster (and therefore true
// is returned if the finalizers were added).
//
// The result of calling this method on an object that is being deleted is that the all the registered finalizers are
// called and the finalizers are removed if they succeed (i.e. they don't return an error).
//
// Note also, that, given the logic described above, there is no need to check for the object's deletion timestamp during
// the reconciliation. Returning early from the reconciler when this method returns true (or an error) is the correct
// thing to do in all cases.
func (f *Finalizers) FinalizeAndUpdate(ctx context.Context, cl client.Client, obj client.Object) (bool, error) {
	f.ensureInitialized()

	res, err := f.finalizers.Finalize(ctx, obj)

	var errs []error

	if err != nil {
		errs = append(errs, err)
	}

	if res.Updated {
		if err := cl.Update(ctx, obj); err != nil {
			errs = append(errs, err)
		}
	}
	if res.StatusUpdated {
		if err := cl.Status().Update(ctx, obj); err != nil {
			errs = append(errs, err)
		}
	}

	return res.Updated || res.StatusUpdated, errors.Join(errs...)
}

func (f *Finalizers) ensureInitialized() {
	if f.finalizers == nil {
		f.finalizers = finalizer.NewFinalizers()
	}
}

func (f FinalizerFunc) Finalize(ctx context.Context, obj client.Object) (finalizer.Result, error) {
	return f(ctx, obj)
}

var _ finalizer.Finalizer = (FinalizerFunc)(nil)
