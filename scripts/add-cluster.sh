#!/usr/bin/env bash

user_help () {
    echo "Creates ToolchainCluster"
    echo "options:"
    echo "-t, --type            joining cluster type (host or member)"
    echo "-tn, --type-name      the type name of the joining cluster (host, member or e2e)"
    echo "-mn, --member-ns      namespace where member-operator is running"
    echo "-hn, --host-ns        namespace where host-operator is running"
    echo "-s,  --single-cluster running both operators on single cluster"
    echo "-kc,  --kube-config   kubeconfig for managing multiple clusters"
    exit 0
}

login_to_cluster() {
    if [[ ${SINGLE_CLUSTER} != "true" ]]; then
      if [[ -z ${KUBECONFIG} ]]; then
        echo "Please specify the path to kube config file using the parameter --kube-config"
      else
        oc config use-context "$1-admin"
      fi
    fi
}

create_service_account() {
cat <<EOF | oc apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
rules:
- apiGroups:
  - toolchain.dev.openshift.com
  resources:
  - "*"
  verbs:
  - "*"
- apiGroups:
  - route.openshift.io
  resources:
  - routes
  verbs:
  - "get"
  - "list"
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${SA_NAME}
rules:
- apiGroups:
  - route.openshift.io
  resources:
  - routes
  verbs:
  - "get"
  - "list"
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
subjects:
- kind: ServiceAccount
  name: ${SA_NAME}
roleRef:
  kind: Role
  name: ${SA_NAME}
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${SA_NAME}
subjects:
- kind: ServiceAccount
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
roleRef:
  kind: ClusterRole
  name: ${SA_NAME}
  apiGroup: rbac.authorization.k8s.io
EOF
}

create_service_account_e2e() {
ROLE_NAME=`oc get Roles -n ${OPERATOR_NS} -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep "^toolchain-${JOINING_CLUSTER_TYPE}-operator\.v"`
if [[ -z ${ROLE_NAME} ]]; then
    echo "Role that would have a prefix 'toolchain-${JOINING_CLUSTER_TYPE}-operator.v' wasn't found - available roles are:"
    echo `oc get Roles -n ${OPERATOR_NS}`
    exit 1
fi
echo "using Role ${ROLE_NAME}"
CLUSTER_ROLE_NAME=`oc get ClusterRoles -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep "^toolchain-${JOINING_CLUSTER_TYPE}-operator\.v"`
if [[ -z ${CLUSTER_ROLE_NAME} ]]; then
    echo "ClusterRole that would have a prefix 'toolchain-${JOINING_CLUSTER_TYPE}-operator.v' wasn't found - available ClusterRoles are:"
    echo `oc get ClusterRoles`
    exit 1
fi
echo "using ClusterRole ${CLUSTER_ROLE_NAME}"
cat <<EOF | oc apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
subjects:
- kind: ServiceAccount
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
roleRef:
  kind: Role
  name: ${ROLE_NAME}
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${SA_NAME}-${OPERATOR_NS}
subjects:
- kind: ServiceAccount
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
roleRef:
  kind: ClusterRole
  name: ${CLUSTER_ROLE_NAME}
  apiGroup: rbac.authorization.k8s.io
EOF
}

if [[ $# -lt 2 ]]
then
    user_help
fi

while test $# -gt 0; do
       case "$1" in
            -h|--help)
                user_help
                ;;
            -t|--type)
                shift
                JOINING_CLUSTER_TYPE=$1
                shift
                ;;
            -tn|--type-name)
                shift
                JOINING_CLUSTER_TYPE_NAME=$1
                shift
                ;;
            -mn|--member-ns)
                shift
                MEMBER_OPERATOR_NS=$1
                shift
                ;;
            -hn|--host-ns)
                shift
                HOST_OPERATOR_NS=$1
                shift
                ;;
            -kc|--kube-config)
                shift
                KUBECONFIG=$1
                shift
                ;;
            -s|--single-cluster)
                SINGLE_CLUSTER=true
                shift
                ;;
            *)
               echo "$1 is not a recognized flag!"
               user_help
               exit -1
               ;;
      esac
done

CLUSTER_JOIN_TO="host"

# We need this to configurable to work with dynamic namespaces from end to end tests
OPERATOR_NS=${MEMBER_OPERATOR_NS}
CLUSTER_JOIN_TO_OPERATOR_NS=${HOST_OPERATOR_NS}
if [[ ${JOINING_CLUSTER_TYPE} == "host" ]]; then
  CLUSTER_JOIN_TO="member"
  OPERATOR_NS=${HOST_OPERATOR_NS}
  CLUSTER_JOIN_TO_OPERATOR_NS=${MEMBER_OPERATOR_NS}
fi
JOINING_CLUSTER_TYPE_NAME=${JOINING_CLUSTER_TYPE_NAME:-${JOINING_CLUSTER_TYPE}}

# This is using default values i.e. toolchain-member-operator or toolchain-host-operator for local setup
if [[ ${OPERATOR_NS} == "" &&  ${CLUSTER_JOIN_TO_OPERATOR_NS} == "" ]]; then
  OPERATOR_NS=toolchain-${JOINING_CLUSTER_TYPE}-operator
  CLUSTER_JOIN_TO_OPERATOR_NS=toolchain-${CLUSTER_JOIN_TO}-operator
fi

echo ${OPERATOR_NS}
echo ${CLUSTER_JOIN_TO_OPERATOR_NS}

login_to_cluster ${JOINING_CLUSTER_TYPE}

if [[ ${JOINING_CLUSTER_TYPE_NAME} != "e2e" ]]; then
    SA_NAME="toolchaincluster-${JOINING_CLUSTER_TYPE}-operator"
    create_service_account
else
    SA_NAME="e2e-service-account"
    create_service_account_e2e
fi

echo "Getting ${JOINING_CLUSTER_TYPE} SA token"
SA_SECRET=`oc get sa ${SA_NAME} -n ${OPERATOR_NS} -o json | jq -r .secrets[].name | grep token`
SA_TOKEN=`oc get secret ${SA_SECRET} -n ${OPERATOR_NS}  -o json | jq -r '.data["token"]' | base64 --decode`
SA_CA_CRT=`oc get secret ${SA_SECRET} -n ${OPERATOR_NS} -o json | jq -r '.data["ca.crt"]'`

API_ENDPOINT=`oc get infrastructure cluster -o jsonpath='{.status.apiServerURL}'`
JOINING_CLUSTER_NAME=`oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'`

login_to_cluster ${CLUSTER_JOIN_TO}

CLUSTER_JOIN_TO_NAME=`oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'`

SECRET_NAME=${SA_NAME}-${JOINING_CLUSTER_NAME}
if [[ -n `oc get secret -n ${CLUSTER_JOIN_TO_OPERATOR_NS} | grep ${SECRET_NAME}` ]]; then
    oc delete secret ${SECRET_NAME} -n ${CLUSTER_JOIN_TO_OPERATOR_NS}
fi
oc create secret generic ${SECRET_NAME} --from-literal=token="${SA_TOKEN}" --from-literal=ca.crt="${SA_CA_CRT}" -n ${CLUSTER_JOIN_TO_OPERATOR_NS}

TOOLCHAINCLUSTER_CRD="apiVersion: toolchain.dev.openshift.com/v1alpha1
kind: ToolchainCluster
metadata:
  name: ${JOINING_CLUSTER_TYPE_NAME}-${JOINING_CLUSTER_NAME}
  namespace: ${CLUSTER_JOIN_TO_OPERATOR_NS}
  labels:
    type: ${JOINING_CLUSTER_TYPE_NAME}
    namespace: ${OPERATOR_NS}
    ownerClusterName: ${CLUSTER_JOIN_TO}-${CLUSTER_JOIN_TO_NAME}
spec:
  apiEndpoint: ${API_ENDPOINT}
  caBundle: ${SA_CA_CRT}
  secretRef:
    name: ${SECRET_NAME}
"

echo "Creating ToolchainCluster representation of ${JOINING_CLUSTER_TYPE} in ${CLUSTER_JOIN_TO}:"
echo ${TOOLCHAINCLUSTER_CRD}

cat <<EOF | oc apply -f -
${TOOLCHAINCLUSTER_CRD}
EOF
