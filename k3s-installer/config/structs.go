package config

// NodeConfig represents one node (master or worker)
type NodeConfig struct {
	IP      string `json:"ip"`
	SSHUser string `json:"ssh_user"`
	SSHPass string `json:"ssh_pass"`
}

// NFSConfig represents NFS settings
type NFSConfig struct {
	Server   string `json:"server"`
	Export   string `json:"export"`
	Capacity string `json:"capacity"`
}

// AppConfig represents the entire configuration
type AppConfig struct {
	Masters           []NodeConfig `json:"masters"`
	Workers           []NodeConfig `json:"workers"`
	K3sTokenFile      string       `json:"k3s_token_file"`
	NFS               NFSConfig    `json:"nfs"`
	Email             string       `json:"email"`
	Domain            string       `json:"domain"`
	ClusterIssuerName string       `json:"cluster_issuer_name"`
}
