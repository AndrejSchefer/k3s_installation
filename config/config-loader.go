package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadConfig reads the JSON config file, decodes it and validates all fields.
func LoadConfig(filename string) (*AppConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var cfg AppConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not decode JSON: %w", err)
	}

	// Run validation on the decoded config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks that no required field is left empty.
func (c *AppConfig) Validate() error {
	// Check master nodes
	for i, m := range c.Masters {
		if m.IP == "" {
			return fmt.Errorf("masters[%d].ip must not be empty", i)
		}
		if m.SSHUser == "" {
			return fmt.Errorf("masters[%d].ssh_user must not be empty", i)
		}
		if m.SSHPass == "" {
			return fmt.Errorf("masters[%d].ssh_pass must not be empty", i)
		}
	}

	// Check worker nodes
	for i, w := range c.Workers {
		if w.IP == "" {
			return fmt.Errorf("workers[%d].ip must not be empty", i)
		}
		if w.SSHUser == "" {
			return fmt.Errorf("workers[%d].ssh_user must not be empty", i)
		}
		if w.SSHPass == "" {
			return fmt.Errorf("workers[%d].ssh_pass must not be empty", i)
		}
	}

	// Check registry settings
	if c.DockerRegistry.URL == "" {
		return fmt.Errorf("docker_registry.url must not be empty")
	}
	if c.DockerRegistry.PVCStorageCapacity == "" {
		return fmt.Errorf("docker_registry.pvc_storage_capacity must not be empty")
	}
	if c.DockerRegistry.Pass == "" {
		return fmt.Errorf("docker_registry.pass must not be empty")
	}
	if c.DockerRegistry.User == "" {
		return fmt.Errorf("docker_registry.user must not be empty")
	}

	// Check K3s token file
	if c.K3sTokenFile == "" {
		return fmt.Errorf("k3s_token_file must not be empty")
	}

	// Check NFS settings
	if c.NFS.Server == "" {
		return fmt.Errorf("nfs.nfs_server must not be empty")
	}
	if c.NFS.NFS_User == "" {
		return fmt.Errorf("nfs.nfs_user must not be empty")
	}
	if c.NFS.NFS_Pass == "" {
		return fmt.Errorf("nfs.nfs_pass must not be empty")
	}

	if c.NFS.ExportDockerRegistry == "" {
		return fmt.Errorf("nfs.export-docker-registry must not be empty")
	}

	if c.NFS.ExportGrafana == "" {
		return fmt.Errorf("nfs.export-grafana must not be empty")
	}

	if c.NFS.Capacity == "" {
		return fmt.Errorf("nfs.capacity must not be empty")
	}

	// Check global settings
	if c.Email == "" {
		return fmt.Errorf("email must not be empty")
	}
	if c.Domain == "" {
		return fmt.Errorf("domain must not be empty")
	}
	if c.ClusterIssuerName == "" {
		return fmt.Errorf("cluster_issuer_name must not be empty")
	}

	return nil
}
