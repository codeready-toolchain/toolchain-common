package configuration

import (
	"context"
	"sync"

	errs "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var configCache = &cache{}

var cacheLog = logf.Log.WithName("cache_toolchainconfig")

type cache struct {
	sync.RWMutex
	configObj runtime.Object
	secrets   map[string]map[string]string // map of secret key-value pairs indexed by secret name
}

func (c *cache) set(config runtime.Object, secrets map[string]map[string]string) {
	c.Lock()
	defer c.Unlock()
	c.configObj = config.DeepCopyObject()
	c.secrets = CopyOf(secrets)
}

func (c *cache) get() (runtime.Object, map[string]map[string]string) {
	c.RLock()
	defer c.RUnlock()
	if c.configObj == nil {
		return nil, CopyOf(c.secrets)
	}
	return c.configObj.DeepCopyObject(), CopyOf(c.secrets)
}

func updateConfig(config runtime.Object, secrets map[string]map[string]string) {
	configCache.set(config, secrets)
}

// loadLatest retrieves the latest configuration object and secrets using the provided client and updates the cache.
// If the resource is not found, then returns nil for the configuration and secret.
// If any failure happens while getting the configuration object or secrets, then returns an error.
func loadLatest(cl client.Client, configObj client.Object) (runtime.Object, map[string]map[string]string, error) {
	namespace, err := GetWatchNamespace()
	if err != nil {
		return nil, nil, errs.Wrap(err, "failed to get watch namespace")
	}

	if err := cl.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "config"}, configObj); err != nil {
		if apierrors.IsNotFound(err) {
			cacheLog.Info("ToolchainConfig resource with the name 'config' wasn't found, default configuration will be used", "namespace", namespace)
			return nil, nil, nil
		}
		return nil, nil, err
	}

	allSecrets, err := LoadSecrets(cl, namespace)
	if err != nil {
		return nil, nil, err
	}

	configCache.set(configObj, allSecrets)
	configCopy, secretsCopy := configCache.get()
	return configCopy, secretsCopy, nil
}

// getConfig returns a cached configuration object
// If no config is stored in the cache, then it retrieves it from the cluster using the provided LoadConfiguration func
// and stores in the cache.
// If the resource is not found, then returns nil for the configuration and secret.
// If any failure happens while getting the configuration object or secrets, then returns an error.
func getConfig(cl client.Client, configObj client.Object) (runtime.Object, map[string]map[string]string, error) {
	config, secrets := configCache.get()
	if config == nil {
		return loadLatest(cl, configObj)
	}
	return config, secrets, nil
}

// getCachedConfig returns the cached toolchainconfig or a toolchainconfig with default values
func getCachedConfig() (runtime.Object, map[string]map[string]string) {
	return configCache.get()
}

// Reset resets the cache.
// Should be used only in tests, but since it has to be used in other packages,
// then the function has to be exported and placed here.
func Reset() {
	configCache = &cache{}
}
