package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

func createCRDs() error {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	master := cfg.Masters[0]

	namespace := "monitoring"

	script := fmt.Sprintf(`echo '%[1]s' | sudo -S bash -c '
		kubectl create namespace %[2]s --dry-run=client -o yaml | kubectl apply -f -
	'`, master.SSHPass, namespace)

	log.Printf("[INFO] Creating registry Secret on %s in namespace %s…", master.IP, namespace)
	if err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, script); err != nil {
		return fmt.Errorf("error creating registry Secret on %s: %w", master.IP, err)
	}

	log.Printf("[OK] Registry Secret successfully created/updated in namespace %s on %s", namespace, master.IP)

	// Step 1: Apply Prometheus CRDs
	utils.PrintSectionHeader("Installing Prometheus CRDs", "[INFO]", utils.ColorBlue, false)

	localCRDDir := "internal/templates/monitoring/" + cfg.K3sVersion + "/monitoring-crds-offline"
	files, err := os.ReadDir(localCRDDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to read CRD directory: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".yaml" {
			continue
		}
		templatePath := filepath.Join(localCRDDir, file.Name())
		remotePath := file.Name()

		utils.PrintSectionHeader(fmt.Sprintf("Applying CRD: %s", file.Name()), "[INFO]", utils.ColorBlue, false)

		if err := ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, templatePath, remotePath, nil); err != nil {
			log.Fatalf("[ERROR] Failed to apply CRD '%s': %v", file.Name(), err)
		}
	}

	// Wait for CRDs to be registered by the API server
	time.Sleep(10 * time.Second)
	return nil
}

func InstallMonitoring() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("[ERROR] Failed to load configuration: %v", err)
	}
	createCRDs()
	master := cfg.Masters[0]

	// Step 3: Apply monitoring resource templates
	steps := []struct {
		name       string
		template   string
		remotePath string
		vars       map[string]string
		active     bool
	}{
		{
			name:       "Prometheus Operator",
			template:   "internal/templates/monitoring/02-prometheus-operator.yaml",
			remotePath: "02-prometheus-operator.yaml",
			active:     true,
		},
		{
			name:       "Prometheus Instance",
			template:   "internal/templates/monitoring/03-prometheus.yaml",
			remotePath: "03-prometheus.yaml",
			active:     true,
		},
		{
			name:       "Grafana Deployment",
			template:   "internal/templates/monitoring/05-grafana.yaml",
			remotePath: "05-grafana.yaml",
			active:     true,
		},
		{
			name:       "ServiceMonitors",
			template:   "internal/templates/monitoring/04-servicemonitors.yaml",
			remotePath: "04-servicemonitors.yaml",
			active:     true,
		},
		{
			name:       "Services (Grafana & Prometheus)",
			template:   "internal/templates/monitoring/06-services.yaml",
			remotePath: "06-services.yaml",
			active:     true,
		},
		{
			name:       "ClusterRole (Grafana & Prometheus)",
			template:   "internal/templates/monitoring/07-cluster-role.yaml",
			remotePath: "07-cluster-role.yaml",
			active:     true,
		},
		{
			name:       "ClusterRoleBinding (Grafana & Prometheus)",
			template:   "internal/templates/monitoring/08-cluster-role-binding.yaml",
			remotePath: "08-cluster-role-binding.yaml",
			active:     true,
		},
		{
			name:       "Ingress (Grafana)",
			template:   "internal/templates/monitoring/09-grafana-ingress-local.yaml",
			remotePath: "09-grafana-ingress-local.yaml",
			active:     true,
		},
	}

	for _, step := range steps {
		if step.active {
			utils.PrintSectionHeader(fmt.Sprintf("Applying %s", step.name), "[INFO]", utils.ColorBlue, false)
			if err := ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
				log.Fatalf("[ERROR] Step '%s' failed: %v", step.name, err)
			}
		}
	}

	// TODO: Add a check to see if the monitoring components are already installed
	time.Sleep(20 * time.Second)
	// Step 2: Patch ServiceAccount in the Deployment before applying monitoring components
	patchCmd := fmt.Sprintf(`echo '%[1]s' | sudo -S kubectl patch deploy -n monitoring prometheus-operator --patch '{"spec": {"template": {"spec": {"serviceAccountName": "prometheus-operator"}}}}'`, master.SSHPass)
	if err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, patchCmd); err != nil {
		log.Fatalf("[ERROR] Failed to patch ServiceAccount in prometheus-operator deployment: %v", err)
	}
	utils.PrintSectionHeader("Monitoring stack successfully installed", "[SUCCESS]", utils.ColorGreen, false)
}
