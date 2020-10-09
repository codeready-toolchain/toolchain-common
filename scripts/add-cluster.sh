#!/usr/bin/env bash

user_help () {
    echo "Creates ToolchainCluster"
    echo "options:"
    echo "-t, --type            joining cluster type (host or member)"
    echo "-mn, --member-ns      namespace where member-operator is running"
    echo "-hn, --host-ns        namespace where host-operator is running"
    echo "-s,  --single-cluster running both operators on single cluster"
    echo "-kc,  --kube-config   kubeconfig for managing multiple clusters"
    exit 0
}

login_to_cluster() {
    if [[ ${SINGLE_CLUSTER} != "true" ]]; then
        if [[ -z "${IS_MINISHIFT}" ]]; then
          if [[ -z ${KUBECONFIG} ]]; then
            echo "Write the server url for the cluster type: $1:"
            read -p '> ' CLUSTER_URL
            echo "Provide the token to log in to the cluster ${CLUSTER_URL}:"
            read -sp '> ' CLUSTER_TOKEN
            echo ""
            oc login --token=${CLUSTER_TOKEN} --server=${CLUSTER_URL}
          else
            oc config use-context "$1-admin"
          fi
        else
            echo "Switching to profile $1"
            minishift profile set $1
            oc login -u=system:admin
        fi
    fi
}

create_service_account() {
ROLE_NAME=`oc get Roles -n ${OPERATOR_NS} -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep "^toolchain-${JOINING_CLUSTER_TYPE}-operator\.v"`
echo "using Role ${ROLE_NAME}"
CLUSTER_ROLE_NAME=`oc get ClusterRoles -n ${OPERATOR_NS} -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep "^toolchain-${JOINING_CLUSTER_TYPE}-operator\.v"`
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
  name: ${SA_NAME}
  namespace: ${OPERATOR_NS}
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

IS_MINISHIFT=$(curl -k -XGET -H "Authorization: Bearer $(oc whoami -t 2>/dev/null)" $(oc whoami --show-server)/version/openshift 2>/dev/null | grep paths)
CLUSTER_JOIN_TO="host"

# We need this to configurable to work with dynamic namespaces from end to end tests
OPERATOR_NS=${MEMBER_OPERATOR_NS}
CLUSTER_JOIN_TO_OPERATOR_NS=${HOST_OPERATOR_NS}
if [[ ${JOINING_CLUSTER_TYPE} == "host" ]]; then
  CLUSTER_JOIN_TO="member"
  OPERATOR_NS=${HOST_OPERATOR_NS}
  CLUSTER_JOIN_TO_OPERATOR_NS=${MEMBER_OPERATOR_NS}
fi

# This is using default values i.e. toolchain-member-operator or toolchain-host-operator for local setup
if [[ ${OPERATOR_NS} == "" &&  ${CLUSTER_JOIN_TO_OPERATOR_NS} == "" ]]; then
  OPERATOR_NS=toolchain-${JOINING_CLUSTER_TYPE}-operator
  CLUSTER_JOIN_TO_OPERATOR_NS=toolchain-${CLUSTER_JOIN_TO}-operator
fi

SA_NAME="toolchaincluster-${JOINING_CLUSTER_TYPE}-operator"

echo ${OPERATOR_NS}
echo ${CLUSTER_JOIN_TO_OPERATOR_NS}

login_to_cluster ${JOINING_CLUSTER_TYPE}

create_service_account

echo "Getting ${JOINING_CLUSTER_TYPE} SA token"
SA_SECRET=`oc get sa ${SA_NAME} -n ${OPERATOR_NS} -o json | jq -r .secrets[].name | grep token`
SA_TOKEN=`oc get secret ${SA_SECRET} -n ${OPERATOR_NS}  -o json | jq -r '.data["token"]' | base64 --decode`
SA_CA_CRT=`oc get secret ${SA_SECRET} -n ${OPERATOR_NS} -o json | jq -r '.data["ca.crt"]'`

# the api endpoint and name retrieve from kubeconfig only for minishift, in case of OS4 use the oc infrastructure cluster command
if [[ -n "${IS_MINISHIFT}" ]]; then
    echo "Running locally in minishift environment"
    API_ENDPOINT=`oc config view --raw --minify -o json | jq -r '.clusters[0].cluster["server"]'`
    JOINING_CLUSTER_NAME=`oc config view --raw --minify -o json | jq -r '.clusters[0].name' | sed 's/[^[:alnum:]._-]/-/g'`
else
    echo "Running in openshift 4.x cluster"
    API_ENDPOINT=`oc get infrastructure cluster -o jsonpath='{.status.apiServerURL}'`
    JOINING_CLUSTER_NAME=`oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'`
fi

login_to_cluster ${CLUSTER_JOIN_TO}

# the api endpoint and name retrieve from kubeconfig only for minishift, in case of OS4 use the oc infrastructure cluster command
if [[ -n "${IS_MINISHIFT}" ]]; then
    echo "Running locally in minishift environment"
    CLUSTER_JOIN_TO_NAME=`oc config view --raw --minify -o json | jq -r '.clusters[0].name' | sed 's/[^[:alnum:]._-]/-/g'`
else
    echo "Running in openshift 4.x cluster"
    CLUSTER_JOIN_TO_NAME=`oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'`
fi

SECRET_NAME=${SA_NAME}-${JOINING_CLUSTER_NAME}
if [[ -n `oc get secret -n ${CLUSTER_JOIN_TO_OPERATOR_NS} | grep ${SECRET_NAME}` ]]; then
    oc delete secret ${SECRET_NAME} -n ${CLUSTER_JOIN_TO_OPERATOR_NS}
fi
oc create secret generic ${SECRET_NAME} --from-literal=token="${SA_TOKEN}" --from-literal=ca.crt="${SA_CA_CRT}" -n ${CLUSTER_JOIN_TO_OPERATOR_NS}

TOOLCHAINCLUSTER_CRD="apiVersion: toolchain.dev.openshift.com/v1alpha1
kind: ToolchainCluster
metadata:
  name: ${JOINING_CLUSTER_TYPE}-${JOINING_CLUSTER_NAME}
  namespace: ${CLUSTER_JOIN_TO_OPERATOR_NS}
  labels:
    type: ${JOINING_CLUSTER_TYPE}
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
