package internal

import (
	"fmt"
	"log"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// MountNFS reads the configuration and configures NFS export on the specified NFS server
func MountNFS() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf(colorRed+"[ERROR] Failed to load configuration: %v"+colorReset, err)
	}

	nfsIP := cfg.NFS.NFS_Server
	nfsUser := cfg.NFS.NFS_User
	nfsPass := cfg.NFS.NFS_Pass
	exportPath := cfg.NFS.Export

	fmt.Printf(colorBlue+"[INFO] Configuring NFS export on server %s (Export path: %s)\n"+colorReset, nfsIP, exportPath)

	// Shell script to set up NFS export
	script := fmt.Sprintf(`
	echo '%[4]s[INFO]%[5]s Installing nfs-kernel-server if not already present'
	if ! dpkg -s nfs-kernel-server >/dev/null 2>&1; then apt-get update && apt-get install -y nfs-kernel-server
	else echo '%[4]s[INFO]%[5]s nfs-kernel-server is already installed'; fi

	echo '%[4]s[INFO]%[5]s Creating export directory: %[1]s'
	mkdir -p '%[1]s' && chown nobody:nogroup '%[1]s' && chmod 777 '%[1]s'

	echo '%[4]s[INFO]%[5]s Checking /etc/exports for existing entries'
	if grep -qs '%[1]s' /etc/exports; then
	echo '%[4]s[INFO]%[5]s Export already exists in /etc/exports'
	else
	echo '%[1]s %[2]s(rw,sync,no_subtree_check,no_root_squash)' >> /etc/exports
	echo '%[3]s[SUCCESS]%[5]s Export added to /etc/exports'
	fi

	echo '%[4]s[INFO]%[5]s Reloading NFS exports'
	exportfs -ra && exportfs -v

	echo '%[3]s[SUCCESS]%[5]s NFS export is ready on %[2]s'
	`, exportPath, nfsIP, colorGreen, colorBlue, colorReset)

	// Prepare remote command
	fullCommand := fmt.Sprintf("echo '%s' | sudo -S bash -c \"%s\"",
		nfsPass, escapeForDoubleQuotes(script))

	// Execute remotely
	err = remote.RemoteExec(nfsUser, nfsPass, nfsIP, fullCommand)
	if err != nil {
		log.Printf(colorRed+"[ERROR] Failed to configure NFS export on %s: %v"+colorReset, nfsIP, err)
	} else {
		fmt.Printf(colorGreen+"[OK] NFS export successfully configured on %s\n"+colorReset, nfsIP)
	}
}

// escapeForDoubleQuotes escapes all double quotes for bash -c execution
func escapeForDoubleQuotes(input string) string {
	return strings.ReplaceAll(input, `"`, `\"`)
}
