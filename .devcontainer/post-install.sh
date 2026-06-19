#!/bin/bash
set -euo pipefail

curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
mv ./kind /usr/local/bin/kind

OPERATOR_SDK_VERSION=v1.42.2
curl -sSL "https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_linux_amd64" -o operator-sdk
chmod +x operator-sdk
mv operator-sdk /usr/local/bin/operator-sdk

KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/kubectl

docker network create -d=bridge --subnet=172.19.0.0/24 kind 2>/dev/null || true

kind version
operator-sdk version
go version
kubectl version --client
