package remote

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func RemoteExec(user, password, host string, command string) error {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return fmt.Errorf("SSH-Verbindung fehlgeschlagen: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("SSH-Session konnte nicht erstellt werden: %v", err)
	}
	defer session.Close()

	// Leite Stdout/Stderr durch
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Aktiviere ein Pseudo-Terminal
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("Fehler beim Setzen des Terminalzustands: %v", err)
	}
	defer term.Restore(fd, oldState)

	// Fange Signale ab und stelle den Terminalzustand wieder her
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		term.Restore(fd, oldState)
		os.Exit(0)
	}()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // Enable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return fmt.Errorf("PTY konnte nicht angefordert werden: %v", err)
	}

	log.Printf("[SSH] %s: Führe aus:\n%s\n", host, command)

	if err := session.Run(command); err != nil {
		return fmt.Errorf("Remote-Command fehlgeschlagen: %v", err)
	}

	log.Printf("[SSH] %s: Kommando erfolgreich abgeschlossen\n", host)
	return nil
}

func RemoteExecOutput(user, password, host, command string) (string, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return "", fmt.Errorf("SSH-Verbindung fehlgeschlagen: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("Session konnte nicht erstellt werden: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}
