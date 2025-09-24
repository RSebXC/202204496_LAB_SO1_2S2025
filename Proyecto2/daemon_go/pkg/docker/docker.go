package docker

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"container-manager/pkg/kernel"
)

type ContainerInfo struct {
	ID         string  `json:"ID"`    // "ID"
	Names      string  `json:"Names"` // "Names"
	Image      string  `json:"Image"`
	Status     string  `json:"Status"`
	Created    string  `json:"CreatedAt"` // Cambiado a string para evitar parsing error
	RSSKB      int64   `json:"rss_kb"`
	VSZKB      int64   `json:"vsz_kb"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	PID        int     `json:"pid"`
	RSSMB      float64 `json:"rss_mb"`
}

type Decision struct {
	Action      string `json:"action"` // "stop_and_remove", "create_high", "create_low"
	Target      string `json:"target"` // Container ID
	Reason      string `json:"reason"`
	ContainerID string `json:"container_id"`
}

func ListContainers(continfoPath, sysinfoPath string) ([]ContainerInfo, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️ Error ejecutando docker ps: %v, cayendo a legacy", err)
		return listContainersLegacy()
	}

	lines := strings.Split(string(output), "\n")
	var containers []ContainerInfo
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var info ContainerInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			log.Printf("⚠️ Error unmarshaling línea JSON: %v (línea: %s)", err, line)
			continue // Sigue con las demás, no falla todo
		}
		containers = append(containers, info)
	}

	if len(containers) == 0 {
		log.Println("⚠️ No se parsearon contenedores, cayendo a legacy")
		return listContainersLegacy()
	}

	// Ahora sí, procede con kernel data (esto debería ejecutarse ahora)
	kernelData, _, err := kernel.ReadProcData(continfoPath, sysinfoPath)
	if err != nil {
		log.Printf("⚠️  Error leyendo datos del kernel: %v", err)
		return containers, nil // Continúa sin métricas
	}

	pidMap := make(map[int]kernel.ProcInfo)
	for _, proc := range kernelData.Containers {
		pidMap[proc.PID] = proc
		//log.Printf("📋 PID del kernel: %d, Name: %s, RSSKB: %d", proc.PID, proc.Name, proc.RSSKB) // Tu log
	}

	for i := range containers {
		inspectCmd := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", containers[i].ID)
		pidOutput, err := inspectCmd.Output()
		if err == nil {
			pidStr := strings.TrimSpace(string(pidOutput))
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				log.Printf("⚠️ Error parseando PID para %s: %s (%v)", containers[i].Names, pidStr, err)
				continue
			}
			//log.Printf("📋 PID de contenedor %s: %d", containers[i].Names, pid)
			if proc, exists := pidMap[pid]; exists {
				containers[i].RSSKB = proc.RSSKB
				containers[i].VSZKB = proc.VSZKB
				containers[i].CPUPercent = float64(proc.CPUPercent)
				containers[i].MemPercent = float64(proc.MemPercent)
				containers[i].PID = pid
				containers[i].RSSMB = float64(proc.RSSKB) / 1024
				log.Printf("✅ Mapeado contenedor %s (PID %d) con kernel: RSSKB=%d, CPU=%d%%, Mem=%d%%", containers[i].Names, pid, proc.RSSKB, proc.CPUPercent, proc.MemPercent)

				// Fallback si kernel da 0 en CPU/Mem%
				if containers[i].CPUPercent == 0 {
					containers[i].CPUPercent = float64(getContainerCPU(containers[i].ID))
					//log.Printf("ℹ️ Fallback CPU para %s: %.2f%% (desde docker stats)", containers[i].Names, containers[i].CPUPercent)
				}
				if containers[i].MemPercent == 0 {
					containers[i].MemPercent = float64(getContainerMemPercent(containers[i].ID))
					//log.Printf("ℹ️ Fallback MemPercent para %s: %.2f%% (desde docker stats)", containers[i].Names, containers[i].MemPercent)
				}
			} else {
				log.Printf("⚠️  No se encontró PID %d del contenedor %s en kernel data, usando full fallback", pid, containers[i].Names)
				// Full fallback si no mapea
				containers[i].CPUPercent = float64(getContainerCPU(containers[i].ID))
				containers[i].MemPercent = float64(getContainerMemPercent(containers[i].ID))
				containers[i].RSSKB = getContainerRSS(containers[i].ID)
				containers[i].VSZKB = getContainerVSZ(containers[i].ID)
				containers[i].PID = pid
				containers[i].RSSMB = float64(containers[i].RSSKB) / 1024
			}
		} else {
			log.Printf("⚠️  Error obteniendo PID de %s: %v", containers[i].Names, err)
		}
	}
	return containers, nil
}

// listContainersLegacy método de respaldo usando docker ps
func listContainersLegacy() ([]ContainerInfo, error) {
	cmd := exec.Command("docker", "ps", "--format", "table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error listando contenedores: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var containers []ContainerInfo

	for _, line := range lines[1:] { // Saltar header
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			id := strings.TrimPrefix(parts[0], "sha256:")
			containers = append(containers, ContainerInfo{
				ID:     id[:12], // Primeros 12 chars del ID
				Names:  parts[1],
				Image:  parts[2],
				Status: parts[3],
			})
		}
	}

	return containers, nil
}

// ExecuteDecisions ejecuta las decisiones sobre contenedores
func ExecuteDecisions(decisions []Decision) error {
	for _, decision := range decisions {
		switch decision.Action {
		case "stop_and_remove":
			if err := stopAndRemoveContainer(decision.ContainerID); err != nil {
				log.Printf("⚠️  Error eliminando contenedor %s: %v", decision.ContainerID, err)
			} else {
				log.Printf("🗑️  Contenedor %s eliminado: %s", decision.ContainerID, decision.Reason)
			}
		case "keep":
			log.Printf("✅ Contenedor %s mantenido: %s", decision.ContainerID, decision.Reason)
		case "create_high":
			cmd := exec.Command("docker", "run", "-d", "--name", decision.Target, "ram_hog")
			if err := cmd.Run(); err != nil {
				log.Printf("⚠️  Error creando contenedor high %s: %v", decision.Target, err)
			} else {
				log.Printf("✅ Contenedor high %s creado", decision.Target)
			}
		case "create_low":
			cmd := exec.Command("docker", "run", "-d", "--name", decision.Target, "low_consumption")
			if err := cmd.Run(); err != nil {
				log.Printf("⚠️  Error creando contenedor low %s: %v", decision.Target, err)
			} else {
				log.Printf("✅ Contenedor low %s creado", decision.Target)
			}
		default:
			log.Printf("⚠️  Acción desconocida para %s: %s", decision.ContainerID, decision.Action)
		}
	}
	return nil
}

// stopAndRemoveContainer detiene y elimina un contenedor
func stopAndRemoveContainer(containerID string) error {
	// Primero intentar detener
	cmd := exec.Command("docker", "stop", containerID)
	if err := cmd.Run(); err != nil {
		log.Printf("⚠️  No se pudo detener %s, forzando eliminación", containerID)
	}

	// Luego eliminar
	cmd = exec.Command("docker", "rm", "-f", containerID)
	return cmd.Run()
}

// getContainerRSS obtiene RSS de un contenedor (simplificado)
func getContainerRSS(containerID string) int64 {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️  Error obteniendo MemUsage para %s: %v", containerID, err)
		return 0
	}
	memStr := strings.TrimSpace(string(output))
	parts := strings.Split(memStr, "/")
	if len(parts) < 2 {
		return 0
	}
	used := parts[0]
	val, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(used, " ")[0]), 64)
	if err != nil {
		log.Printf("⚠️  Error parseando MemUsage para %s: %v", containerID, err)
		return 0
	}
	unit := strings.ToLower(strings.Split(used, " ")[1])
	switch unit {
	case "mib":
		return int64(val * 1024)
	case "gib":
		return int64(val * 1024 * 1024)
	case "kib":
		return int64(val)
	default:
		return 0
	}
}

func getContainerVSZ(containerID string) int64 {
	return getContainerRSS(containerID) * 5 // Aproximación, o usa kernel si disponible
}

func getContainerCPU(containerID string) int {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.CPUPerc}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️  Error obteniendo CPU para %s: %v", containerID, err)
		return 0
	}
	cpuStr := strings.TrimSuffix(strings.TrimSpace(string(output)), "%")
	//log.Printf("📊 CPUPerc para %s: %s", containerID, cpuStr) // Log después de trim para debug
	val, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		log.Printf("⚠️  Error parseando CPU para %s: %v (str: %s)", containerID, err, cpuStr)
		return 0
	}
	return int(math.Round(val)) // Redondea ej. 0.49 -> 0, 0.5 -> 1
}

func getContainerMemPercent(containerID string) int {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemPerc}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️  Error obteniendo MemPerc para %s: %v", containerID, err)
		return 0
	}
	memStr := strings.TrimSuffix(strings.TrimSpace(string(output)), "%")
	//log.Printf("📊 MemPerc para %s: %s", containerID, memStr) // Log después de trim
	val, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		log.Printf("⚠️  Error parseando MemPerc para %s: %v (str: %s)", containerID, err, memStr)
		return 0
	}
	return int(math.Round(val))
}

func SetupCronjob() error {
	scriptPath := "/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/create_containers.sh"
	crontabEntry := "*/2 * * * * /bin/bash " + scriptPath + " >> /tmp/container_cron.log 2>&1"
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

// SortContainersByRAM ordena contenedores por uso de RAM - EXPORTADA
func SortContainersByRAM(containers []ContainerInfo) []ContainerInfo {
	sorted := make([]ContainerInfo, len(containers))
	copy(sorted, containers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RSSKB > sorted[j].RSSKB
	})
	return sorted
}

func SortContainersByCPU(containers []ContainerInfo) []ContainerInfo {
	sorted := make([]ContainerInfo, len(containers))
	copy(sorted, containers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CPUPercent > sorted[j].CPUPercent
	})
	return sorted
}

// CountLowConsumption cuenta contenedores de bajo consumo - EXPORTADA
func CountLowConsumption(containers []ContainerInfo) int {
	count := 0
	for _, c := range containers {
		if c.Image == "low_consumption" || (c.MemPercent < 10 && c.CPUPercent < 10) {
			count++
		}
	}
	return count
}

// CountHighConsumption cuenta contenedores de alto consumo - EXPORTADA
func CountHighConsumption(containers []ContainerInfo) int {
	count := 0
	for _, c := range containers {
		if (c.Image == "ram_hog" || c.Image == "cpu_hog") ||
			(c.MemPercent > 50 || c.CPUPercent > 50) {
			count++
		}
	}
	return count
}

func getContainerInfo() ([]ContainerInfo, error) {
	data, err := os.ReadFile("/proc/continfo_so1_202204496")
	if err != nil {
		return nil, err
	}
	var kernelData kernel.KernelData
	if err := json.Unmarshal(data, &kernelData); err != nil {
		return nil, err
	}
	var containers []ContainerInfo
	for _, proc := range kernelData.Containers {
		containers = append(containers, ContainerInfo{
			ID:         fmt.Sprintf("cont_%d", proc.PID), // Generar ID único
			Names:      proc.Name,
			PID:        proc.PID,
			RSSKB:      proc.RSSKB,
			VSZKB:      proc.VSZKB,
			CPUPercent: float64(proc.CPUPercent),
			MemPercent: float64(proc.MemPercent),
		})
	}
	return containers, nil
}
