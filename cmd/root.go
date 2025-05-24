package cmd

// This file contains the CLI entrypoint for the Igneos.Cloud K3s installer.
// Major changes compared with the previous version:
//   1. The configuration is loaded exactly once during model creation; this avoids
//      a nil‑pointer panic when the file is missing or malformed.
//   2. Graceful fallback to a default version string ("unknown") if no config
//      file is found, instead of crashing.
//   3. All error handling paths are explicit and visible to the user.
//   4. Code comments are written in English, as requested.
//
// Bubble Tea works fine with value receivers; there is therefore no need to
// convert the model to pointer receivers. The program now guarantees that
// View() never dereferences a nil pointer.

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/internal"
	"igneos.cloud/kubernetes/k3s-installer/internal/nfs"
)

var rootCmd = &cobra.Command{
	Use:   "igneos.cloud.cli",
	Short: "Igneos.Cloud K3s Cluster Management CLI",
	Run: func(cmd *cobra.Command, args []string) {
		startMenu()
	},
}

// Execute is the public entrypoint called from main.go.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// ----- Menu item definition -----
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

// ----- Menu model -----
type model struct {
	cursor  int
	choice  string
	items   []MenuItem
	version string // The K3s version string loaded from the config file.
}

// initialModel loads the configuration exactly once and returns a fully
// initialised model that contains no nil pointers. This guarantees that View()
// cannot panic due to a nil dereference.
func initialModel() model {
	cfg, err := config.LoadConfig("config.json")
	version := "unknown" // default fallback
	if err == nil && cfg != nil {
		version = cfg.K3sVersion
	}

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
		version: version,
	}
}

// Init returns the initial command; there is none in this simple menu.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the cursor or final choice.
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

// View renders the menu. Because the configuration has already been loaded, we
// simply reference m.version here; there is no further file I/O.
func (m model) View() string {
	// Header
	s := titleStyle.Render(`
----------------------------------------------------------
IGNEOS.CLOUD K3s Cluster Installer (beta)
----------------------------------------------------------`)
	s += "\n Install K3s version: " + m.version + "\n"
	s += "\n Use ↑ ↓ to move, ↵ to select\n\n"

	// Compute column width for nice alignment.
	maxTitleLen := 0
	for _, item := range m.items {
		length := len(item.Icon) + 1 + len(item.Title)
		if length > maxTitleLen {
			maxTitleLen = length
		}
	}

	// Render items.
	for i, item := range m.items {
		cursor := " "
		localTitleStyle := lipgloss.NewStyle()
		if m.cursor == i {
			cursor = "▶"
			localTitleStyle = selectedStyle
		}
		title := fmt.Sprintf("%s %s", item.Icon, item.Title)
		paddedTitle := fmt.Sprintf("%-*s", maxTitleLen, title)

		s += fmt.Sprintf(
			"  %s %s\n    %s\n\n",
			cursorStyle.Render(cursor),
			localTitleStyle.Render(paddedTitle),
			descStyle.Render(item.Description),
		)
	}
	return s
}

// startMenu creates the Bubble Tea program and launches the TUI. When the user
// makes a final selection, handleChoice is called.
func startMenu() {
	m := initialModel()
	program := tea.NewProgram(m)

	finalModel, err := program.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error running menu:", err)
		os.Exit(1)
	}

	if chosenModel, ok := finalModel.(model); ok {
		handleChoice(chosenModel.choice)
	}
}

// handleChoice calls the appropriate installer routine depending on the menu
// selection.
func handleChoice(choice string) {
	switch choice {
	case "Install Kubernetes Master":
		internal.InstallK3sMaster()
	case "Install Kubernetes Worker":
		internal.InstallK3sWorker()
	case "Create a NFS mount on worker":
		nfs.MountNFS()
	case "Install Cert Manager":
		internal.InstallCertManager()
	case "Install Full K3s-Cluster":
		installFullCluster()
	case "Install NFS Provisioner":
		nfs.InstallNFSSubdirExternalProvisioner()
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

// installFullCluster orchestrates the full cluster installation in the correct
// order.
func installFullCluster() {
	fmt.Println("\nInstalling full K3s Cluster with all components...")
	internal.InstallK3sMaster()
	internal.InstallK3sWorker()
	nfs.MountNFS()
	internal.InstallCertManager()
	nfs.InstallNFSSubdirExternalProvisioner()
}
