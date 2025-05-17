package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
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

	// Determine the NFS export directory from config
	exportPath := cfg.NFS.Export

	for _, node := range append(cfg.Masters, cfg.Workers...) {
		utils.PrintSectionHeader(fmt.Sprintf("[INFO] Uninstalling K3s on %s...\n", node.IP), "[INFO]", utils.ColorBlue, false)
		// Build a shell script to run on the remote host
		script := fmt.Sprintf(`
# [INFO] Stop K3s services if active
systemctl stop k3s || true
systemctl stop k3s-agent || true

# [INFO] Run uninstall scripts if present
[ -f /usr/local/bin/k3s-uninstall.sh ] && /usr/local/bin/k3s-uninstall.sh
[ -f /usr/local/bin/k3s-agent-uninstall.sh ] && /usr/local/bin/k3s-agent-uninstall.sh

# [INFO] Remove remaining data directories
rm -rf /etc/rancher /var/lib/rancher /var/lib/kubelet /etc/cni /opt/cni /var/lib/containerd

# [INFO] Conditionally remove NFS local storage directory if it exists %s
if [ -d "%s" ]; then
    echo "[INFO] Removing %s"
	sudo  rm -rf "%s"
else
    echo "[INFO] %s not found, skipping"
fi

echo "[INFO] K3s services completely removed on $(hostname)"
`, exportPath, exportPath, exportPath, exportPath, exportPath)

		// Construct the command to execute the script with sudo privileges
		fullCommand := fmt.Sprintf("echo '%s' | sudo -S bash -c \"%s\"", node.SSHPass, escapeForDoubleQuotes(script))

		// Execute the command on the remote host
		if err := remote.RemoteExec(node.SSHUser, node.SSHPass, node.IP, fullCommand); err != nil {
			return fmt.Errorf("error uninstalling K3s on %s: %w", node.IP, err)
		}

		utils.PrintSectionHeader(fmt.Sprintf("[OK] K3s successfully uninstalled from %s.\n", node.IP), "[OK]", utils.ColorGreen, true)
	}

	return nil
}
