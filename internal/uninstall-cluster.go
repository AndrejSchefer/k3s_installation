package internal

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

// confirmAction prompts the user for confirmation before proceeding.
// Returns true if the user confirms with 'y' or 'yes'.
func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/N]: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return false
		}

		response := strings.ToLower(strings.TrimSpace(input))
		switch response {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Please respond with 'y' or 'n'.")
		}
	}
}

// UninstallK3sCluster uninstalls K3s from all nodes defined in the configuration.
func UninstallK3sCluster() error {
	if !confirmAction("Do you really want to uninstall the K3s cluster?") {
		fmt.Println("[ABORTED] Uninstallation canceled.")
		return nil
	}

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	for _, node := range append(cfg.Masters, cfg.Workers...) {
		fmt.Printf("[INFO] Starting uninstallation of K3s on %s...\n", node.IP)

		// Shell script to uninstall K3s
		script := `
		echo '[INFO] Stopping K3s services if active'
		systemctl stop k3s || true
		systemctl stop k3s-agent || true

		echo '[INFO] Executing uninstallation scripts if present'
		[ -f /usr/local/bin/k3s-uninstall.sh ] && /usr/local/bin/k3s-uninstall.sh
		[ -f /usr/local/bin/k3s-agent-uninstall.sh ] && /usr/local/bin/k3s-agent-uninstall.sh

		echo '[INFO] Removing remaining data directories'
		rm -rf /etc/rancher /var/lib/rancher /var/lib/kubelet /etc/cni /opt/cni /var/lib/containerd

		echo '[INFO] K3s services completely removed on $(hostname)'
		`

		// Construct the command to execute the script with sudo privileges
		fullCommand := fmt.Sprintf("echo '%s' | sudo -S bash -c \"%s\"", node.SSHPass, escapeForDoubleQuotes(script))

		// Execute the command on the remote host
		if err := remote.RemoteExec(node.SSHUser, node.SSHPass, node.IP, fullCommand); err != nil {
			return fmt.Errorf("error uninstalling K3s on %s: %w", node.IP, err)
		}

		log.Printf("[OK] K3s successfully uninstalled from %s.", node.IP)
	}

	return nil
}
