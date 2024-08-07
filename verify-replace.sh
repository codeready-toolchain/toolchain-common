#!/bin/bash
TMP_DIR=/tmp/
BASE_REPO_PATH=$(mktemp -d ${TMP_DIR}replace-verify.XXX)
GH_BASE_URL_KS=https://github.com/kubesaw/
GH_BASE_URL_CRT=https://github.com/codeready-toolchain/
GH_KSCTL=${GH_BASE_URL_KS}ksctl
GH_HOST=${GH_BASE_URL_CRT}host-operator
GH_MEMBER=${GH_BASE_URL_CRT}member-operator
GH_REGSVC=${GH_BASE_URL_CRT}registration-service
GH_E2E=${GH_BASE_URL_CRT}toolchain-e2e
GH_TC=${GH_BASE_URL_CRT}toolchain-common
C_PATH=${PWD}
ERRORLIST=()

for repo in ${GH_HOST} ${GH_REGSVC} ${GH_KSCTL} ${GH_MEMBER} ${GH_E2E}
do
    REPO_PATH=$BASE_REPO_PATH/$(basename $repo)
    git clone --depth=1 $repo $REPO_PATH
    cd $REPO_PATH
    go mod edit -replace github.com/codeready-toolchain/toolchain-common=$C_PATH
    make verify-dependencies || ERRORLIST+=($(basename $repo))
done
echo Errors in repos:
for e in ${ERRORLIST[*]}
do
    echo $e
done