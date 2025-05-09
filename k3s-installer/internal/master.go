package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

func InstallK3sMaster() error {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfiguration: %v", err)
	}

	fmt.Println("Master Nodes:")
	for _, master := range cfg.Masters {
		user := master.SSHUser
		password := master.SSHPass
		domain := cfg.Domain

		// Befehl mit sudo -S und Passwortübergabe
		command := fmt.Sprintf(`echo '%s' | sudo -S bash -c '
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode=644 --secrets-encryption --tls-san=%s" sh -s - server &&
mkdir -p /home/%s/.kube &&
cp /etc/rancher/k3s/k3s.yaml /home/%s/.kube/config &&
chown %s:%s /home/%s/.kube/config &&
chmod 600 /home/%s/.kube/config &&
SERVER_IP=$(hostname -I | awk "{print $1}") &&
sed -i "s/127\\.0\\.0\\.1/$SERVER_IP/" /home/%s/.kube/config
'`, password, domain, user, user, user, user, user, user, user)
		// Execute the command on the remote server
		err := remote.RemoteExec(master.SSHUser, master.SSHPass, master.IP, command)
		if err != nil {
			return fmt.Errorf("Erro Remote-Server: %v", err)
		}

		fmt.Printf("  %s (%s, %s)\n", master.IP, master.SSHUser, master.SSHPass)
	}

	fetchK3sToken(cfg.Masters[0].IP, cfg.Masters[0].SSHUser, cfg.Masters[0].SSHPass, cfg.K3sTokenFile)
	fetchKubeconfigLocal(cfg.Masters[0].IP, cfg.Masters[0].SSHUser, cfg.Masters[0].SSHPass)
	return nil
}

func fetchK3sToken(masterHost, user, password, tokenFile string) error {
	fmt.Printf("[INFO] Lese node-token vom Master (%s)...\n", masterHost)

	output, err := remote.RemoteExecOutput(user, password, masterHost, "cat /var/lib/rancher/k3s/server/node-token")
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen des node-token: %v", err)
	}

	tokenRaw := strings.TrimSpace(output)
	fmt.Printf("[DEBUG] Antwort vom Master:\n%s\n", tokenRaw)

	var token string
	for _, line := range strings.Split(tokenRaw, "\n") {
		if strings.HasPrefix(line, "K") {
			token = line
			break
		}
	}

	if token == "" {
		return fmt.Errorf("Kein gültiger Token gefunden in:\n%s", tokenRaw)
	}

	err = os.WriteFile(tokenFile, []byte(token+"\n"), 0600)
	if err != nil {
		return fmt.Errorf("Fehler beim Schreiben der Token-Datei: %v", err)
	}

	fmt.Printf("[OK] Token erfolgreich gespeichert unter %s\n", tokenFile)
	return nil
}

func fetchKubeconfigLocal(masterHost, userName, password string) error {
	fmt.Println("[INFO] Hole kubeconfig vom Master...")

	// SSH Verbindung
	config := &ssh.ClientConfig{
		User:            userName,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", masterHost+":22", config)
	if err != nil {
		return fmt.Errorf("SSH-Fehler: %v", err)
	}
	defer client.Close()

	// SFTP-Client starten
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP-Fehler: %v", err)
	}
	defer sftpClient.Close()

	// Zielpfad bestimmen (~/.kube/config)
	usr, _ := user.Current()
	kubeDir := usr.HomeDir + "/.kube"
	os.MkdirAll(kubeDir, 0700)
	dst := kubeDir + "/config"

	// Quelldatei öffnen
	srcFile, err := sftpClient.Open("/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return fmt.Errorf("Remote-Datei nicht gefunden: %v", err)
	}
	defer srcFile.Close()

	// Lokale Datei schreiben
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen lokaler Datei: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("Fehler beim Kopieren: %v", err)
	}

	// IP in Datei ersetzen
	content, err := os.ReadFile(dst)
	if err != nil {
		return err
	}

	newContent := strings.ReplaceAll(string(content), "127.0.0.1", masterHost)
	err = os.WriteFile(dst, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("Fehler beim Schreiben der geänderten config: %v", err)
	}

	fmt.Printf("[OK] kubeconfig gespeichert unter: %s\n", dst)
	return nil
}
