#!/usr/bin/env bash
set -euo pipefail  # Exit on errors, unset vars and failed pipes

#####################################
# Secure k3s Cluster Installer (Enhanced)
#####################################

# Define color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Lade Konfiguration
CONFIG_FILE="./cluster.conf"
if [[ -f "$CONFIG_FILE" ]]; then
  source "$CONFIG_FILE"
else
  echo "[ERROR] Konfigurationsdatei $CONFIG_FILE nicht gefunden!"
  exit 1
fi

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

# remote_exec leitet jetzt STDIN an den Remote-bash weiter
remote_exec_for_yaml() {
  local host="$1"; shift
  local user pass

  if [[ "$host" == "$MASTER_IP" ]]; then
    user="$SSH_USER_MASTER"; pass="$SSH_PASS_MASTER"
  elif [[ "$host" == "$WORKER_IP" ]]; then
    user="$SSH_USER_WORKER"; pass="$SSH_PASS_WORKER"
  else
    print_error "Unbekannter Host: $host"; return 1
  fi

  # Everything after 'sudo -S' goes to bash -s, script folgt via STDIN
  sshpass -p "$pass" ssh -tt -o StrictHostKeyChecking=no "$user@$host" \
    "echo '$pass' | sudo -S bash -euo pipefail -s" "$@"       # ← wichtig: -s
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

# Konfiguriert den NFS-Server: Exportverzeichnis, /etc/exports, Berechtigungen
mount_nfs_remote() {
  local host="$1"
  print_info "Konfiguriere NFS-Server auf $host (Export: $NFS_EXPORT)"

  remote_exec "$host" "
    echo '[INFO] Installiere nfs-kernel-server, falls nicht vorhanden'
    if ! dpkg -s nfs-kernel-server >/dev/null 2>&1; then
      apt-get update && apt-get install -y nfs-kernel-server
    else
      echo '[INFO] nfs-kernel-server ist bereits installiert'
    fi

    echo '[INFO] Erstelle Exportverzeichnis: ${NFS_EXPORT}'
    mkdir -p '${NFS_EXPORT}'
    chown nobody:nogroup '${NFS_EXPORT}'
    chmod 777 '${NFS_EXPORT}'

    echo '[INFO] Prüfe /etc/exports auf vorhandene Einträge'
    if grep -qs '${NFS_EXPORT}' /etc/exports; then
      echo '[INFO] Export ist bereits in /etc/exports eingetragen'
    else
      echo '${NFS_EXPORT} ${NFS_SERVER}/24(rw,sync,no_subtree_check,no_root_squash)' >> /etc/exports
      echo '[SUCCESS] Export zu /etc/exports hinzugefügt'
    fi

    echo '[INFO] Lade NFS-Exports neu'
    exportfs -ra
    exportfs -v

    echo '[SUCCESS] NFS-Server auf $host ist bereit'
  "

  if [[ $? -eq 0 ]]; then
    print_success "NFS-Export auf $host erfolgreich eingerichtet"
  else
    print_error "Fehler beim Einrichten des NFS-Exports auf $host"
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


# Install Cert Manager
install_cert_manager() {
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

########################################
# Erstellt den Namespace für den NFS Provisioner
create_nfs_namespace() {

  cat >"nfs-namespace.yaml" <<YAML
apiVersion: v1
kind: Namespace
metadata:
  name: nfs-provisioner
YAML

  # Datei auf den Master kopieren
  sshpass -p "$SSH_PASS_MASTER" \
    scp -o StrictHostKeyChecking=no "nfs-namespace.yaml" \
    "$SSH_USER_MASTER@$MASTER_IP:/tmp/nfs-namespace.yaml"

  # Remote anwenden
  remote_exec "$MASTER_IP" "kubectl apply -f /tmp/nfs-namespace.yaml"
}

########################################
# Erstellt die ServiceAccount-, ClusterRole-, ...
create_nfs_rbac() {
    cat >"nfs-rbac.yaml" <<YAML
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nfs-client-provisioner
  namespace: nfs-provisioner
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nfs-client-provisioner-runner
rules:
  - apiGroups: ['']
    resources: ['nodes']
    verbs: ['get', 'list', 'watch']
  - apiGroups: ['']
    resources: ['persistentvolumes']
    verbs: ['get', 'list', 'watch', 'create', 'delete']
  - apiGroups: ['']
    resources: ['persistentvolumeclaims']
    verbs: ['get', 'list', 'watch', 'update']
  - apiGroups: ['storage.k8s.io']
    resources: ['storageclasses']
    verbs: ['get', 'list', 'watch']
  - apiGroups: ['']
    resources: ['events']
    verbs: ['create', 'update', 'patch']
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: run-nfs-client-provisioner
subjects:
  - kind: ServiceAccount
    name: nfs-client-provisioner
    namespace: nfs-provisioner
roleRef:
  kind: ClusterRole
  name: nfs-client-provisioner-runner
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: leader-locking-nfs-client-provisioner
  namespace: nfs-provisioner
rules:
  - apiGroups: ['']
    resources: ['endpoints']
    verbs: ['get', 'list', 'watch', 'create', 'update', 'patch']
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: leader-locking-nfs-client-provisioner
  namespace: nfs-provisioner
subjects:
  - kind: ServiceAccount
    name: nfs-client-provisioner
    namespace: nfs-provisioner
roleRef:
  kind: Role
  name: leader-locking-nfs-client-provisioner
  apiGroup: rbac.authorization.k8s.io
YAML

  # Datei auf den Master kopieren
  sshpass -p "$SSH_PASS_MASTER" \
    scp -o StrictHostKeyChecking=no "nfs-rbac.yaml" \
    "$SSH_USER_MASTER@$MASTER_IP:/tmp/nfs-rbac.yaml"

  # Remote anwenden
  remote_exec "$MASTER_IP" "kubectl apply -f /tmp/nfs-rbac.yaml"
  remote_exec "$MASTER_IP" "rm /tmp/nfs-rbac.yaml"
}

########################################
# Erstellt die Deployment-Ressource
create_nfs_deployment() {
    cat >"nfs-deployment.yaml" <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-client-provisioner
  namespace: nfs-provisioner
  labels:
    app: nfs-client-provisioner
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: nfs-client-provisioner
  template:
    metadata:
      labels:
        app: nfs-client-provisioner
    spec:
      serviceAccountName: nfs-client-provisioner
      containers:
        - name: nfs-client-provisioner
          image: registry.k8s.io/sig-storage/nfs-subdir-external-provisioner:v4.0.2
          volumeMounts:
            - name: nfs-client-root
              mountPath: /persistentvolumes
          env:
            - name: PROVISIONER_NAME
              value: k8s-sigs.io/nfs-subdir-external-provisioner
            - name: NFS_SERVER
              value: ${NFS_SERVER}
            - name: NFS_PATH
              value: ${NFS_EXPORT}
      volumes:
        - name: nfs-client-root
          nfs:
            server: ${NFS_SERVER}
            path: ${NFS_EXPORT}
YAML

  # Datei auf den Master kopieren
  sshpass -p "$SSH_PASS_MASTER" \
    scp -o StrictHostKeyChecking=no "nfs-deployment.yaml" \
    "$SSH_USER_MASTER@$MASTER_IP:/tmp/nfs-deployment.yaml"

  # Remote anwenden
  remote_exec "$MASTER_IP" "kubectl apply -f /tmp/nfs-deployment.yaml"
  remote_exec "$MASTER_IP" "rm /tmp/nfs-deployment.yaml"
}

########################################
# Erstellt die StorageClass
create_nfs_storageclass() {
cat >"nfs-sc.yaml" <<'YAML'
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-client
provisioner: k8s-sigs.io/nfs-subdir-external-provisioner
parameters:
  pathPattern: "${.PVC.namespace}/${.PVC.name}"
  onDelete: delete
reclaimPolicy: Delete
volumeBindingMode: Immediate
YAML

  # Datei auf den Master kopieren
  sshpass -p "$SSH_PASS_MASTER" \
    scp -o StrictHostKeyChecking=no "nfs-sc.yaml" \
    "$SSH_USER_MASTER@$MASTER_IP:/tmp/nfs-sc.yaml"

  # Remote anwenden
  remote_exec "$MASTER_IP" "kubectl apply -f /tmp/nfs-sc.yaml"
  remote_exec "$MASTER_IP" "rm /tmp/nfs-sc.yaml"
}


# Führt die Installation des NFS Subdir External Provisioners durch
install_nfs_subdir_external_provisioner() {
  print_info "Starte Installation des NFS Subdir External Provisioners auf dem Master (${MASTER_IP})..."
  create_nfs_namespace
  create_nfs_deployment
  create_nfs_rbac
  create_nfs_storageclass
  print_success "NFS Subdir External Provisioner erfolgreich installiert."
}


# uninstall_k3s_node: Deinstalliert k3s (server oder agent) auf einem Host
uninstall_k3s_node() {
  local host="$1"
  print_info "Starte Deinstallation von k3s auf $host..."

  remote_exec "$host" "
    echo '[INFO] Stoppe k3s-Dienste, falls aktiv'
    systemctl stop k3s || true
    systemctl stop k3s-agent || true

    echo '[INFO] Führe Deinstallationsscript aus, falls vorhanden'
    if [ -f /usr/local/bin/k3s-uninstall.sh ]; then
      /usr/local/bin/k3s-uninstall.sh
    fi

    if [ -f /usr/local/bin/k3s-agent-uninstall.sh ]; then
      /usr/local/bin/k3s-agent-uninstall.sh
    fi

    echo '[INFO] Entferne verbleibende Datenverzeichnisse'
    rm -rf /etc/rancher /var/lib/rancher /var/lib/kubelet /etc/cni /opt/cni /var/lib/containerd

    echo '[INFO] K3s-Dienste vollständig entfernt auf $host'
  "

  if [[ $? -eq 0 ]]; then
    print_success "k3s wurde erfolgreich von $host deinstalliert."
  else
    print_error "Fehler bei der Deinstallation von k3s auf $host"
    exit 1
  fi
}


# Main program
main() {
  print_info "Welcome to the k3s cluster setup!"
  print_info "Please select an action:"

  echo -e "${YELLOW}1) Install k3s Master${NC}"
  echo -e "${YELLOW}2) Install k3s Worker${NC}"
  echo -e "${YELLOW}3) Create a NFS mount on worker${NC}"
  echo -e "${YELLOW}4) Create a NFS PV${NC}"
  echo -e "${YELLOW}5) Install Cert Manager ${NC}"
  echo -e "${YELLOW}6) Install full K3s-Cluser  ${NC}"
  echo -e "${YELLOW}7) Install NFS PV  ${NC}"
  echo -e "${YELLOW}99) Deinstall k3s FULL Cluster${NC}"
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
    6)
      install_k3s_master_remote "$MASTER_IP"
      install_k3s_worker_remote "$WORKER_IP"
      mount_nfs_remote "$WORKER_IP"
      create_nfs_pv "$WORKER_IP"
      install_cert_manager "$WORKER_IP"
      ;;
    7)
      install_nfs_subdir_external_provisioner "$WORKER_IP"
      ;;
    0)
      print_info "Exiting."
      exit 0
      ;;
    99)
      uninstall_k3s_node "$MASTER_IP"
      uninstall_k3s_node "$WORKER_IP"
      ;;

    *)
      print_error "Invalid choice! Please enter a valid number."
      ;;
  esac
}

main
