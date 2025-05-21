package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"igneos.cloud/kubernetes/k3s-installer/config"
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

// ----- Menüeintrag -----
type MenuItem struct {
	Icon        string
	Title       string
	Description string
}

// ----- Styling -----
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	descStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// ----- Menümodell -----
type model struct {
	cursor int
	choice string
	items  []MenuItem
}

func initialModel() model {
	return model{
		items: []MenuItem{
			{"🚀", "Install Full K3s-Cluster", "Installs Master, Worker, NFS, CertManager, Registry, and Monitoring stack"},
			{"🧠", "Install Kubernetes Master", "Installs a K3s master node"},
			{"👷", "Install Kubernetes Worker", "Adds a worker node to the cluster"},
			{"📦", "Create a NFS mount on worker", "Mounts NFS storage on the worker node"},
			{"🔒", "Install Cert Manager", "Installs cert-manager for TLS certificate management"},
			{"🗂️", "Install NFS Provisioner", "Installs NFS Subdir External Provisioner for dynamic storage"},
			{"📦", "Install Docker Registry", "Installs a private Docker registry"},
			{"📊", "Install Monitoring With Prometheus and Grafana", "Installs Prometheus and Grafana for cluster monitoring"},
			{"💣", "Uninstall Kubernetes FULL Cluster", "Completely removes all cluster components"},
			{"🚪", "Exit", "Exit the application"},
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
			m.choice = m.items[m.cursor].Title
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		fmt.Errorf("Error loading config: %v", err)
	}

	s := titleStyle.Render(`
----------------------------------------------------------
IGNEOS.CLOUD K3s Cluster Installer (beta)
----------------------------------------------------------
`)
	s += "\n Install K3s version: " + cfg.K3sVersion + "\n"
	s += "\n Use ↑ ↓ to move, ↵ to select\n\n"

	maxTitleLen := 0
	for _, item := range m.items {
		length := len(item.Icon) + 1 + len(item.Title)
		if length > maxTitleLen {
			maxTitleLen = length
		}
	}

	for i, item := range m.items {
		cursor := " "
		titleStyle := lipgloss.NewStyle()
		if m.cursor == i {
			cursor = "▶"
			titleStyle = selectedStyle
		}
		title := fmt.Sprintf("%s %s", item.Icon, item.Title)
		paddedTitle := fmt.Sprintf("%-*s", maxTitleLen, title)

		s += fmt.Sprintf(
			"  %s %s\n    %s\n\n",
			cursorStyle.Render(cursor),
			titleStyle.Render(paddedTitle),
			descStyle.Render(item.Description),
		)
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
	case "Install Kubernetes Master":
		internal.InstallK3sMaster()
	case "Install Kubernetes Worker":
		internal.InstallK3sWorker()
	case "Create a NFS mount on worker":
		internal.MountNFS()
	case "Install Cert Manager":
		internal.InstallCertManager()
	case "Install Full K3s-Cluster":
		installFullCluster()
	case "Install NFS Provisioner":
		internal.InstallNFSSubdirExternalProvisioner()
	case "Install Docker Registry":
		internal.InstallDockerRegistry()
	case "Install Monitoring With Prometheus and Grafana":
		internal.InstallMonitoring()
	case "Uninstall Kubernetes FULL Cluster":
		internal.UninstallK3sCluster()
	case "Exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	}
}

func installFullCluster() {
	fmt.Println("\nInstalling full K3s Cluster with all components...")
	internal.InstallK3sMaster()
	internal.InstallK3sWorker()
	internal.MountNFS()
	internal.InstallCertManager()
	internal.InstallNFSSubdirExternalProvisioner()
}
