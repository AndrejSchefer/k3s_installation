package internal

import (
	"fmt"
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

func InstallK3sMaster() error {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfiguration: %v", err)
	}

	fmt.Println("Master Nodes:")
	for _, master := range cfg.Masters {

		command := fmt.Sprintf(`
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='--write-kubeconfig-mode=644 --secrets-encryption --tls-san=%s' sh -s - server &&
mkdir -p /home/%s/.kube &&
cp /etc/rancher/k3s/k3s.yaml /home/%s/.kube/config &&
chown %s:%s /home/%s/.kube/config &&
chmod 600 /home/%s/.kube/config &&
SERVER_IP=$(hostname -I | awk '{print $1}') &&
sed -i "s/127.0.0.1/$SERVER_IP/" /home/%s/.kube/config
`, cfg.Domain, master.SSHUser, master.SSHUser, master.SSHUser, master.SSHUser, master.SSHUser, master.SSHUser, master.SSHUser)
		// Execute the command on the remote server
		err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, command)
		if err != nil {
			return fmt.Errorf("Erro Remote-Server: %v", err)
		}

		fmt.Printf("  %s (%s, %s)\n", master.IP, master.SSHUser, master.SSHPass)
	}

	return nil
}
