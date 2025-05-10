package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"igneos.cloud/kubernetes/k3s-installer/internal"
)

var rootCmd = &cobra.Command{
	Use:   "igneos.cloud.cli",
	Short: "Igneos.Cloud K3s Cluster Management CLI",
	Run: func(cmd *cobra.Command, args []string) {
		showMenu()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func showMenu() {
	// Define the options for the interactive menu
	options := []string{
		"Install k3s Master",
		"Install k3s Worker",
		"Create a NFS mount on worker",
		"Create a NFS PV",
		"Install Cert Manager",
		"Install full K3s-Cluster",
		"Install NFS Provisioner",
		"Uninstall k3s FULL Cluster",
		"Exit",
	}

	var choice string
	prompt := &survey.Select{
		Message: "Please select an action:",
		Options: options,
	}

	// Display the menu
	err := survey.AskOne(prompt, &choice)
	if err != nil {
		fmt.Println("Prompt failed:", err)
		return
	}

	// Switch based on user choice
	switch choice {
	case "Install k3s Master":
		internal.InstallK3sMaster()
	case "Install k3s Worker":
		internal.InstallK3sWorker()
	case "Create a NFS mount on worker":
		internal.MountNFS()
	case "Create a NFS PV":
		createNFSPV()
	case "Install Cert Manager":
		internal.InstallCertManager()
	case "Install full K3s-Cluster":
		installFullCluster()
	case "Install NFS Provisioner":
		internal.InstallNFSSubdirExternalProvisioner()
	case "Uninstall k3s FULL Cluster":
		uninstallFullCluster()
	case "Exit":
		fmt.Println("Exiting.")
		return
	}
}

// Dummy implementation placeholders
func createNFSPV() {
	fmt.Println("Creating NFS Persistent Volume...")
}

func installCertManager() {
	fmt.Println("Installing Cert Manager...")
}

func installFullCluster() {
	fmt.Println("Installing full K3s Cluster...")
}

func uninstallFullCluster() {
	fmt.Println("Uninstalling full K3s Cluster...")
}
