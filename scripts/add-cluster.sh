#!/usr/bin/env bash

user_help () {
    echo "Creates KubeFedCluster"
    echo "options:"
    echo "-t, --type            joining cluster type (host or member)"
    echo "-mn, --member-ns      namespace where member-operator is running"
    echo "-hn, --host-ns        namespace where host-operator is running"
    echo "-s,  --single-cluster running both operators on single cluster"
    exit 0
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

# This is using default values i.e. toolchain-member-operator or toolchain-host-operator for local setup
if [[ ${OPERATOR_NS} == "" &&  ${CLUSTER_JOIN_TO_OPERATOR_NS} == "" ]]; then
  OPERATOR_NS=toolchain-${JOINING_CLUSTER_TYPE}-operator
  CLUSTER_JOIN_TO_OPERATOR_NS=toolchain-${CLUSTER_JOIN_TO}-operator
fi

SA_NAME=${JOINING_CLUSTER_TYPE}"-operator"

echo ${OPERATOR_NS}
echo ${CLUSTER_JOIN_TO_OPERATOR_NS}

# This is to work with multiple profiles of minishift. By default profile is true
if [[ ${SINGLE_CLUSTER} != "true" ]]; then
  echo "Switching to profile ${JOINING_CLUSTER_TYPE}"
  minishift profile set ${JOINING_CLUSTER_TYPE}
  oc login -u=system:admin
fi

echo "Getting ${JOINING_CLUSTER_TYPE} SA token"
SA_SECRET=`oc get sa ${SA_NAME} -n ${OPERATOR_NS} -o json | jq -r .secrets[].name | grep token`
SA_TOKEN=`oc get secret ${SA_SECRET} -n ${OPERATOR_NS}  -o json | jq -r '.data["token"]' | base64 --decode`
SA_CA_CRT=`oc get secret ${SA_SECRET} -n ${OPERATOR_NS} -o json | jq -r '.data["ca.crt"]'`

# this env variable is set in openshift-ci environment.
# openshift ci has long name for cluster i.e.> 63 characters if read from config which is not allowed by k8s/openshift
if [[ -z ${OPENSHIFT_BUILD_NAMESPACE} ]]; then
    echo "Running locally in minishift environment"
    API_ENDPOINT=`oc config view --raw --minify -o json | jq -r '.clusters[0].cluster["server"]'`
    JOINING_CLUSTER_NAME=`oc config view --raw --minify -o json | jq -r '.clusters[0].name' | sed 's/[^[:alnum:]._-]/-/g'`
else
    echo "Running in openshift-ci environment with openshift 4.x cluster"
    API_ENDPOINT=`oc get infrastructure cluster -o jsonpath='{.status.apiServerURL}'`
    JOINING_CLUSTER_NAME=`oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'`
fi

# This is to work with multiple profiles of minishift. By default profile is true
if [[ ${SINGLE_CLUSTER} != "true" ]]; then
  echo "Switching to profile ${CLUSTER_JOIN_TO}"
  minishift profile set ${CLUSTER_JOIN_TO}
  oc login -u=system:admin
fi

# this env variable is set in openshift-ci environment.
# openshift ci has long name for cluster i.e.> 63 characters if read from config which is not allowed by k8s/openshift
if [[ -z ${OPENSHIFT_BUILD_NAMESPACE} ]]; then
    echo "Running locally in minishift environment"
    CLUSTER_JOIN_TO_NAME=`oc config view --raw --minify -o json | jq -r '.clusters[0].name' | sed 's/[^[:alnum:]._-]/-/g'`
else
    echo "Running in openshift-ci environment with openshift 4.x cluster"
    CLUSTER_JOIN_TO_NAME=`oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'`
fi


oc create secret generic ${SA_NAME}-${JOINING_CLUSTER_NAME} --from-literal=token="${SA_TOKEN}" --from-literal=ca.crt="${SA_CA_CRT}" -n ${CLUSTER_JOIN_TO_OPERATOR_NS}

KUBEFEDCLUSTER_CRD="apiVersion: core.kubefed.k8s.io/v1beta1
kind: KubeFedCluster
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
    name: ${SA_NAME}-${JOINING_CLUSTER_NAME}
"

echo "Creating KubeFedCluster representation of ${JOINING_CLUSTER_TYPE} in ${CLUSTER_JOIN_TO}:"
echo ${KUBEFEDCLUSTER_CRD}

cat <<EOF | oc apply -f -
${KUBEFEDCLUSTER_CRD}
EOF
