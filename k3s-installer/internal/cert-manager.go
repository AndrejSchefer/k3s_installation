package internal

import (
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
)

func InstallCertManager() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	master := cfg.Masters[0]

	steps := []struct {
		name       string
		template   string
		remotePath string
		vars       map[string]string
	}{
		{
			name:       "Cert Manager",
			template:   "internal/templates/cert-manager/cert-manager.yaml",
			remotePath: "/tmp/cert-manager.yaml",
		},
	}

	for _, step := range steps {
		log.Printf("[STEP] Applying %s...", step.name)
		if err := ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
			log.Fatalf("%s step failed: %v", step.name, err)
		}
	}

	log.Println("[SUCCESS] Cert Manager successfully installed")
}
