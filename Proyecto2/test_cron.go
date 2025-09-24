package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if err := SetupCronjob(); err != nil {
		log.Printf("Error en SetupCronjob: %v", err)
	}
}

func SetupCronjob() error {
	scriptPath := "/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/create_containers.sh"
	crontabEntry := "* * * * * /bin/bash " + scriptPath + " >> /tmp/container_cron.log 2>&1"
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "no crontab") {
			// Set to empty if no crontab message
			output = []byte("")
		} else {
			log.Printf("⚠️  Error leyendo crontab: %v, Output: %s", err, string(output))
			return fmt.Errorf("error leyendo crontab: %w", err)
		}
	}

	currentCrontab := strings.TrimSpace(string(output))
	if strings.Contains(currentCrontab, scriptPath) {
		log.Println("ℹ️  Cronjob ya existe, no agregando")
		return nil
	}

	newCrontab := currentCrontab
	if newCrontab != "" {
		newCrontab += "\n"
	}
	newCrontab += crontabEntry + "\n"

	tempFile := "/tmp/crontab_temp"
	if err := os.WriteFile(tempFile, []byte(newCrontab), 0644); err != nil {
		log.Printf("⚠️  Error escribiendo temp file: %v", err)
		return err
	}
	defer os.Remove(tempFile)
	cmd = exec.Command("crontab", tempFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("⚠️  Error actualizando crontab: %v, Output: %s", err, string(output))
		return fmt.Errorf("error actualizando crontab: %w", err)
	}
	log.Println("✅ Cronjob configurado: crea contenedores cada minuto")
	return nil
}

// RemoveCronjob elimina el cronjob
func RemoveCronjob() error {
	scriptPath := "/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/create_containers.sh"
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(string(output), "no crontab") {
			log.Println("ℹ️  No hay crontab, nada que remover")
			return nil
		}
		return fmt.Errorf("error leyendo crontab: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var newCrontab []string
	found := false

	for _, line := range lines {
		if strings.Contains(line, scriptPath) {
			found = true
			continue
		}
		newCrontab = append(newCrontab, line)
	}

	if found {
		tempFile := "/tmp/crontab_temp"
		if err := os.WriteFile(tempFile, []byte(strings.Join(newCrontab, "\n")), 0644); err != nil {
			return err
		}
		defer os.Remove(tempFile)

		cmd = exec.Command("crontab", tempFile)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error actualizando crontab: %w", err)
		}
		log.Println("✅ Cronjob eliminado")
	}

	return nil
}
