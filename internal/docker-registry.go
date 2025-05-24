package internal

import (
	"fmt"
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

// createRegistrySecretWithHtpasswd creates an htpasswd file and Kubernetes Secret on the master node
func createRegistrySecretWithHtpasswd() error {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	master := cfg.Masters[0]

	namespace := "ic-docker-registry"
	user := cfg.DockerRegistry.User
	pass := cfg.DockerRegistry.Pass
	htpasswdPath := "/home/kubernetes/.htpasswd"

	script := fmt.Sprintf(`echo '%[1]s' | sudo -S bash -c '
kubectl create namespace %[5]s --dry-run=client -o yaml | kubectl apply -f -
mkdir -p $(dirname %[2]s) && chmod 700 $(dirname %[2]s)
htpasswd -b -c %[2]s %[3]s %[4]s

kubectl create secret generic registry-credentials \
  --from-file=htpasswd=%[2]s \
  -n %[5]s \
  --dry-run=client -o yaml > /tmp/registry-credentials-secret.yaml

kubectl apply -f /tmp/registry-credentials-secret.yaml -n %[5]s
rm /tmp/registry-credentials-secret.yaml
'`, master.SSHPass, htpasswdPath, user, pass, namespace)

	log.Printf("[INFO] Creating registry Secret on %s in namespace %s…", master.IP, namespace)
	if err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, script); err != nil {
		return fmt.Errorf("error creating registry Secret on %s: %w", master.IP, err)
	}

	log.Printf("[OK] Registry Secret successfully created/updated in namespace %s on %s", namespace, master.IP)
	return nil
}

// InstallDockerRegistry deploys the Docker registry based on config
func InstallDockerRegistry() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("[ERROR] Failed to load configuration: %v", err)
	}
	if err := createRegistrySecretWithHtpasswd(); err != nil {
		log.Fatalf("[ERROR] Failed to create registry Secret: %v", err)
	}

	master := cfg.Masters[0]
	local := cfg.DockerRegistry.Local

	steps := []struct {
		name       string
		template   string
		remotePath string
		vars       map[string]string
		active     bool
	}{
		{
			name:       "PVC (Localhost)",
			template:   "internal/templates/docker-registry/pvc-localhost.yaml",
			remotePath: "docker-registry-pvc-localhost.yaml",
			vars: map[string]string{
				"{{PVC_Storage_Capacity}}": cfg.DockerRegistry.PVCStorageCapacity,
			},
			active: true,
		},
		{
			name:       "Config (no TLS)",
			template:   "internal/templates/docker-registry/config_without_tls.yaml",
			remotePath: "config_without_tls.yaml",
			active:     local,
		},
		{
			name:       "Service (no TLS)",
			template:   "internal/templates/docker-registry/service_without_tls.yaml",
			remotePath: "docker-registry-service.yaml",
			active:     local,
		},
		{
			name:       "Ingress (no TLS)",
			template:   "internal/templates/docker-registry/ingress_without_tls.yaml",
			remotePath: "docker-registry-ingress.yaml",
			vars: map[string]string{
				"{{DOCKER_REGISTRY_URL}}": cfg.DockerRegistry.URL,
			},
			active: local,
		},
		{
			name:       "Deployment (no TLS)",
			template:   "internal/templates/docker-registry/deployment_without_tls.yaml",
			remotePath: "deployment.yaml",
			active:     local,
		},
		{
			name:       "Service (TLS)",
			template:   "internal/templates/docker-registry/service.yaml",
			remotePath: "docker-registry-service.yaml",
			active:     !local,
		},
		{
			name:       "Ingress (TLS)",
			template:   "internal/templates/docker-registry/ingress.yaml",
			remotePath: "docker-registry-ingress.yaml",
			vars: map[string]string{
				"{{DOCKER_REGISTRY_URL}}": cfg.DockerRegistry.URL,
			},
			active: !local,
		},
		{
			name:       "Domain TLS Certificate",
			template:   "internal/templates/docker-registry/domain-tls-certificate.yaml",
			remotePath: "docker-registry-domain-tls-certificate.yaml",
			vars: map[string]string{
				"{{DOCKER_REGISTRY_URL}}": cfg.DockerRegistry.URL,
			},
			active: !local,
		},
		{
			name:       "Deployment (TLS)",
			template:   "internal/templates/docker-registry/deployment.yaml",
			remotePath: "deployment.yaml",
			active:     !local,
		},
	}

	for _, step := range steps {
		if step.active {
			utils.PrintSectionHeader(fmt.Sprintf("Applying %s", step.name), "[INFO]", utils.ColorBlue, false)
			if err := utils.ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
				log.Fatalf("[ERROR] Step '%s' failed: %v", step.name, err)
			}
		}
	}

	utils.PrintSectionHeader("Docker Registry successfully installed", "[SUCCESS]", utils.ColorGreen, false)

	// Access info
	fmt.Println("\nYou can access the Docker Registry at:")
	if local {
		fmt.Println("→ docker login http://registry.local:80")
	} else {
		fmt.Printf("→ docker login %s\n", cfg.DockerRegistry.URL)
	}
	fmt.Printf("→ Username: %s\n", cfg.DockerRegistry.User)
	fmt.Printf("→ Password: %s\n", cfg.DockerRegistry.Pass)
}
