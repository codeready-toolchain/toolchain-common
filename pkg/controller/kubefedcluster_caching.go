package controller

import (
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func StartCachingController(mgr manager.Manager, namespace string, stopChan <-chan struct{}) error {
	cntrlName := "controller_kubefedcluster_with_cache"

	clusterCacheService := cluster.NewKubeFedClusterService(mgr.GetClient(), logf.Log.WithName(cntrlName), namespace)
	eventHandler := &cache.ResourceEventHandlerFuncs{
		DeleteFunc: clusterCacheService.DeleteKubeFedCluster,
		AddFunc:    clusterCacheService.AddKubeFedCluster,
		UpdateFunc: clusterCacheService.UpdateKubeFedCluster,
	}

	gvk, err := apiutil.GVKForObject(&v1beta1.KubeFedCluster{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	client, err := apiutil.RESTClientForGVK(gvk, mgr.GetConfig(), scheme.Codecs)
	if err != nil {
		return err
	}

	listWatch := cache.NewListWatchFromClient(client, "kubefedclusters", namespace, fields.Everything())

	_, clusterController := cache.NewInformer(listWatch, &v1beta1.KubeFedCluster{}, util.NoResyncPeriod, eventHandler)

	logf.Log.Info("Starting Controller", "controller", cntrlName)
	go clusterController.Run(stopChan)
	return nil
}
