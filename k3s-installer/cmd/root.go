package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"igneos.cloud/kubernetes/k3s-installer/internal"
)

var rootCmd = &cobra.Command{
	Use:   "igneos.cloud.cli",
	Short: "Igneos.Cloud K3s Cluster Management CLI",
	Run: func(cmd *cobra.Command, args []string) {
		startMenu()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// ----- Styling -----
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
)

// ----- Model -----
type model struct {
	cursor int
	choice string
	items  []string
}

func initialModel() model {
	return model{
		items: []string{
			"Install k3s Master",
			"Install k3s Worker",
			"Create a NFS mount on worker",
			"Create a NFS PV",
			"Install Cert Manager",
			"Install full K3s-Cluster",
			"Install NFS Provisioner",
			"Install Docker Registry",
			"Uninstall k3s FULL Cluster",
			"Exit",
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.choice = m.items[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	s := titleStyle.Render("\n  IGNEOS.CLOUD K3s Cluster Installer\n")
	s += "  Use ↑ ↓ to move, ↵ to select\n\n"

	for i, item := range m.items {
		cursor := " "
		style := lipgloss.NewStyle()
		if m.cursor == i {
			cursor = "▶"
			style = selectedStyle
		}
		s += fmt.Sprintf("  %s %s\n", cursorStyle.Render(cursor), style.Render(item))
	}
	return s
}

// ----- Menüfunktion -----
func startMenu() {
	m := initialModel()
	program := tea.NewProgram(m)

	finalModel, err := program.Run()
	if err != nil {
		fmt.Println("Error running menu:", err)
		os.Exit(1)
	}

	if chosenModel, ok := finalModel.(model); ok {
		handleChoice(chosenModel.choice)
	}
}

func handleChoice(choice string) {
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
	case "Install Docker Registry":
		internal.InstallDockerRegistry()
	case "Uninstall k3s FULL Cluster":
		internal.UninstallK3sCluster()
	case "Exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	}
}

// Dummy Funktionen
func createNFSPV() {
	fmt.Println("Creating NFS Persistent Volume...")
}

func installFullCluster() {
	fmt.Println("\nInstalling full K3s Cluster with all components...\n")
	internal.InstallK3sMaster()
	internal.InstallK3sWorker()
	internal.MountNFS()
	internal.InstallCertManager()
	internal.InstallNFSSubdirExternalProvisioner()
	internal.InstallDockerRegistry()
}
