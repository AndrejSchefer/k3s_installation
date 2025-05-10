package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
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
	}

	for _, step := range steps {
		log.Printf("[STEP] Applying %s...", step.name)
		if err := applyRemoteYAML(master.IP, master.SSHUser, master.SSHPass, step.template, step.remotePath, step.vars); err != nil {
			log.Fatalf("%s step failed: %v", step.name, err)
		}
	}

	log.Println("[SUCCESS] NFS Subdir External Provisioner successfully installed")
}

// applyRemoteYAML uploads a YAML file to a remote host and applies it with kubectl
func applyRemoteYAML(host, user, password, localPath, remotePath string, replacements map[string]string) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local YAML file: %w", err)
	}

	fmt.Println("----------------------------------------------------------")
	log.Println(host)
	fmt.Println("----------------------------------------------------------")

	yaml := string(content)
	for key, value := range replacements {
		yaml = strings.ReplaceAll(yaml, key, value)
	}

	tmpFile := "temp-upload.yaml"
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		return fmt.Errorf("failed to write temporary YAML: %w", err)
	}
	defer os.Remove(tmpFile)

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", host+":22", sshConfig)
	log.Println(conn)

	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("SFTP setup failed: %w", err)
	}
	defer sftpClient.Close()

	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}

	srcFile, err := os.Open(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer srcFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy YAML to remote: %w", err)
	}
	fmt.Println("[INFO] YAML uploaded to master:", remotePath)
	fmt.Println("----------------------------------------------------------")
	log.Println(remotePath)
	fmt.Println("----------------------------------------------------------")

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("SSH session creation failed: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	//	applyCmd := fmt.Sprintf("echo '%s' | sudo -S bash -c 'kubectl apply -f %s && rm -f %s'", password, remotePath, remotePath)
	applyCmd := fmt.Sprintf("echo '%s' | sudo -S bash -c 'kubectl apply -f %s && echo %s'", password, remotePath, remotePath)
	if err := session.Run(applyCmd); err != nil {
		return fmt.Errorf("failed to apply YAML remotely: %w", err)
	}

	fmt.Println("[OK] Applied YAML on master")
	return nil
}
