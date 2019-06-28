package cluster

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

const (
	labelType             = "type"
	labelNamespace        = "namespace"
	labelOwnerClusterName = "ownerClusterName"

	defaultHostOperatorNamespace   = "toolchain-host-operator"
	defaultMemberOperatorNamespace = "toolchain-member-operator"
)

// KubeFedClusterService manages cached cluster kube clients and related KubeFedCluster CRDs
// it's used for adding/updating/deleting
type KubeFedClusterService struct {
	Client client.Client
	Log    logr.Logger
}

// AddKubeFedCluster takes the KubeFedCluster CR object,
// creates FedCluster with a kube client and stores it in a cache
func (s *KubeFedClusterService) AddKubeFedCluster(obj interface{}) {
	cluster, err := castToKubeFedCluster(obj)
	if err != nil {
		s.Log.Error(err, "cluster not added")
		return
	}
	log := s.enrichLogger(cluster)
	log.Info("observed a new cluster")

	err = s.addKubeFedCluster(cluster)
	if err != nil {
		log.Error(err, "the new cluster was not added")
	}
}

func (s *KubeFedClusterService) addKubeFedCluster(fedCluster *v1beta1.KubeFedCluster) error {
	// create the restclient of fedCluster
	clusterConfig, err := s.buildClusterConfig(fedCluster, fedCluster.Namespace)
	if err != nil {
		return errors.Wrap(err, "cannot create KubeFedCluster config")
	}
	cl, err := client.New(clusterConfig, client.Options{})
	if err != nil {
		return errors.Wrap(err, "cannot create KubeFedCluster client")
	}

	cluster := &FedCluster{
		Name:              fedCluster.Name,
		Client:            cl,
		ClusterStatus:     &fedCluster.Status,
		Type:              Type(fedCluster.Labels[labelType]),
		OperatorNamespace: fedCluster.Labels[labelNamespace],
		OwnerClusterName:  fedCluster.Labels[labelOwnerClusterName],
	}
	if cluster.Type == "" {
		cluster.Type = Member
	}
	if cluster.OperatorNamespace == "" {
		if cluster.Type == Host {
			cluster.OperatorNamespace = defaultHostOperatorNamespace
		} else {
			cluster.OperatorNamespace = defaultMemberOperatorNamespace
		}
	}

	clusterCache.addFedCluster(cluster)
	return nil
}

// DeleteKubeFedCluster takes the KubeFedCluster CR object
// and deletes FedCluster instance that has same name from a cache (if exists)
func (s *KubeFedClusterService) DeleteKubeFedCluster(obj interface{}) {
	cluster, err := castToKubeFedCluster(obj)
	if err != nil {
		s.Log.Error(err, "cluster not deleted")
		return
	}
	log := s.enrichLogger(cluster)
	log.Info("observed a deleted cluster")
	clusterCache.deleteFedCluster(cluster.Name)
}

// UpdateKubeFedCluster takes the KubeFedCluster CR object,
// creates FedCluster with a kube client and stores it in a cache.
// If there cache already contains such an instance, then it is overridden.
func (s *KubeFedClusterService) UpdateKubeFedCluster(_, newObj interface{}) {
	newCluster, err := castToKubeFedCluster(newObj)
	if err != nil {
		s.Log.Error(err, "cluster not updated")
		return
	}
	log := s.enrichLogger(newCluster)
	log.Info("observed an updated cluster")

	err = s.addKubeFedCluster(newCluster)
	if err != nil {
		log.Error(err, "the cluster was not updated")
	}
}

func (s *KubeFedClusterService) enrichLogger(cluster *v1beta1.KubeFedCluster) logr.Logger {
	return s.Log.
		WithValues("Request.Namespace", cluster.Namespace, "Request.Name", cluster.Name)
}

func (s *KubeFedClusterService) buildClusterConfig(fedCluster *v1beta1.KubeFedCluster, fedNamespace string) (*rest.Config, error) {
	clusterName := fedCluster.Name

	apiEndpoint := fedCluster.Spec.APIEndpoint
	if apiEndpoint == "" {
		return nil, errors.Errorf("the api endpoint of cluster %s is empty", clusterName)
	}

	secretName := fedCluster.Spec.SecretRef.Name
	if secretName == "" {
		return nil, errors.Errorf("cluster %s does not have a secret name", clusterName)
	}
	secret := &v1.Secret{}
	name := types.NamespacedName{Namespace: fedNamespace, Name: secretName}
	err := s.Client.Get(context.TODO(), name, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get secret %s for cluster %s", name, clusterName)
	}

	token, tokenFound := secret.Data[util.TokenKey]
	if !tokenFound || len(token) == 0 {
		return nil, errors.Errorf("the secret for cluster %s is missing a non-empty value for %q", clusterName, util.TokenKey)
	}

	clusterConfig, err := clientcmd.BuildConfigFromFlags(apiEndpoint, "")
	if err != nil {
		return nil, err
	}
	clusterConfig.CAData = fedCluster.Spec.CABundle
	clusterConfig.BearerToken = string(token)
	clusterConfig.QPS = util.KubeAPIQPS
	clusterConfig.Burst = util.KubeAPIBurst

	return clusterConfig, nil
}

func castToKubeFedCluster(obj interface{}) (*v1beta1.KubeFedCluster, error) {
	cluster, ok := obj.(*v1beta1.KubeFedCluster)
	if !ok {
		return nil, fmt.Errorf("incorrect type of KubeFedCluster: %+v", obj)
	}
	return cluster, nil
}
