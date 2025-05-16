package config

// NodeConfig represents one node (master or worker)
type NodeConfig struct {
	IP      string `json:"ip"`
	SSHUser string `json:"ssh_user"`
	SSHPass string `json:"ssh_pass"`
}

// NFSConfig represents NFS settings
type NFSConfig struct {
	NFS_Server string `json:"nfs_server"`
	NFS_User   string `json:"nfs_user"`
	NFS_Pass   string `json:"nfs_pass"`
	Server     string `json:"server"`
	Export     string `json:"export"`
	Capacity   string `json:"capacity"`
}

type DockerRegistry struct {
	URL                string `json:"url"`
	PVCStorageCapacity string `json:"pvc_storagy_capacity"`
	User               string `json:"user"`
	Pass               string `json:"pass"`
}

// AppConfig represents the entire configuration
type AppConfig struct {
	Masters           []NodeConfig   `json:"masters"`
	Workers           []NodeConfig   `json:"workers"`
	K3sTokenFile      string         `json:"k3s_token_file"`
	NFS               NFSConfig      `json:"nfs"`
	DockerRegistry    DockerRegistry `json:"docker_registry"`
	Email             string         `json:"email"`
	Domain            string         `json:"domain"`
	ClusterIssuerName string         `json:"cluster_issuer_name"`
}
