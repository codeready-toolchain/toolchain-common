// Code generated for package cluster by go-bindata DO NOT EDIT. (@generated)
// sources:
// deploy/crds/kubefed/core.kubefed.io_kubefedclusters.yaml
package cluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)
type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _coreKubefedIo_kubefedclustersYaml = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: (devel)
  creationTimestamp: null
  name: kubefedclusters.core.kubefed.io
spec:
  additionalPrinterColumns:
  - JSONPath: .metadata.creationTimestamp
    name: age
    type: date
  - JSONPath: .status.conditions[?(@.type=='Ready')].status
    name: ready
    type: string
  group: core.kubefed.io
  names:
    kind: KubeFedCluster
    listKind: KubeFedClusterList
    plural: kubefedclusters
    singular: kubefedcluster
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: KubeFedCluster configures KubeFed to be aware of a Kubernetes cluster
        and encapsulates the details necessary to communicate with the cluster.
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: KubeFedClusterSpec defines the desired state of KubeFedCluster
          properties:
            apiEndpoint:
              description: The API endpoint of the member cluster. This can be a hostname,
                hostname:port, IP or IP:port.
              type: string
            caBundle:
              description: CABundle contains the certificate authority information.
              format: byte
              type: string
            disabledTLSValidations:
              description: DisabledTLSValidations defines a list of checks to ignore
                when validating the TLS connection to the member cluster.  This can
                be any of *, SubjectName, or ValidityPeriod. If * is specified, it
                is expected to be the only option in list.
              items:
                type: string
              type: array
            secretRef:
              description: Name of the secret containing the token required to access
                the member cluster. The secret needs to exist in the same namespace
                as the control plane and should have a "token" key.
              properties:
                name:
                  description: Name of a secret within the enclosing namespace
                  type: string
              required:
              - name
              type: object
          required:
          - apiEndpoint
          - secretRef
          type: object
        status:
          description: KubeFedClusterStatus contains information about the current
            status of a cluster updated periodically by cluster controller.
          properties:
            conditions:
              description: Conditions is an array of current cluster conditions.
              items:
                description: ClusterCondition describes current state of a cluster.
                properties:
                  lastProbeTime:
                    description: Last time the condition was checked.
                    format: date-time
                    type: string
                  lastTransitionTime:
                    description: Last time the condition transit from one status to
                      another.
                    format: date-time
                    type: string
                  message:
                    description: Human readable message indicating details about last
                      transition.
                    type: string
                  reason:
                    description: (brief) reason for the condition's last transition.
                    type: string
                  status:
                    description: Status of the condition, one of True, False, Unknown.
                    type: string
                  type:
                    description: Type of cluster condition, Ready or Offline.
                    type: string
                required:
                - lastProbeTime
                - status
                - type
                type: object
              type: array
            region:
              description: Region is the name of the region in which all of the nodes
                in the cluster exist.  e.g. 'us-east1'.
              type: string
            zones:
              description: Zones are the names of availability zones in which the
                nodes of the cluster exist, e.g. 'us-east1-a'.
              items:
                type: string
              type: array
          required:
          - conditions
          type: object
      required:
      - spec
  version: v1beta1
  versions:
  - name: v1beta1
    served: true
    storage: true
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`)

func coreKubefedIo_kubefedclustersYamlBytes() ([]byte, error) {
	return _coreKubefedIo_kubefedclustersYaml, nil
}

func coreKubefedIo_kubefedclustersYaml() (*asset, error) {
	bytes, err := coreKubefedIo_kubefedclustersYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "core.kubefed.io_kubefedclusters.yaml", size: 5406, mode: os.FileMode(420), modTime: time.Unix(1593504704, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"core.kubefed.io_kubefedclusters.yaml": coreKubefedIo_kubefedclustersYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"core.kubefed.io_kubefedclusters.yaml": &bintree{coreKubefedIo_kubefedclustersYaml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
