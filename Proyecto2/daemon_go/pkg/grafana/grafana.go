package grafana

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

func StartGrafana() error {
	// Verificar si ya está corriendo
	cmd := exec.Command("docker", "ps", "--filter", "name=grafana", "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		log.Println("ℹ️  Grafana ya está corriendo, deteniendo para reiniciar...")
		StopGrafana() // Llama a StopGrafana para eliminar el existente
	}

	// Ruta absoluta a la DB
	dbPath, err := filepath.Abs("/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/daemon_go/cmd/daemon/container_data.db")
	if err != nil {
		return fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}
	log.Printf("📁 Montando DB desde: %s", dbPath)

	// Iniciar Grafana con volumen
	cmd = exec.Command("docker", "run", "-d",
		"--name", "grafana",
		"-p", "3000:3000",
		"-e", "GF_SECURITY_ADMIN_USER=admin",
		"-e", "GF_SECURITY_ADMIN_PASSWORD=admin",
		"-v", fmt.Sprintf("%s:/data/container_data.db:ro", dbPath),
		"-v", "grafana_storage:/var/lib/grafana",
		"grafana/grafana:latest",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error iniciando Grafana: %w. Output: %s", err, string(output))
	}

	log.Println("✅ Grafana iniciado en http://localhost:3000")
	log.Println("   Usuario: admin, Contraseña: admin")
	log.Println("   DB montada en /data/container_data.db")
	return nil
}

func StopGrafana() error {
	cmd := exec.Command("docker", "stop", "grafana")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("⚠️  Error deteniendo Grafana: %v. Output: %s", err, string(output))
	}
	cmd = exec.Command("docker", "rm", "-f", "grafana")
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("⚠️  Error removiendo Grafana: %v. Output: %s", err, string(output))
	}
	log.Println("🛑 Grafana detenido")
	return nil
}
