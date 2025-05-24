package nfs

import (
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

func InstallNFSSubdirExternalProvisioner() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	master := cfg.Masters[0]

	utils.PrintSectionHeader(
		"Installing NFS Subdir External Provisioner...", "[INFO]", utils.ColorBlue, true,
	)
	steps := []struct {
		name       string
		template   string
		remotePath string
		vars       map[string]string
	}{
		{
			name:       "Namespace",
			template:   "internal/templates/nfs/nfs-namespace.yaml",
			remotePath: "nfs-namespace.yaml",
		},
		{
			name:       "RBAC",
			template:   "internal/templates/nfs/nfs-rbac.yaml",
			remotePath: "nfs-rbac.yaml",
		},
		{
			name:       "StorageClass",
			template:   "internal/templates/nfs/nfs-storageclass.yaml",
			remotePath: "nfs-storageclass.yaml",
		},
		{ // Deployment
			name:       "Deployment",
			template:   "internal/templates/nfs/nfs-deployment.yaml",
			remotePath: "nfs-deployment.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}":    cfg.NFS.NFS_Server,
				"{{NFS_ROOT_PATH}}": cfg.NFS.NFSRootPath,
			},
		},

		// PV für die Docker-Registry
		{
			name:       "PV for Docker Registry",
			template:   "internal/templates/nfs/pv-nfs-docker-registry-data.yaml",
			remotePath: "pv-nfs-docker-registry-data.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}":   cfg.NFS.NFS_Server,
				"{{NFS_EXPORT}}":   cfg.NFS.ExportDockerRegistry, // ← getrennt
				"{{NFS_CAPACITY}}": cfg.NFS.Capacity,
			},
		},
		// PV für Grafana
		{
			name:       "PV for Grafana",
			template:   "internal/templates/nfs/pv-nfs-grafana-data.yaml",
			remotePath: "pv-nfs-grafana-data.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}":   cfg.NFS.NFS_Server,
				"{{NFS_EXPORT}}":   cfg.NFS.ExportGrafana,
				"{{NFS_CAPACITY}}": cfg.NFS.Capacity,
			},
		},
	}

	for _, step := range steps {
		utils.PrintSectionHeader(
			"Applying "+step.name+"...", "[INFO]", utils.ColorBlue, false,
		)
		if err := utils.ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
			log.Fatalf("%s step failed: %v", step.name, err)
		}
	}

	log.Println("[SUCCESS] NFS Subdir External Provisioner successfully installed")
	utils.PrintSectionHeader(
		"NFS Subdir External Provisioner successfully installed", "[SUCCESS]", utils.ColorGreen, false,
	)
}
