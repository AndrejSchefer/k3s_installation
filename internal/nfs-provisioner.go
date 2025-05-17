package internal

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
		{
			name:       "Deployment",
			template:   "internal/templates/nfs/nfs-deployment.yaml",
			remotePath: "nfs-deployment.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}": cfg.NFS.NFS_Server,
				"{{NFS_EXPORT}}": cfg.NFS.Export,
			},
		},
		{
			name:       "PV",
			template:   "internal/templates/nfs/pv.yaml",
			remotePath: "pv.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}":   cfg.NFS.NFS_Server,
				"{{NFS_EXPORT}}":   cfg.NFS.Export,
				"{{NFS_CAPACITY}}": cfg.NFS.Capacity,
			},
		},
	}

	for _, step := range steps {
		utils.PrintSectionHeader(
			"Applying "+step.name+"...", "[INFO]", utils.ColorBlue, false,
		)
		if err := ApplyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
			log.Fatalf("%s step failed: %v", step.name, err)
		}
	}

	log.Println("[SUCCESS] NFS Subdir External Provisioner successfully installed")
	utils.PrintSectionHeader(
		"NFS Subdir External Provisioner successfully installed", "[SUCCESS]", utils.ColorGreen, false,
	)
}
