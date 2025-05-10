package internal

import (
	"log"

	"igneos.cloud/kubernetes/k3s-installer/config"
)

func InstallNFSSubdirExternalProvisioner() {
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
			name:       "Namespace",
			template:   "internal/templates/nfs/nfs-namespace.yaml",
			remotePath: "/tmp/nfs-namespace.yaml",
		},
		{
			name:       "RBAC",
			template:   "internal/templates/nfs/nfs-rbac.yaml",
			remotePath: "/tmp/nfs-rbac.yaml",
		},
		{
			name:       "StorageClass",
			template:   "internal/templates/nfs/nfs-storageclass.yaml",
			remotePath: "/tmp/nfs-storageclass.yaml",
		},
		{
			name:       "Deployment",
			template:   "internal/templates/nfs/nfs-deployment.yaml",
			remotePath: "/tmp/nfs-deployment.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}": cfg.NFS.NFS_Server,
				"{{NFS_EXPORT}}": cfg.NFS.Export,
			},
		},
		{
			name:       "PV",
			template:   "internal/templates/nfs/pv.yaml",
			remotePath: "/tmp/pv.yaml",
			vars: map[string]string{
				"{{NFS_SERVER}}":   cfg.NFS.NFS_Server,
				"{{NFS_EXPORT}}":   cfg.NFS.Export,
				"{{NFS_CAPACITY}}": cfg.NFS.Capacity,
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
