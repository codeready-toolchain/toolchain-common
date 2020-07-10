package cluster

import (
	"context"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelType              = "type"
	labelNamespace         = "namespace"
	labelOwnerClusterName  = "ownerClusterName"
	labelCapacityExhausted = "toolchain.dev.openshift.com/capacity-exhausted"

	defaultHostOperatorNamespace   = "toolchain-host-operator"
	defaultMemberOperatorNamespace = "toolchain-member-operator"

	toolchainAPIQPS   = 20.0
	toolchainAPIBurst = 30
	toolchainTokenKey = "token"
)

// ToolchainClusterService manages cached cluster kube clients and related ToolchainCluster CRDs
// it's used for adding/updating/deleting
type ToolchainClusterService struct {
	client    client.Client
	log       logr.Logger
	namespace string
	timeout   time.Duration
}

// NewToolchainClusterService creates a new instance of ToolchainClusterService object and assigns the refreshCache function to the cache instance
func NewToolchainClusterService(client client.Client, log logr.Logger, namespace string, timeout time.Duration) ToolchainClusterService {
	service := ToolchainClusterService{
		client:    client,
		log:       log,
		namespace: namespace,
		timeout:   timeout,
	}
	clusterCache.refreshCache = service.refreshCache
	return service
}

// AddOrUpdateToolchainCluster takes the ToolchainCluster CR object,
// creates CachedToolchainCluster with a kube client and stores it in a cache
func (s *ToolchainClusterService) AddOrUpdateToolchainCluster(cluster *v1alpha1.ToolchainCluster) error {
	log := s.enrichLogger(cluster)
	log.Info("observed a cluster")

	err := s.addToolchainCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "the cluster was not added nor updated")
	}
	return nil
}

func (s *ToolchainClusterService) addToolchainCluster(toolchainCluster *v1alpha1.ToolchainCluster) error {
	// create the restclient of toolchainCluster
	clusterConfig, err := s.buildClusterConfig(toolchainCluster, toolchainCluster.Namespace)
	if err != nil {
		return errors.Wrap(err, "cannot create ToolchainCluster Config")
	}
	cl, err := client.New(clusterConfig, client.Options{})
	if err != nil {
		return errors.Wrap(err, "cannot create ToolchainCluster client")
	}

	capacityExhausted := false
	if c, exists := toolchainCluster.Labels[labelCapacityExhausted]; exists {
		capacityExhausted, err = strconv.ParseBool(c)
		if err != nil {
			return errors.Wrap(err, "cannot create ToolchainCluster client")
		}
	}
	cluster := &CachedToolchainCluster{
		Name:              toolchainCluster.Name,
		APIEndpoint:       toolchainCluster.Spec.APIEndpoint,
		Client:            cl,
		Config:            clusterConfig,
		ClusterStatus:     &toolchainCluster.Status,
		Type:              Type(toolchainCluster.Labels[labelType]),
		OperatorNamespace: toolchainCluster.Labels[labelNamespace],
		OwnerClusterName:  toolchainCluster.Labels[labelOwnerClusterName],
		CapacityExhausted: capacityExhausted,
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

	clusterCache.addCachedToolchainCluster(cluster)
	return nil
}

// DeleteToolchainCluster takes the ToolchainCluster CR object
// and deletes CachedToolchainCluster instance that has same name from a cache (if exists)
func (s *ToolchainClusterService) DeleteToolchainCluster(name string) {
	s.log.WithValues("Request.Name", name).Info("observed a deleted cluster")
	clusterCache.deleteCachedToolchainCluster(name)
}

func (s *ToolchainClusterService) refreshCache() {
	toolchainClusters := &v1alpha1.ToolchainClusterList{}
	if err := s.client.List(context.TODO(), toolchainClusters, &client.ListOptions{Namespace: s.namespace}); err != nil {
		s.log.Error(err, "the cluster cache was not refreshed")
	}
	for _, cluster := range toolchainClusters.Items {
		log := s.enrichLogger(&cluster)
		err := s.addToolchainCluster(&cluster)
		if err != nil {
			log.Error(err, "the cluster was not added", "cluster", cluster)
		}
	}
}

func (s *ToolchainClusterService) enrichLogger(cluster *v1alpha1.ToolchainCluster) logr.Logger {
	return s.log.
		WithValues("Request.Namespace", cluster.Namespace, "Request.Name", cluster.Name)
}

func (s *ToolchainClusterService) buildClusterConfig(toolchainCluster *v1alpha1.ToolchainCluster, toolchainNamespace string) (*rest.Config, error) {
	clusterName := toolchainCluster.Name

	apiEndpoint := toolchainCluster.Spec.APIEndpoint
	if apiEndpoint == "" {
		return nil, errors.Errorf("the api endpoint of cluster %s is empty", clusterName)
	}

	secretName := toolchainCluster.Spec.SecretRef.Name
	if secretName == "" {
		return nil, errors.Errorf("cluster %s does not have a secret name", clusterName)
	}
	secret := &v1.Secret{}
	name := types.NamespacedName{Namespace: toolchainNamespace, Name: secretName}
	err := s.client.Get(context.TODO(), name, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get secret %s for cluster %s", name, clusterName)
	}

	token, tokenFound := secret.Data[toolchainTokenKey]
	if !tokenFound || len(token) == 0 {
		return nil, errors.Errorf("the secret for cluster %s is missing a non-empty value for %q", clusterName, toolchainTokenKey)
	}

	clusterConfig, err := clientcmd.BuildConfigFromFlags(apiEndpoint, "")
	if err != nil {
		return nil, err
	}

	ca, err := base64.StdEncoding.DecodeString(toolchainCluster.Spec.CABundle)
	if err != nil {
		return nil, err
	}
	clusterConfig.CAData = ca
	clusterConfig.BearerToken = string(token)
	clusterConfig.QPS = toolchainAPIQPS
	clusterConfig.Burst = toolchainAPIBurst
	clusterConfig.Timeout = s.timeout

	return clusterConfig, nil
}

func IsReady(clusterStatus *v1alpha1.ToolchainClusterStatus) bool {
	for _, condition := range clusterStatus.Conditions {
		if condition.Type == v1alpha1.ToolchainClusterReady {
			if condition.Status == v1.ConditionTrue {
				return true
			}
		}
	}
	return false
}
