#!/usr/bin/env bash

# Exit on error
set -e

user_help () {
    echo "Updates kubeFedClusterCrd const in the given *.go file"
    echo "options:"
    echo "-c, --crd path to CRD file to be used as the value of kubeFedClusterCrd const"
    echo "-s, --source path to .go file where the kubeFedClusterCrd const to updated"
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
            -c|--crd)
                shift
                CRD_FILE=$1
                shift
                ;;
            -s|--source)
                shift
                SRC=$1
                shift
                ;;
            *)
               echo "$1 is not a recognized flag!"
               user_help
               exit -1
               ;;
      esac
done

# Delete two first lines of CRDs ("\n----\n") from the generated CRD to make a single manifest file out of the original multiple manifest file
# Also we remove the line with 'type: object' from validation.openAPIV3Schema.properties path because it's incompatible with kube 1.11 which is used by minishift
echo "Updating $SRC..."
CRD=`sed -e '1,2d' -e '/^      type: object/d' $CRD_FILE`
printf -v ESCAPED_CRD "%q" "$CRD"
sed '/const/Q' $SRC | sed "\$aconst kubeFedClusterCrd = \`$ESCAPED_CRD\`" | sed -e "s/$'/\n/" -e "s/'\`/\n\`/" > deploy/crds/kubefedcluster_crd.go
mv deploy/crds/kubefedcluster_crd.go $SRC
