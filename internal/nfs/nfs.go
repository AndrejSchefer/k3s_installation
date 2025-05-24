package nfs

import (
	"fmt"
	"log"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
	"igneos.cloud/kubernetes/k3s-installer/utils"
)

// MountNFS configures the NFS export on the remote server.
func MountNFS() {
	//--------------------------------------------------------------------
	// 1) Load config
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf(utils.ColorRed+"[ERROR] Failed to load configuration: %v"+utils.ColorReset, err)
	}

	nfsIP := cfg.NFS.NFS_Server
	nfsUser := cfg.NFS.NFS_User
	nfsPass := cfg.NFS.NFS_Pass
	exportRoot := cfg.NFS.NFSRootPath
	nfsCIDR := cfg.NFS.NetworkCIDR

	utils.PrintSectionHeader(
		fmt.Sprintf("[INFO] Configuring NFS export on server %s (export root: %s)\n", nfsIP, exportRoot),
		"[INFO]", utils.ColorBlue, false,
	)

	//--------------------------------------------------------------------
	// 2) Render embedded shell template
	script, err := BuildScript(map[string]string{
		"Root":  exportRoot,
		"CIDR":  nfsCIDR,
		"Blue":  utils.ColorBlue,
		"Green": utils.ColorGreen,
		"Reset": utils.ColorReset,
	})
	if err != nil {
		log.Fatalf(utils.ColorRed+"[ERROR] template render failed: %v"+utils.ColorReset, err)
	}

	//--------------------------------------------------------------------
	// 3) Execute remotely via SSH
	// sudo -S reads password from stdin; script is embedded via bash -c
	fullCmd := fmt.Sprintf(
		`echo '%s' | sudo -S bash -c "%s"`,
		EscapeForDoubleQuotes(nfsPass),
		EscapeForDoubleQuotes(script),
	)

	if err := remote.RemoteExec(nfsUser, nfsPass, nfsIP, fullCmd); err != nil {
		log.Printf(utils.ColorRed+"[ERROR] Failed to configure NFS export on %s: %v"+utils.ColorReset, nfsIP, err)
	} else {
		utils.PrintSectionHeader(
			fmt.Sprintf("NFS export successfully configured on %s\n", nfsIP),
			"[OK]", utils.ColorGreen, true,
		)
	}
}

// escapeForDoubleQuotes makes a string safe for inclusion in bash -c "..."
func EscapeForDoubleQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
