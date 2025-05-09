package remote

import (
	"bytes"
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"
)

func RemoteExec(user, password, host string, command string) error {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Capture stdout + stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	log.Printf("Running command on %s:\n%s", host, command)
	if err := session.Run(command); err != nil {
		return fmt.Errorf("remote command failed: %v\nSTDERR: %s", err, stderrBuf.String())
	}

	log.Printf("Output:\n%s", stdoutBuf.String())
	return nil
}
