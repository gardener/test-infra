#!/usr/bin/env bash
# Creates a token-based kubeconfig suitable for the conformance test container,
# which cannot use credential plugins. Requires cluster-admin access via $KUBECONFIG.
# outputs a kubeconfig file at the path specified as the first argument, or conformance-kubeconfig.yaml if none is provided.

set -euo pipefail

NAMESPACE="kube-system"
SA_NAME="conformance-sa"
CRB_NAME="conformance-sa-admin"
TOKEN_DURATION="12h"
OUTPUT_KUBECONFIG="${1:-conformance-kubeconfig.yaml}"

echo "Creating ServiceAccount '$SA_NAME' in namespace '$NAMESPACE'..."
kubectl create serviceaccount "$SA_NAME" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo "Binding '$SA_NAME' to cluster-admin..."
kubectl create clusterrolebinding "$CRB_NAME" \
  --clusterrole=cluster-admin \
  --serviceaccount="${NAMESPACE}:${SA_NAME}" --dry-run=client -o yaml | kubectl apply -f -

echo "Requesting token (duration: $TOKEN_DURATION)..."
TOKEN=$(kubectl create token "$SA_NAME" -n "$NAMESPACE" --duration="$TOKEN_DURATION")

echo "Extracting cluster info from current kubeconfig..."
CA_TMPFILE=$(mktemp)
trap 'rm -f "$CA_TMPFILE"' EXIT

kubectl config view --minify --raw \
  -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' \
  | base64 -d > "$CA_TMPFILE"

CLUSTER_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

echo "Building token-based kubeconfig at '$OUTPUT_KUBECONFIG'..."
kubectl config set-cluster conformance-cluster \
  --server="$CLUSTER_SERVER" \
  --certificate-authority="$CA_TMPFILE" \
  --embed-certs=true \
  --kubeconfig="$OUTPUT_KUBECONFIG"

kubectl config set-credentials conformance-user \
  --token="$TOKEN" \
  --kubeconfig="$OUTPUT_KUBECONFIG"

kubectl config set-context default \
  --cluster=conformance-cluster \
  --user=conformance-user \
  --kubeconfig="$OUTPUT_KUBECONFIG"

kubectl config use-context default --kubeconfig="$OUTPUT_KUBECONFIG"

echo "Done. Kubeconfig written to: $OUTPUT_KUBECONFIG"
echo "Token expires in $TOKEN_DURATION. Mount it into the container with:"
echo "  docker run ... -v \$PWD/$OUTPUT_KUBECONFIG:/mye2e/shoot.config -e KUBECONFIG=/mye2e/shoot.config ..."
echo ""
echo "Clean up manually after the tests complete:"
echo "  kubectl delete clusterrolebinding $CRB_NAME"
echo "  kubectl delete serviceaccount $SA_NAME -n $NAMESPACE"