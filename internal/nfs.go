package internal

import (
	"fmt"
	"log"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

func MountNFS() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf(utils.ColorRed+"[ERROR] Failed to load configuration: %v"+utils.ColorReset, err)
	}

	nfsIP := cfg.NFS.NFS_Server
	nfsUser := cfg.NFS.NFS_User
	nfsPass := cfg.NFS.NFS_Pass
	exportPath := cfg.NFS.Export
	// New: define the client network CIDR (e.g. "192.168.179.0/24")
	nfsCIDR := cfg.NFS.NetworkCIDR

	//fmt.Printf(utils.ColorBlue+"[INFO] Configuring NFS export on server %s (Export path: %s)\n"+ColotReset, nfsIP, exportPath)
	utils.PrintSectionHeader(fmt.Sprintf("[INFO] Configuring NFS export on server %s (Export path: %s)\n", nfsIP, exportPath), "[INFO]", utils.ColorBlue, false)
	// Shell script to set up NFS export
	script := fmt.Sprintf(`
    echo '%[5]s[INFO]%[6]s Installing nfs-kernel-server if not already present'
    if ! dpkg -s nfs-kernel-server >/dev/null 2>&1; then apt-get update && apt-get install -y nfs-kernel-server
    else echo '%[5]s[INFO]%[6]s nfs-kernel-server is already installed'; fi

    echo '%[5]s[INFO]%[6]s Creating export directory: %[1]s'
    mkdir -p '%[1]s' && chown nobody:nogroup '%[1]s' && chmod 0777 '%[1]s'

    echo '%[5]s[INFO]%[6]s Checking /etc/exports for existing entries'
    if grep -qs '%[1]s' /etc/exports; then
      echo '%[5]s[INFO]%[6]s Export already exists in /etc/exports'
    else
      # Export with rw, sync, no_subtree_check, no_root_squash for the entire k3s node subnet
      echo '%[1]s %[4]s(rw,sync,no_subtree_check,no_root_squash)' >> /etc/exports
      echo '%[3]s[SUCCESS]%[6]s Export added to /etc/exports'
    fi

    echo '%[5]s[INFO]%[6]s Reloading NFS exports'
    exportfs -ra && exportfs -v

    echo '%[3]s[SUCCESS]%[6]s NFS export is ready for clients in %[4]s'
    `,
		exportPath,       // %[1]s => export directory
		nfsIP,            // %[2]s => server IP (unused in exports line)
		utils.ColorGreen, // %[3]s => SUCCESS utils.Color
		nfsCIDR,          // %[4]s => client network CIDR
		utils.ColorBlue,  // %[5]s => INFO utils.Color
		utils.ColorReset, // %[6]s => reset utils.Color
	)

	// Prepare remote command
	fullCommand := fmt.Sprintf("echo '%s' | sudo -S bash -c \"%s\"", nfsPass, escapeForDoubleQuotes(script))

	// Execute remotely
	err = remote.RemoteExec(nfsUser, nfsPass, nfsIP, fullCommand)
	if err != nil {
		log.Printf(utils.ColorRed + "[ERROR] Failed to configure NFS export on %s: %v" + utils.ColorReset)
	} else {
		utils.PrintSectionHeader(fmt.Sprintf("NFS export successfully configured on %s\n", nfsIP), "[OK]", utils.ColorGreen, true)
	}
}

// escapeForDoubleQuotes escapes all double quotes for bash -c execution
func escapeForDoubleQuotes(input string) string {
	return strings.ReplaceAll(input, `"`, `\"`)
}
