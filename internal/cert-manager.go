package internal

import (
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

// InstallCertManager installs cert-manager and applies the ClusterIssuer.
func InstallCertManager() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("[ERROR] Failed to load configuration: %v", err)
	}

	master := cfg.Masters[0]

	utils.PrintSectionHeader(
		"Installing cert-manager and ClusterIssuer...", "[INFO]", utils.ColorBlue, true,
	)

	if err := ApplyRemoteYAML(
		master.IP,
		master.SSHUser,
		master.SSHPass,
		"internal/templates/cert-manager/cert-manager.yaml",
		"cert-manager.yaml",
		nil,
	); err != nil {
		log.Fatalf("[ERROR] Failed to apply cert-manager: %v", err)
	}

	// Step 2: Wait for the cert-manager webhook deployment to become ready
	utils.PrintSectionHeader(
		"Waiting for cert-manager webhook to become ready...", "[INFO]", utils.ColorBlue, false,
	)
	waitCmd := "kubectl -n cert-manager rollout status deploy/cert-manager-webhook --timeout=90s"
	if err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, waitCmd); err != nil {
		log.Fatalf("[ERROR] cert-manager webhook not ready: %v", err)
	}

	// Step 3: Apply ClusterIssuer with templated values
	utils.PrintSectionHeader(
		"Applying ClusterIssuer...", "[INFO]", utils.ColorBlue, false,
	)
	vars := map[string]string{
		"{{EMAIL}}":               cfg.Email,
		"{{CLUSTER_ISSUER_NAME}}": cfg.ClusterIssuerName,
	}

	if err := ApplyRemoteYAML(
		master.IP,
		master.SSHUser,
		master.SSHPass,
		"internal/templates/cert-manager/clusterIssuer.yaml",
		"clusterIssuer.yaml",
		vars,
	); err != nil {
		log.Fatalf("[ERROR] Failed to apply ClusterIssuer: %v", err)
	}

	utils.PrintSectionHeader(
		"cert-manager and ClusterIssuer successfully installed.", "[SUCCESS]", utils.ColorGreen, false,
	)
}
