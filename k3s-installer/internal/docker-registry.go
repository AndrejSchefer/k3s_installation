package internal

import (
	"fmt"
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

func createRegistrySecretWithHtpasswd() error {
	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	master := cfg.Masters[0]

	// Define namespace (from config or hardcoded)
	namespace := "ic-docker-registry"

	// Retrieve username and password for registry access
	user := cfg.DockerRegistry.User
	pass := cfg.DockerRegistry.Pass

	// Define the path on the remote host where the .htpasswd file will be stored
	htpasswdPath := "/tmp/.htpasswd"

	// Bash script to create htpasswd file and Kubernetes Secret
	script := fmt.Sprintf(`echo '%[1]s' | sudo -S bash -c '

	kubectl create namespace %[5]s

	# Ensure the directory exists	
	mkdir -p $(dirname %[2]s) && chmod 700 $(dirname %[2]s)

	# Create htpasswd file with new entry (-b uses username:password non-interactively)
	htpasswd -b -c %[2]s %[3]s %[4]s

	# Create or update Kubernetes Secret
	kubectl create secret generic registry-credentials \
	--from-file=htpasswd=%[2]s \
	-n %[5]s \
	--dry-run=client -o yaml | kubectl apply -f -


	# Remove htpasswd file
	rm -f %[2]s
	'`, master.SSHPass, htpasswdPath, user, pass, namespace)

	log.Printf("[INFO] Creating registry Secret on %s in namespace %s…", master.IP, namespace)
	if err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, script); err != nil {
		return fmt.Errorf("error creating registry Secret on %s: %w", master.IP, err)
	}
	log.Printf("[OK] Registry Secret successfully created/updated in namespace %s on %s", namespace, master.IP)
	return nil
}

func InstallDockerRegistry() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	createRegistrySecretWithHtpasswd()

	master := cfg.Masters[0]

	steps := []struct {
		name       string
		template   string
		remotePath string
		vars       map[string]string
	}{
		/*{
			name:       "Namespace",
			template:   "internal/templates/docker-registry/namespace.yaml",
			remotePath: "/tmp/docker-registry-namespace.yaml",
		},*/
		{
			name:       "Deployment",
			template:   "internal/templates/docker-registry/deployment.yaml",
			remotePath: "docker-registry-deployment.yaml",
		},
		{
			name:       "Service",
			template:   "internal/templates/docker-registry/service.yaml",
			remotePath: "docker-registry-service.yaml",
		},
		{
			name:       "ingress",
			template:   "internal/templates/docker-registry/ingress.yaml",
			remotePath: "docker-registry-ingress.yaml",
			vars: map[string]string{
				"{{DOCKER_REGISTRY_URL}}": cfg.DockerRegistry.URL,
			},
		},
		{
			name:       "Domain tls certificate",
			template:   "internal/templates/docker-registry/domain-tls-certificate.yaml",
			remotePath: "docker-registry-domain-tls-certificate.yaml",
			vars: map[string]string{
				"{{DOCKER_REGISTRY_URL}}": cfg.DockerRegistry.URL,
			},
		},
		{
			name:       "pvc-localhost",
			template:   "internal/templates/docker-registry/pvc-localhost.yaml",
			remotePath: "docker-registry-pvc-localhost.yaml",
			vars: map[string]string{
				"{{PVC_Storage_Capacity}}": cfg.DockerRegistry.PVCStorageCapacity,
			},
		},
	}

	for _, step := range steps {
		log.Printf("[STEP] Applying %s...", step.name)
		if err := ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
			log.Fatalf("%s step failed: %v", step.name, err)
		}
	}

	log.Println("[SUCCESS] NFS Subdir External Provisioner successfully installed")
}
