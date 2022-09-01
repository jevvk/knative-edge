#!/bin/bash
set -e
set -o pipefail
set -o errexit

# Add user to k8s using service account, no RBAC (must create RBAC after this script)
if [[ -z "$1" ]] || [[ -z "$2" ]] || [[ -z "$3" ]] || [[ -z "$4" ]]; then
 echo "usage: $0 <cluster_name> <cluster_url> <service_account_name> <namespace>"
 exit 1
fi

CLUSTER_NAME="$1"
CLUSTER_URL="$2"
SERVICE_ACCOUNT_NAME="$3"
NAMESPACE="$4"

ca=$(kubectl config view --minify --raw --output 'jsonpath={..cluster.certificate-authority-data}')
token=$(kubectl create token --namespace $NAMESPACE $SERVICE_ACCOUNT_NAME)

echo "
---
apiVersion: v1
kind: Config
clusters:
  - name: ${CLUSTER_NAME}
    cluster:
      certificate-authority-data: ${ca}
      server: ${CLUSTER_URL}
contexts:
  - name: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
    context:
      cluster: ${CLUSTER_NAME}
      namespace: ${NAMESPACE}
      user: ${SERVICE_ACCOUNT_NAME}
users:
  - name: ${SERVICE_ACCOUNT_NAME}
    user:
      token: ${token}
current-context: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
"
