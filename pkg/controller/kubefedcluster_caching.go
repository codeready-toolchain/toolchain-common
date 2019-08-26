package controller

import (
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func StartCachingController(mgr manager.Manager, namespace string, stopChan <-chan struct{}) error {
	cntrlName := "controller_kubefedcluster_with_cache"
	clusterCacheService := cluster.KubeFedClusterService{
		Client: mgr.GetClient(),
		Log:    logf.Log.WithName(cntrlName),
	}

	_, clusterController, err := util.NewGenericInformerWithEventHandler(
		mgr.GetConfig(),
		namespace,
		&v1beta1.KubeFedCluster{},
		util.NoResyncPeriod,
		&cache.ResourceEventHandlerFuncs{
			DeleteFunc: clusterCacheService.DeleteKubeFedCluster,
			AddFunc:    clusterCacheService.AddKubeFedCluster,
			UpdateFunc: clusterCacheService.UpdateKubeFedCluster,
		},
	)
	if err != nil {
		return err
	}
	logf.Log.Info("Starting Controller", "controller", cntrlName)
	go clusterController.Run(stopChan)
	return nil
}
