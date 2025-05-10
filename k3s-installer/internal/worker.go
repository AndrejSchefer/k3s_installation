package internal

import (
	"fmt"
	"log"
	"os"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

// InstallK3sWorker installiert den K3s-Agent auf einem Worker-Knoten via SSH
func InstallK3sWorker() error {
	log.Println("[START] Installing worker nodes...")

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
	log.Printf("[INFO] K3s Token erfolgreich geladen aus %s\n", cfg.K3sTokenFile)

	for _, worker := range cfg.Workers {
		user := worker.SSHUser
		password := worker.SSHPass
		host := worker.IP

		fmt.Printf("[INFO] Installing k3s agent on worker node %s\n", host)

		// Secure and robust installation command with set -e
		installCmd := fmt.Sprintf(`
		echo '%s' | sudo -S bash -c '
		set -e
		curl -sfL https://get.k3s.io | K3S_URL="https://%s:6443" K3S_TOKEN="%s" sh -s - agent
		'`, password, cfg.Masters[0].IP, token)

		if err := remote.RemoteExec(user, password, host, installCmd); err != nil {
			return fmt.Errorf("Fehler bei der Installation auf Worker %s: %v", host, err)
		}

		fmt.Printf("[INFO] Verifiziere k3s-agent auf %s...\n", host)

		checkCmd := "systemctl is-active --quiet k3s-agent"
		err = remote.RemoteExec(user, password, host, checkCmd)
		if err == nil {
			fmt.Printf("[OK] k3s agent auf %s ist aktiv und bereit!\n", host)
		} else {
			return fmt.Errorf("❌ k3s agent auf %s ist NICHT aktiv: %v", host, err)
		}
	}

	log.Println("[SUCCESS] Alle Worker-Knoten erfolgreich installiert.")
	return nil
}
