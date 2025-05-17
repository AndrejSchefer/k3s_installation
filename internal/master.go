package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

func InstallK3sMaster() error {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	utils.PrintSectionHeader("Installing K3s on master nodes...", "[INFO]", utils.ColorBlue, true)

	for _, master := range cfg.Masters {
		user := master.SSHUser
		pass := master.SSHPass
		ip := master.IP
		tlsDomain := cfg.Domain

		log.Printf("[STEP] Installing K3s on %s (%s@%s)\n", ip, user, ip)

		// Remote installation script with proper IP substitution
		cmd := fmt.Sprintf(`echo '%s' | sudo -S bash -c '
		if ! command -v htpasswd >/dev/null 2>&1; then
			apt-get update && \
			apt-get install -y apache2-utils
		fi && \

		curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode=644 --secrets-encryption --tls-san=%s" sh -s - server &&
		mkdir -p /home/%s/.kube &&
		cp /etc/rancher/k3s/k3s.yaml /home/%s/.kube/config &&
		chown %s:%s /home/%s/.kube/config &&
		chmod 600 /home/%s/.kube/config &&
		SERVER_IP=$(hostname -I | awk "{print \$1}") &&
		sed -i "s/127\\.0\\.0\\.1/$SERVER_IP/" /home/%s/.kube/config
		'`, pass, tlsDomain, user, user, user, user, user, user, user)

		if err := remote.RemoteExec(user, pass, ip, cmd); err != nil {
			return fmt.Errorf("failed to install K3s on %s: %w", ip, err)
		}

		utils.PrintSectionHeader(fmt.Sprintf("[SUCCESS] K3s installed successfully on %s\n", ip), "[SUCCESS]", utils.ColorGreen, false)
	}

	// Fetch the token and kubeconfig from the first master
	master := cfg.Masters[0]
	utils.PrintSectionHeader("[INFO] Fetching node token and kubeconfig...", "[INFO]", utils.ColorBlue, false)

	if err := fetchK3sToken(master.IP, master.SSHUser, master.SSHPass, cfg.K3sTokenFile); err != nil {
		return fmt.Errorf("failed to fetch node-token: %w", err)
	}

	if err := fetchKubeconfigLocal(master.IP, master.SSHUser, master.SSHPass); err != nil {
		return fmt.Errorf("failed to fetch kubeconfig: %w", err)
	}

	utils.PrintSectionHeader("[SUCCESS] K3s master installation complete.", "[SUCCESS]", utils.ColorGreen, true)
	return nil
}

func fetchK3sToken(masterHost, user, password, tokenFile string) error {
	msg := fmt.Sprintf("Read node-token of Master (%s)...", masterHost)
	utils.PrintSectionHeader(msg, "[INFO]", utils.ColorBlue, true)

	var output string
	var err error
	const maxRetries = 10

	cmd := fmt.Sprintf("echo '%s' | sudo -S cat /var/lib/rancher/k3s/server/node-token", password)
	for i := 0; i < maxRetries; i++ {
		output, err = remote.RemoteExecOutput(user, password, masterHost, cmd)
		if err == nil {
			break
		}
		msg := fmt.Sprintf("[WARN] Token is not available (Attempt %d/%d): %v", i+1, maxRetries, err)
		utils.PrintSectionHeader(msg, "[WARN]", utils.ColorYellow, false)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen des node-token: %v\nOutput: %s", err, output)
	}

	token := strings.TrimSpace(output)

	// Token-Datei schreiben
	result := strings.Split(token, ": ")
	err = os.WriteFile(tokenFile, []byte(result[1]+"\n"), 0600)
	if err != nil {
		return fmt.Errorf("Fehler beim Schreiben der Token-Datei (%s): %v", tokenFile, err)
	}

	utils.PrintSectionHeader("Token is save successfully.", "[SUCCESS]", utils.ColorGreen, false)
	return nil
}

func fetchKubeconfigLocal(masterHost, userName, password string) error {

	utils.PrintSectionHeader("Fetch kubeconfig of Master...", "[INFO]", utils.ColorBlue, true)
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
		return fmt.Errorf("Remote-file not found: %v", err)
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

	utils.PrintSectionHeader("kubeconfig is save successfully.", "[SUCCESS]", utils.ColorGreen, false)
	return nil
}
