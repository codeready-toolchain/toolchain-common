package configuration

import (
	"context"
	"fmt"
	"os"
	"strings"

	errs "k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which is the namespace where the watch activity happens.
	// this value is empty if the operator is running with clusterScope.
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"

	// OperatorNameEnvVar is the constant for env variable OPERATOR_NAME
	// which is the name of the current operator
	OperatorNameEnvVar = "OPERATOR_NAME"
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
	namespace, err := GetWatchNamespace()
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

	for key, value := range secret.StringData {
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
	namespace, err := GetWatchNamespace()
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

// LoadSecrets lists all secrets in the provided namespace and indexes them into a map by name along with its secret data.
// Service account secrets are skipped.
func LoadSecrets(cl client.Client, namespace string) (map[string]map[string]string, error) {
	var allSecrets = make(map[string]map[string]string)
	secretList := &v1.SecretList{}
	err := cl.List(context.TODO(), secretList, client.InNamespace(namespace))
	if err != nil {
		return allSecrets, err
	}
	for _, secret := range secretList.Items {
		if _, ok := secret.Annotations["kubernetes.io/service-account.name"]; ok {
			// skip service account secrets
			continue
		}
		logf.Log.Info("found secret", "name", secret.Name)
		var secretData = make(map[string]string)
		for key, value := range secret.Data {
			secretData[key] = string(value)
			if key == "vm.access" {
				logf.Log.Info("data", "key", key, "value", string(value))
			}
		}
		for key, value := range secret.StringData {
			secretData[key] = string(value)
			if key == "vm.access" {
				logf.Log.Info("stringdata", "key", key)
				logf.Log.Info("data", "key", key, "value", string(value))
			}
		}
		allSecrets[secret.Name] = secretData
	}
	return allSecrets, err
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	if len(ns) == 0 {
		return "", fmt.Errorf("%s must not be empty", WatchNamespaceEnvVar)
	}
	return ns, nil
}

// GetOperatorName return the operator name
func GetOperatorName() (string, error) {
	operatorName, found := os.LookupEnv(OperatorNameEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", OperatorNameEnvVar)
	}
	if len(operatorName) == 0 {
		return "", fmt.Errorf("%s must not be empty", OperatorNameEnvVar)
	}
	return operatorName, nil
}
