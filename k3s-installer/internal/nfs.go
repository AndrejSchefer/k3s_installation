package internal

import (
	"fmt"
	"log"
	"strings"

	"igneos.cloud/kubernetes/k3s-installer/config"
	"igneos.cloud/kubernetes/k3s-installer/remote"
)

// MountNFS lädt die Konfiguration und richtet NFS auf allen Workern ein
func MountNFS() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfiguration: %v", err)
	}

	exportPath := "/home/kubernetes/ic-k3s-nfs"
	//exportCIDR := fmt.Sprintf("%s/24", cfg.Workers[0].IP)

	for _, worker := range cfg.Workers {
		fmt.Printf("[INFO] Konfiguriere NFS-Server auf %s (Export: %s)\n", worker.IP, exportPath)

		// Shell-Skript vorbereiten
		script := fmt.Sprintf(`
		echo '[INFO] Installiere nfs-kernel-server, falls nicht vorhanden'
		if ! dpkg -s nfs-kernel-server >/dev/null 2>&1; then
		apt-get update && apt-get install -y nfs-kernel-server
		else
		echo '[INFO] nfs-kernel-server ist bereits installiert'
		fi

		echo '[INFO] Erstelle Exportverzeichnis: %[1]s'
		mkdir -p '%[1]s'
		chown nobody:nogroup '%[1]s'
		chmod 777 '%[1]s'

		echo '[INFO] Prüfe /etc/exports auf vorhandene Einträge'
		if grep -qs '%[1]s' /etc/exports; then
		echo '[INFO] Export ist bereits in /etc/exports eingetragen'
		else
		echo '%[1]s %[2]s(rw,sync,no_subtree_check,no_root_squash)' >> /etc/exports
		echo '[SUCCESS] Export zu /etc/exports hinzugefügt'
		fi

		echo '----------------------------------------------------------------'
		echo '[INFO] Lade NFS-Exports neu'
		exportfs -ra
		exportfs -v

		echo '[SUCCESS] NFS-Server auf %[3]s ist bereit'
    `, exportPath, worker.IP, worker.IP)

		// Befehl mit Passwort vorbereiten
		fullCommand := fmt.Sprintf("echo '%s' | sudo -S bash -c \"%s\"",
			worker.SSHPass, escapeForDoubleQuotes(script))

		err := remote.RemoteExec(worker.SSHUser, worker.SSHPass, worker.IP, fullCommand)
		if err != nil {
			log.Printf("[FEHLER] NFS-Export auf %s fehlgeschlagen: %v\n", worker.IP, err)
		} else {
			fmt.Printf("[OK] NFS-Export auf %s erfolgreich eingerichtet\n", worker.IP)
		}
	}
}

// escapeForDoubleQuotes escaped doppelte Anführungszeichen für bash -c
func escapeForDoubleQuotes(input string) string {
	return strings.ReplaceAll(input, `"`, `\"`)
}
