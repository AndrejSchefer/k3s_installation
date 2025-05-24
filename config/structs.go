package config

// NodeConfig represents one node (master or worker)
type NodeConfig struct {
	IP      string `json:"ip"`
	SSHUser string `json:"ssh_user"`
	SSHPass string `json:"ssh_pass"`
}

// NFSConfig represents NFS settings
type NFSConfig struct {
	NetworkCIDR          string `json:"network_CIDR"`
	NFS_Server           string `json:"nfs_server"`
	NFS_User             string `json:"nfs_user"`
	NFS_Pass             string `json:"nfs_pass"`
	Server               string `json:"server"`
	ExportDockerRegistry string `json:"export-docker-registry"`
	ExportGrafana        string `json:"export-grafana"`
	Capacity             string `json:"capacity"`
	NFSRootPath          string `json:"nfs_root_path"`
}

type DockerRegistry struct {
	URL                string `json:"url"`
	PVCStorageCapacity string `json:"pvc_storage_capacity"`
	User               string `json:"user"`
	Pass               string `json:"pass"`
	Local              bool   `json:"local"`
}

// AppConfig represents the entire configuration
type AppConfig struct {
	K3sVersion        string         `json:"k3s_version"`
	Masters           []NodeConfig   `json:"masters"`
	Workers           []NodeConfig   `json:"workers"`
	K3sTokenFile      string         `json:"k3s_token_file"`
	NFS               NFSConfig      `json:"nfs"`
	DockerRegistry    DockerRegistry `json:"docker_registry"`
	Email             string         `json:"email"`
	Domain            string         `json:"domain"`
	ClusterIssuerName string         `json:"cluster_issuer_name"`
}
