package cluster

import (
	"context"
	errs "github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureKubeFedClusterCrd creates a KubeFedCluster CRD in the cluster.
// If the creation returns an error that is of the type "AlreadyExists" then the error is ignored,
// if the error is of another type then it is returned
func EnsureKubeFedClusterCrd(scheme *runtime.Scheme, client client.Client) error {
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	kubeFedCrd := &v1beta1.CustomResourceDefinition{}
	_, _, err := decoder.Decode([]byte(kubeFedClusterCrd), nil, kubeFedCrd)
	if err != nil {
		return errs.Wrap(err, "unable to decode the KubeFedCluster CRD")
	}
	err = client.Create(context.TODO(), kubeFedCrd)
	if err != nil && !errors.IsAlreadyExists(err) {
		return errs.Wrap(err, "unable to create the KubeFedCluster CRD")
	}
	return nil
}
