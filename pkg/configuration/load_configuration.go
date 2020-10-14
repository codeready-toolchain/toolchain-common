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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// LoadFromSecret retrieves an operator secret, loads all keys and values from the secret
// and stores them in a map. This map is then returned by the function.
// The function doesn't take into account any default values - this has to be
// handled while getting the values in the configuration object.
//
// resourceKey: is the env var which contains the secret resource name.
// cl: is the client that should be used to retrieve the secret.
func LoadFromSecret(resourceKey string, cl client.Client) (map[string]string, error) {
	var secretData = make(map[string]string)

	// get the secret name
	secretName := getResourceName(resourceKey)
	if secretName == "" {
		return secretData, nil
	}

	// get the secret
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return secretData, err
	}

	secret := &v1.Secret{}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: secretName}
	err = cl.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		if !errs.IsNotFound(err) {
			return secretData, err
		}
		logf.Log.Info("secret is not found")
	}

	for key, value := range secret.Data {
		secretData[key] = string(value)
	}

	return secretData, nil
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

// createOperatorEnvVarKey creates env vars based on resource data.
// Returns env var key.
//
// prefix: represents the operator prefix (HOST_OPERATOR/MEMBER_OPERATOR)
// key: is the value to convert into an env var key
func createOperatorEnvVarKey(prefix, key string) string {
	return prefix + "_" + (strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), "-", "_")))
}
