#!/bin/bash

ROOT=$(cd $(dirname $0)/../; pwd)

set -o errexit
set -o nounset
set -o pipefail

export CLUSER_NAME=$(kubectl config get-contexts |grep `kubectl config current-context` |awk '{print $3}')
export CA_BUNDLE=$(kubectl config view --raw --flatten -o json | jq -r '.clusters[] | select(.name == "'$CLUSER_NAME'") | .cluster."certificate-authority-data"')

#sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g"
rm -rf 3-validatingwebhook-ca.yaml
cp validatingwebhook.tpl 3-validatingwebhook-ca.yaml 
sed -i "s/\${CA_BUNDLE}/${CA_BUNDLE}/g" 3-validatingwebhook-ca.yaml
