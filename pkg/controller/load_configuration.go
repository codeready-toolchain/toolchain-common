package controller

import (
	"context"
	"os"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// LoadFromSecret retrieves an operator secret
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
		return err
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

// LoadFromConfigMap retrieves the host operator configMap
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
		return err
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
		logf.Log.Info(key + " is not set")
		return ""
	}

	return resourceName
}

// createHostEnvVarKey creates env vars based on resource data
func createOperatorEnvVarKey(prefix, key string) string {
	return prefix + "_" + (strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), "-", "_")))
}
