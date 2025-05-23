#!/usr/bin/env bash

set -euo pipefail

# Function to display usage information
usage() {
  echo "Usage: $0 [-n <namespace>] [-o <output-dir>]"
  echo
  echo "Options:"
  echo "  -n <namespace>    Kubernetes namespace to inspect (default: all namespaces)"
  echo "  -o <output-dir>   Directory to save outputs (default: current directory)"
  echo "  -h                Show this help message"
  exit 1
}

OUTPUT_DIR="./logs"
NAMESPACE=""

# Parse command-line arguments
while getopts ":n:o:h" opt; do
  case ${opt} in
    n )
      NAMESPACE=$OPTARG
      ;;
    o )
      OUTPUT_DIR=$OPTARG
      ;;
    h )
      usage
      ;;
    \? )
      echo "Invalid option: -$OPTARG" >&2
      usage
      ;;
    : )
      echo "Option -$OPTARG requires an argument." >&2
      usage
      ;;
  esac
done

# Determine filename suffix
if [[ -n "$NAMESPACE" ]]; then
  SUFFIX="$NAMESPACE"
else
  SUFFIX="all-namespaces"
fi

mkdir -p "$OUTPUT_DIR"

echo "📦 Gathering resource definitions in ${NAMESPACE:-all namespaces}..."
if [[ -n "$NAMESPACE" ]]; then
  kubectl get svc,ingress,pods,pv,pvc -n "$NAMESPACE" -o yaml \
    > "$OUTPUT_DIR/resources_${SUFFIX}.yaml"
else
  kubectl get svc,ingress,pods,pv,pvc --all-namespaces -o yaml \
    > "$OUTPUT_DIR/resources_${SUFFIX}.yaml"
fi
echo "👉 Resource definitions saved to $OUTPUT_DIR/resources_${SUFFIX}.yaml"

echo "📝 Fetching logs from all pods in ${NAMESPACE:-all namespaces}..."

# Collect pod names (with namespace prefix if all namespaces)
if [[ -n "$NAMESPACE" ]]; then
  PODS=$(kubectl get pods -n "$NAMESPACE" -o name)
else
  PODS=$(kubectl get pods --all-namespaces -o name)
fi

LOG_FILE="$OUTPUT_DIR/logs_${SUFFIX}.txt"

# Temporarily disable exit-on-error for the loop
set +e

if [[ -z "$PODS" ]]; then
  echo "⚠️  No pods found in ${NAMESPACE:-all namespaces}."
else
  for pod in $PODS; do
    echo "=== Logs for $pod ===" >> "$LOG_FILE"
    # try to fetch logs; on failure, write warning and continue
    if [[ -n "$NAMESPACE" ]]; then
      kubectl logs "$pod" -n "$NAMESPACE" --all-containers=true --timestamps \
        >> "$LOG_FILE" 2>> "$LOG_FILE"
      if [[ $? -ne 0 ]]; then
        echo "⚠️  Could not fetch logs for $pod" >> "$LOG_FILE"
      fi
    else
      kubectl logs "$pod" --all-containers=true --timestamps \
        >> "$LOG_FILE" 2>> "$LOG_FILE"
      if [[ $? -ne 0 ]]; then
        echo "⚠️  Could not fetch logs for $pod" >> "$LOG_FILE"
      fi
    fi
    echo >> "$LOG_FILE"
  done
  echo "👉 Logs saved to $LOG_FILE"
fi

# Re-enable exit-on-error
set -e

echo "✅ Fertig!"
