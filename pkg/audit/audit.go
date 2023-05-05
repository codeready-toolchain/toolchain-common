package audit

import (
	"encoding/json"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceChangeType string

const (
	ResourceCreated       ResourceChangeType = "created"
	ResourceUpdated       ResourceChangeType = "updated"
	ResourcePatched       ResourceChangeType = "patched"
	ResourceDeleted       ResourceChangeType = "deleted"
	ResourceStatusUpdated ResourceChangeType = "status_updated"
	ResourceStatusPatched ResourceChangeType = "status_patched"
)

func LogAPIResourceChangeEvent(logger logr.Logger, resource runtimeclient.Object, resourceChangeType ResourceChangeType) {
	logger = logger.WithValues("audit", "true")

	if resource == nil {
		logger.Error(nil, "resource passed to LogAPIResourceChangeEvent was nil")
		return
	}
	secret, isSecret := (resource).(*corev1.Secret)
	if isSecret {
		// Make a copy of the resource, before we modify it
		secret = secret.DeepCopy()
		// Remove the data field, as this contains data that should not be logged.
		secret.Data = map[string][]byte{}
		resource = secret
	}
	jsonRepresentation, err := json.Marshal(resource)

	if err != nil {
		logger.Error(err, "unable to marshall object",
			"namespace", resource.GetNamespace(),
			"name", resource.GetName())
	}

	logger.Info("resource changed",
		"type", string(resourceChangeType),
		"namespace", resource.GetNamespace(),
		"name", resource.GetName(),
		"object", string(jsonRepresentation))
}
