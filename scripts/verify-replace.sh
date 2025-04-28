# This script is written to check whether any changes to toolchain-common, 
# related API changes need update/changes in other related repos or not
# To run this script there is make command 'make verify-replace-run' or 
# you can directly run this script.
# if you have changes in toolchain-common, run this script and it will 
# give you the result with, if there needs to be any change/update in other repos.

#!/bin/bash
TMP_DIR=/tmp/
BASE_REPO_PATH=$(mktemp -d ${TMP_DIR}replace-verify.XXX)
GH_BASE_URL_KS=https://github.com/kubesaw/
GH_BASE_URL_CRT=https://github.com/codeready-toolchain/
declare -a REPOS=("${GH_BASE_URL_KS}ksctl" "${GH_BASE_URL_CRT}host-operator" "${GH_BASE_URL_CRT}member-operator" "${GH_BASE_URL_CRT}registration-service" "${GH_BASE_URL_CRT}toolchain-e2e")
C_PATH=${PWD}
API_PATH=github.com/codeready-toolchain/api
TC_PATH=github.com/codeready-toolchain/toolchain-common
API_REPLACE_PATH=$(go mod edit -json | jq '.Replace[]?|select(.Old.Path=="github.com/codeready-toolchain/api")|.New.Path' -r)
API_REPLACE_VERSION=$(go mod edit -json | jq '.Replace[]?|select(.Old.Path=="github.com/codeready-toolchain/api")|.New.Version' -r)
ERROR_REPO_LIST=()
ERROR_FILE_LIST=()
STD_OUT_FILE_LIST=()
GO_LINT_REGEX="[\s\w.\/]*:[0-9]*:[0-9]*:[\w\s)(*.\`]*"
# unit test or any other failure we log from our controllers or other places goes into stdoutput 
# (since we log it and its not a failure in running the command or dependency check), 
# hence making that regex too, to fetch the error more precisely
ERROR_REGEX="[E\|e][R\|r][R\|r][O\|o][R\|r][:]*\|[F\|f][A\|a][I\|i][l\|L][:]*\|expected[:]*\|actual[:]*" 

echo Initiating verify-replace on dependent repos
for repo in "${REPOS[@]}"
do
    echo =========================================================================================
    echo  
    echo                        "$(basename ${repo})"
    echo                                                                     
    echo =========================================================================================                                            
    repo_path=${BASE_REPO_PATH}/$(basename ${repo})
    err_file=$(mktemp ${BASE_REPO_PATH}/$(basename ${repo})-error.XXX)
    echo "error output file : ${err_file}"
    std_out_file=$(mktemp ${BASE_REPO_PATH}/$(basename ${repo})-output.XXX)
    echo "std output file : ${std_out_file}"
    echo "Cloning repo in /tmp"
    git clone --depth=1 ${repo} ${repo_path}
    echo "Repo cloned successfully"
    cd ${repo_path}
    make pre-verify 2> >(tee ${err_file})
    rc=$?
    if [ ${rc} -ne 0 ]; then
        ERROR_REPO_LIST+="$(basename ${repo}) "
        ERROR_FILE_LIST+="${err_file}  "
        continue
    fi
    echo "Initiating 'go mod replace' of current toolchain common version in dependent repos"
    go mod edit -replace ${TC_PATH}=${C_PATH}
    # we are only fetching api replace and hence 
    # check if there is any api replace in toolchain common - if there is
    # propogate the same to other repos along with toolchain-common replace
    if [[ -n "${API_REPLACE_PATH}" && -n "${API_REPLACE_VERSION}" ]]; then 
    echo "Initiating 'go mod replace' of api as replace of api is present in toolchain-common"
    go mod edit -replace ${API_PATH}=${API_REPLACE_PATH}@${API_REPLACE_VERSION}
    fi
    make verify-dependencies 2> >(tee ${err_file}) 1> >(tee ${std_out_file})
    rc=$?
    if [ ${rc} -ne 0 ]; then
    ERROR_REPO_LIST+="$(basename ${repo}) " 
    ERROR_FILE_LIST+="${err_file}  "
    STD_OUT_FILE_LIST+="${std_out_file} "
    fi
    echo                                                          
    echo =========================================================================================
    echo                                                           
done
echo                "Summary"
if [ ${#ERROR_REPO_LIST[@]} -ne 0 ]; then
    echo "Below are the repos with error: "
    for error_repo_name in ${ERROR_REPO_LIST[*]}
    do
        echo                                                          
        echo =========================================================================================
        echo 
        echo                       "${error_repo_name} has the following errors "
        echo                                                          
        echo =========================================================================================
        echo 
        for error_file_name in ${ERROR_FILE_LIST[*]}
        do
            if [[ ${error_file_name} =~ ${error_repo_name} ]]; then 
                cat "${error_file_name}"
            fi
        done
        for std_out_file_name in ${STD_OUT_FILE_LIST[*]}
            do
                if [[ ${std_out_file_name} =~ ${error_repo_name} ]]; then 
                    cat "${std_out_file_name}" | grep -C 5 "${GO_LINT_REGEX}\|${ERROR_REGEX}"
                fi
        done                                             
    done
    exit 1
else
    echo "No errors detected"
fi