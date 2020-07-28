package configuration

import (
	"context"
	"os"
	"strings"

	errs "k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// LoadFromSecret retrieves an operator secret and sets environment
// variables in order to override default configurations.
// If no secret is found, then configuration will use defaults.
// Returns error if WATCH_NAMESPACE is not set, if the resource GET request failed
// (for other reasons apart from isNotFound) and if setting env vars fails.
//
// prefix: represents the operator prefix (HOST_OPERATOR/MEMBER_OPERATOR)
// resourceKey: is the env var which contains the secret resource name.
// cl: is the client that should be used to retrieve the configmap.
func LoadFromSecret(prefix, resourceKey string, cl client.Client) error {
	// get the secret name
	secretName := getResourceName(resourceKey)
	if secretName == "" {
		return nil
	}

	// get the secret
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	secret := &v1.Secret{}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: secretName}
	err = cl.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		if !errs.IsNotFound(err) {
			return err
		}
		logf.Log.Info("secret is not found")
	}

	// get secrets and set environment variables
	for key, value := range secret.Data {
		secretKey := createOperatorEnvVarKey(prefix, key)
		err := os.Setenv(secretKey, string(value))
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadFromConfigMap retrieves the host operator configmap and sets environment
// variables in order to override default configurations.
// If no configmap is found, then configuration will use all defaults.
// Returns error if WATCH_NAMESPACE is not set, if the resource GET request failed
// (for other reasons apart from isNotFound) and if setting env vars fails.
//
// prefix: represents the operator prefix (HOST_OPERATOR/MEMBER_OPERATOR)
// resourceKey: is the env var which contains the configmap resource name.
// cl: is the client that should be used to retrieve the configmap.
func LoadFromConfigMap(prefix, resourceKey string, cl client.Client) error {
	// get the configMap name
	configMapName := getResourceName(resourceKey)
	if configMapName == "" {
		return nil
	}

	// get the configMap
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	configMap := &v1.ConfigMap{}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: configMapName}
	err = cl.Get(context.TODO(), namespacedName, configMap)
	if err != nil {
		if !errs.IsNotFound(err) {
			return err
		}
		logf.Log.Info("configmap is not found")
	}

	// get configMap data and set environment variables
	for key, value := range configMap.Data {
		configKey := createOperatorEnvVarKey(prefix, key)
		err := os.Setenv(configKey, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// getResourceName gets the resource name via env var
func getResourceName(key string) string {
	// get the resource name
	resourceName := os.Getenv(key)
	if resourceName == "" {
		logf.Log.Info(key + " is not set. Will not override default configurations")
		return ""
	}

	return resourceName
}

// createHostEnvVarKey creates env vars based on resource data
func createOperatorEnvVarKey(prefix, key string) string {
	return prefix + "_" + (strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), "-", "_")))
}
