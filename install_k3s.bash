#!/bin/bash
set -euo pipefail  # Exit on errors, unset vars and failed pipes

#####################################
# Secure k3s Cluster Installer (Enhanced)
# Master: 37.120.167.64
# Worker: 193.31.28.86
# Run as root!
#####################################

# Define color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Define variables

MASTER_IP="192.168.0.1"
SSH_USER_MASTER="USER"
SSH_PASS_MASTER="PASSWORD"
K3S_TOKEN_FILE="master-node-token"
WORKER_IP="192.168.0.2"
SSH_USER_WORKER="kubernetes"
SSH_PASS_WORKER="PASSWORD"
NFS_SERVER="10.0.0.10"
NFS_EXPORT="/mnt/k3s-nfs-localstorage"
NFS_MOUNTPOINT="/mnt/nfs-igneos-cloud-k3s"
CAPACITY="100Gi"

EMAIL="email@andrejschefer.de"   # Change to your email
DOMAIN="demo.example.com"   # Change to your domain
CLUSTER_ISSUER_NAME="letsencrypt-prod"

# Print success message
print_success() {
  echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Print error message
print_error() {
  echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Print info message
print_info() {
  echo -e "${YELLOW}[INFO]${NC} $1"
}

# Function: Check if running as root
check_root() {
  if [[ "$EUID" -ne 0 ]]; then
    print_error "This script must be run as root!"
    exit 1
  fi
}

# remote_exec: Executes a command on the target host with credentials
# remote_exec: Executes a command on the target host with visible live output and clean exit
remote_exec() {
  local host="$1"; shift

  local user pass
  if [[ "$host" == "$MASTER_IP" ]]; then
    user="$SSH_USER_MASTER"
    pass="$SSH_PASS_MASTER"
  elif [[ "$host" == "$WORKER_IP" ]]; then
    user="$SSH_USER_WORKER"
    pass="$SSH_PASS_WORKER"
  else
    print_error "Unbekannter Host: $host"
    return 1
  fi

  # Starte remote Kommando mit sudo und passwort, ohne Heredocs
  sshpass -p "$pass" ssh -tt -o StrictHostKeyChecking=no "$user@$host" \
    "echo '$pass' | sudo -S bash -euo pipefail -c \"$*\""
  
  local status=$?
  if [[ $status -ne 0 ]]; then
    print_error \"Remote execution failed on $host with exit code $status\"
    exit $status
  fi
}
# remote_exec_output: Führt Befehl aus und gibt stdout als Rückgabewert zurück
remote_exec_output() {
  local host="$1"; shift

  local user pass
  if [[ "$host" == "$MASTER_IP" ]]; then
    user="$SSH_USER_MASTER"
    pass="$SSH_PASS_MASTER"
  elif [[ "$host" == "$WORKER_IP" ]]; then
    user="$SSH_USER_WORKER"
    pass="$SSH_PASS_WORKER"
  else
    print_error "Unbekannter Host: $host"
    return 1
  fi

  sshpass -p "$pass" ssh -o StrictHostKeyChecking=no "$user@$host" \
    "echo '$pass' | sudo -S bash -euo pipefail -c \"$*\"" 2>/dev/null
}

# Function: Install k3s Master
install_k3s_master() {
  print_info "Installing k3s Master Node"

  # Install K3s with writeable kubeconfig and encryption enabled
  curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode=644 --secrets-encryption" sh -s - server

  # Set the KUBECONFIG environment variable temporarily for this session
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

  # Ensure the local kube directory exists
  mkdir -p ~/.kube

  # Copy the kubeconfig to the local kube directory
  cp /etc/rancher/k3s/k3s.yaml ~/.kube/config

  # Determine the current logged-in user
  CURRENT_USER=$(logname)

  # Change ownership of the kubeconfig to the current logged-in user
  chown "$CURRENT_USER":"$CURRENT_USER" ~/.kube/config

  # Set secure permissions for the kubeconfig file
  chmod 600 ~/.kube/config

  # Replace the server address from localhost to actual IP address
  SERVER_IP=$(hostname -I | awk '{print $1}')
  sed -i "s/127.0.0.1/$SERVER_IP/" ~/.kube/config

  print_success "k3s Master installation and kubeconfig setup completed"
}

# install_k3s_master_remote: Install and verify k3s master on remote host
install_k3s_master_remote() {
  local host="$1"
  print_info "Installing k3s server on $host"

  remote_exec "$host" "
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='--write-kubeconfig-mode=644 --secrets-encryption --tls-san=$DOMAIN' sh -s - server &&
    mkdir -p /home/$SSH_USER_MASTER/.kube &&
    cp /etc/rancher/k3s/k3s.yaml /home/$SSH_USER_MASTER/.kube/config &&
    chown $SSH_USER_MASTER:$SSH_USER_MASTER /home/$SSH_USER_MASTER/.kube/config &&
    chmod 600 /home/$SSH_USER_MASTER/.kube/config &&
    SERVER_IP=\$(hostname -I | awk '{print \$1}') &&
    sed -i \"s/127\\.0\\.0\\.1/\$SERVER_IP/\" /home/kubernetes/.kube/config
  "

  fetch_k3s_token
  fetch_kubeconfig_local
  print_success "k3s Master auf $host erfolgreich installiert!"
}

fetch_k3s_token() {
  print_info "Lese node-token vom Master ($MASTER_IP)..."

  local token_raw
  token_raw=$(remote_exec_output "$MASTER_IP" "cat /var/lib/rancher/k3s/server/node-token")

  if [[ -z "$token_raw" ]]; then
    print_error "Keine Antwort vom Master erhalten! Remote-Befehl schlug fehl?"
    exit 1
  fi

  # Nur Zeile mit Token behalten
  local token
  token=$(echo "$token_raw" | grep '^K')

  print_info $token_raw

  if [[ -z "$token" ]]; then
    print_error "Kein gültiger Token gefunden in: $token_raw"
    exit 1
  fi

  echo "$token" > "$K3S_TOKEN_FILE"
  chmod 600 "$K3S_TOKEN_FILE"

  print_success "Token erfolgreich gespeichert unter $K3S_TOKEN_FILE"
}

# fetch_kubeconfig_local: Copy kubeconfig locally
fetch_kubeconfig_local() {
  print_info "Fetching kubeconfig from master"
  mkdir -p ~/.kube
  sshpass -p "$SSH_PASS_MASTER" scp -o StrictHostKeyChecking=no "$SSH_USER_MASTER@$MASTER_IP:/etc/rancher/k3s/k3s.yaml" ~/.kube/config
  sed -i "s/127.0.0.1/${MASTER_IP}/" ~/.kube/config
  chmod 600 ~/.kube/config
  print_success "kubeconfig ready locally"
}

# install_k3s_worker_remote: Install k3s agent on remote worker node
install_k3s_worker_remote() {
  local host="$1"
  print_info "Installing k3s agent on worker node $host"

  if [ ! -f "$K3S_TOKEN_FILE" ]; then
    print_error "Master token file not found at $K3S_TOKEN_FILE"
    exit 1
  fi

  local token
  token=$(<"$K3S_TOKEN_FILE")

  remote_exec "$host" "
    curl -sfL https://get.k3s.io | K3S_URL='https://$MASTER_IP:6443' K3S_TOKEN='$token' sh -s - agent
  "

  print_info "Verifying k3s-agent service on $host"
  if remote_exec "$host" "systemctl is-active --quiet k3s-agent"; then
    print_success "k3s agent on $host is active and ready!"
  else
    print_error "k3s agent failed to start on $host"
    exit 1
  fi
}

# remote_exec: führt Befehl auf dem Zielhost mit sudo-Passwortübergabe aus
remote_exec_with_root() {
  local host="$1"; shift

  local user pass
  if [[ "$host" == "$MASTER_IP" ]]; then
    user="$SSH_USER_MASTER"
    pass="$SSH_PASS_MASTER"
  elif [[ "$host" == "$WORKER_IP" ]]; then
    user="$SSH_USER_WORKER"
    pass="$SSH_PASS_WORKER"
  else
    print_error "Unbekannter Host: $host"
    return 1
  fi

  # SSH: führe Befehl direkt aus, Pipe an sudo, keine doppelte Bash-Schachtelung!
  sshpass -p "$pass" ssh -tt -o StrictHostKeyChecking=no "$user@$host" \
  "echo \"$pass\" | sudo -S bash -euo pipefail"
}

mount_nfs_remote() {
  local host="$1"
  print_info "Mounting NFS on worker node $host (mit sudo-Passwort)"

  remote_exec_with_root "$host" "
  echo '[INFO] Erstelle Mountpoint: ${NFS_MOUNTPOINT}'
  mkdir -p ${NFS_MOUNTPOINT} || exit 1

  echo '[INFO] Versuche NFS-Mount: ${NFS_SERVER}:${NFS_EXPORT}'
  if mount -t nfs ${NFS_SERVER}:${NFS_EXPORT} ${NFS_MOUNTPOINT}; then
    echo '[SUCCESS] NFS erfolgreich gemountet'
  else
    echo '[ERROR] NFS konnte nicht gemountet werden' >&2
    exit 1
  fi

  if grep -qs '${NFS_MOUNTPOINT}' /etc/fstab; then
    echo '[INFO] NFS bereits in /etc/fstab'
  else
    echo '${NFS_SERVER}:${NFS_EXPORT} ${NFS_MOUNTPOINT} nfs defaults 0 0' >> /etc/fstab
    echo '[SUCCESS] NFS in /etc/fstab eingetragen'
  fi
  "


  if [[ $? -eq 0 ]]; then
    print_success "NFS auf $host erfolgreich gemountet und konfiguriert"
  else
    print_error "Fehler beim NFS-Mount auf $host"
    exit 1
  fi
}


# Function: Create NFS PersistentVolume
create_nfs_pv() {
  print_info "Creating NFS PersistentVolume"
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-nfs-data
spec:
  capacity:
    storage: $CAPACITY
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: ""
  nfs:
    path: $NFS_EXPORT
    server: $NFS_SERVER
EOF
  print_success "NFS PersistentVolume created"
}

install_cert_menager() {
  print_info "Post-Konfiguration des Clusters auf dem Master ($MASTER_IP) wird durchgeführt..."

  remote_exec_with_root "$MASTER_IP" "
    echo '[INFO] Installing cert-manager...'
    kubectl apply --validate=false -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.3/cert-manager.yaml || exit 1

    echo '[INFO] Waiting for cert-manager-webhook to become ready...'
    kubectl -n cert-manager rollout status deployment cert-manager-webhook --timeout=90s || exit 1

    echo '[INFO] Creating ClusterIssuer...'
    cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: ${CLUSTER_ISSUER_NAME}
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: ${EMAIL}
    privateKeySecretRef:
      name: ${CLUSTER_ISSUER_NAME}-account
    solvers:
    - http01:
        ingress:
          class: traefik
EOF
  "

  print_success "Cluster vollständig konfiguriert (cert-manager, ClusterIssuer, PersistentVolume)."
}




# Main program
main() {
  print_info "Welcome to the k3s cluster setup!"
  print_info "Please select an action:"

  echo -e "${YELLOW}1) Remote Install k3s Master${NC}"
  echo -e "${YELLOW}2) Remote Install k3s Worker${NC}"
  echo -e "${YELLOW}3) Create a NFS mount on worker${NC}"
  echo -e "${YELLOW}4) Create a NFS PV${NC}"
  echo -e "${YELLOW}5) Install Cert Manager ${NC}"
  echo -e "${YELLOW}0) Exit${NC}"

  read -rp "$(echo -e "${YELLOW}Enter your choice: ${NC}")" CHOICE

  case "$CHOICE" in
    1)
      install_k3s_master_remote "$MASTER_IP"
      ;;
    2)
      install_k3s_worker_remote "$WORKER_IP"
      ;;
    3)
      mount_nfs_remote "$WORKER_IP"
      ;;
    4)
      create_nfs_pv "$WORKER_IP"
      ;;
    5)
      install_cert_menager "$WORKER_IP"
      ;;
    0)
      print_info "Exiting."
      exit 0
      ;;
    *)
      print_error "Invalid choice! Please enter a valid number."
      ;;
  esac
}

main
