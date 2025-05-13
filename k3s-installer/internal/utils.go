package internal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func ApplyRemoteYAML(host, user, password, localPath, remotePath string, replacements map[string]string) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local YAML file: %w", err)
	}

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

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("SSH session creation failed: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	applyCmd := fmt.Sprintf("echo '%s' | sudo -S bash -c 'kubectl apply -f %s && rm -f %s'", password, remotePath, remotePath)
	//applyCmd := fmt.Sprintf("echo '%s' | sudo -S bash -c 'kubectl apply -f %s && echo %s'", password, remotePath, remotePath)
	if err := session.Run(applyCmd); err != nil {
		return fmt.Errorf("failed to apply YAML remotely: %w", err)
	}

	fmt.Println("[OK] Applied YAML on master")
	return nil
}
