package internal

import (
	"fmt"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

func restartDeployment(node config.NodeConfig, namespace, deploy string) error {
	cmd := fmt.Sprintf(`
echo '%[1]s' | sudo -S bash -c '
  echo "[INFO] Triggering rollout restart for %[2]s/%[3]s"
  kubectl rollout restart deployment/%[3]s -n %[2]s || true

  echo "[INFO] Deleting old pods (if stuck)..."
  kubectl delete pod -n %[2]s -l app=ic-docker-registry --grace-period=0 --force || true
'`, node.SSHPass, namespace, deploy)

	return remote.RemoteExec(node.SSHUser, node.SSHPass, node.IP, cmd)
}
