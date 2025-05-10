package internal

import (
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

func InstallCertManager() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("[ERROR] Failed to load configuration: %v", err)
	}

	master := cfg.Masters[0]

	// Schritt 1: Installiere cert-manager YAML
	log.Println("[STEP] Applying Cert Manager...")
	if err := ApplyRemoteYAML(
		master.IP,
		master.SSHUser,
		master.SSHPass,
		"internal/templates/cert-manager/cert-manager.yaml",
		"/tmp/cert-manager.yaml",
		nil,
	); err != nil {
		log.Fatalf("[ERROR] Cert Manager step failed: %v", err)
	}

	// Schritt 2: Warte auf cert-manager-webhook, bevor der ClusterIssuer angewendet wird
	log.Println("[INFO] Waiting for cert-manager-webhook to become ready...")
	waitCmd := "kubectl -n cert-manager rollout status deploy/cert-manager-webhook --timeout=90s"
	if err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, waitCmd); err != nil {
		log.Fatalf("[ERROR] Webhook not ready: %v", err)
	}

	// Schritt 3: Wende ClusterIssuer an
	log.Println("[STEP] Applying ClusterIssuer...")
	vars := map[string]string{
		"{{EMAIL}}":               cfg.Email,
		"{{CLUSTER_ISSUER_NAME}}": cfg.ClusterIssuerName,
	}
	if err := ApplyRemoteYAML(
		master.IP,
		master.SSHUser,
		master.SSHPass,
		"internal/templates/cert-manager/clusterIssuer.yaml",
		"/tmp/clusterIssuer.yaml",
		vars,
	); err != nil {
		log.Fatalf("[ERROR] ClusterIssuer step failed: %v", err)
	}

	log.Println("[SUCCESS] Cert Manager successfully installed and ClusterIssuer configured.")
}
