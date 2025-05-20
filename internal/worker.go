package internal

import (
	"fmt"
	"os"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

// InstallK3sWorker installiert den K3s-Agent auf einem Worker-Knoten via SSH
func InstallK3sWorker() error {
	utils.PrintSectionHeader("Installing K3s worker nodes...", "[INFO]", utils.ColorBlue, true)

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		return fmt.Errorf("Fehler beim Laden der Konfiguration: %v", err)
	}

	// Check if token file exists
	if _, err := os.Stat(cfg.K3sTokenFile); os.IsNotExist(err) {
		return fmt.Errorf("Token-Datei nicht gefunden: %s", cfg.K3sTokenFile)
	}

	// Read token file
	tokenBytes, err := os.ReadFile(cfg.K3sTokenFile)
	if err != nil {
		return fmt.Errorf("Fehler beim Lesen der Token-Datei: %v", err)
	}

	token := strings.TrimSpace(string(tokenBytes))

	msg := fmt.Sprintf("K3s Token is loading successfully %s\n", cfg.K3sTokenFile)
	utils.PrintSectionHeader(msg, "[INFO]", utils.ColorBlue, true)

	k3sVersion := cfg.K3sVersion

	for _, worker := range cfg.Workers {
		user := worker.SSHUser
		password := worker.SSHPass
		host := worker.IP

		msg := fmt.Sprintf("[INFO] Installing k3s agent on worker node %s\n", host)
		utils.PrintSectionHeader(msg, "[INFO]", utils.ColorBlue, false)

		// Secure and robust installation command with set -e
		installCmd := fmt.Sprintf(`
		echo '%s' | sudo -S bash -c '
		set -e
		curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="%s" \ K3S_URL="https://%s:6443" K3S_TOKEN="%s" sh -s - agent
		'`, password, k3sVersion, cfg.Masters[0].IP, token)

		if err := remote.RemoteExec(user, password, host, installCmd); err != nil {
			return fmt.Errorf("Fehler bei der Installation auf Worker %s: %v", host, err)
		}

		utils.PrintSectionHeader(fmt.Sprintf("Verify k3s-agent on %s...\n", host), "[INFO]", utils.ColorBlue, true)

		checkCmd := "systemctl is-active --quiet k3s-agent"
		err = remote.RemoteExec(user, password, host, checkCmd)
		if err == nil {
			utils.PrintSectionHeader(fmt.Sprintf("k3s agent om %s is active and ready!\n", host), "[SUCCESS]", utils.ColorGreen, false)

		} else {
			return fmt.Errorf("❌ k3s agent auf %s ist NICHT aktiv: %v", host, err)
		}
	}

	utils.PrintSectionHeader("K3s worker installation complete.", "[SUCCESS]", utils.ColorGreen, true)
	return nil
}
