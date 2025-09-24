package main

import (
	"container-manager/pkg/db"
	"container-manager/pkg/docker"
	"container-manager/pkg/grafana"
	"container-manager/pkg/kernel"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	CARNET        = "202204496"
	CONTINFO_PATH = "/proc/continfo_so1_" + CARNET
	SYSINFO_PATH  = "/proc/sysinfo_so1_" + CARNET
	DB_PATH       = "./container_data.db"
	LOOP_INTERVAL = 20 * time.Second
)

func main() {
	CleanupExcessContainers()
	fmt.Println("🚀 Iniciando Container Manager Daemon...")
	// 1. Inicializar base de datos SQLite
	dbConn, err := db.InitDB(DB_PATH)
	if err != nil {
		log.Fatalf("Error inicializando SQLite: %v", err)
	}
	defer dbConn.Close()
	fmt.Println("✅ Base de datos SQLite inicializada")

	// 2. Iniciar Grafana
	if err := grafana.StartGrafana(); err != nil {
		log.Printf("⚠️  Error iniciando Grafana: %v", err)
	} else {
		fmt.Println("✅ Grafana iniciado")
	}

	// 3. Cargar módulos del kernel
	if err := kernel.LoadModules(); err != nil {
		log.Printf("⚠️  Error cargando módulos del kernel: %v", err)
	} else {
		fmt.Println("✅ Módulos del kernel cargados")
	}

	// 4. Configurar cronjob para crear contenedores
	if err := docker.SetupCronjob(); err != nil {
		log.Printf("⚠️  Error configurando cronjob: %v", err)
	} else {
		fmt.Println("✅ Cronjob configurado (crea contenedores cada minuto)")
	}

	// 5. Loop principal: análisis cada 20 segundos
	ticker := time.NewTicker(LOOP_INTERVAL)
	defer ticker.Stop()

	// Configurar graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n🛑 Recibida señal de terminación...")
		cancel()
	}()

	fmt.Printf("🔄 Iniciando loop de análisis cada %v...\n", LOOP_INTERVAL)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("🧹 Limpiando antes de salir...")
			cleanup(dbConn)
			return
		case <-ticker.C:
			if err := processCycle(dbConn); err != nil {
				log.Printf("❌ Error en ciclo de procesamiento: %v", err)
			}
		}
	}
}

func processCycle(dbConn *db.DBConn) error {
	fmt.Printf("\n[%s] 🔍 Iniciando ciclo de análisis...\n", time.Now().Format("15:04:05"))

	// Leer datos del kernel (solo contenedores y sistema)
	sysData, _, err := kernel.ReadProcData(CONTINFO_PATH, SYSINFO_PATH)
	if err != nil {
		return fmt.Errorf("error leyendo datos del kernel: %w", err)
	}

	fmt.Printf("📊 RAM Total: %dMB, Libre: %dMB, Usada: %dMB\n",
		sysData.TotalRAMMB, sysData.FreeRAMMB, sysData.UsedRAMMB)

	// Listar contenedores (filtra solo los mapeados)
	containers, err := docker.ListContainers(CONTINFO_PATH, SYSINFO_PATH)
	if err != nil {
		return fmt.Errorf("error listando contenedores: %w", err)
	}

	fmt.Printf("🐳 Contenedores activos: %d\n", len(containers))

	// Preparar métricas para DB (solo contenedores válidos)
	var containerMetrics []db.Metric
	for _, container := range containers {
		if container.PID > 0 && !strings.Contains(container.Names, "grafana") { // Ignora Grafana
			metric := db.Metric{
				ContainerID:    container.ID,
				ContainerPID:   container.PID,
				Image:          container.Image,
				Names:          container.Names,
				RSSKB:          container.RSSKB,
				VSZKB:          container.VSZKB,
				CPUPercent:     int(container.CPUPercent),
				MemPercent:     int(container.MemPercent),
				Action:         "running",
				DecisionReason: "monitoring",
				Timestamp:      time.Now(),
			}
			containerMetrics = append(containerMetrics, metric)
		}
	}

	// Tomar decisiones
	decisions := analyzeContainers(containers, sysData)

	// Ordenar y mostrar TOP 3
	ramSorted := docker.SortContainersByRAM(containers)
	cpuSorted := docker.SortContainersByCPU(containers)

	fmt.Printf("🏆 TOP 3 RAM: ")
	for _, c := range ramSorted[:min(3, len(ramSorted))] {
		fmt.Printf("%s(%.1fMB) ", c.Names, float64(c.RSSKB)/1024)
	}
	fmt.Println()

	fmt.Printf("⚡ TOP 3 CPU: ")
	for _, c := range cpuSorted[:min(3, len(cpuSorted))] {
		fmt.Printf("%s(%.1f%%) ", c.Names, c.CPUPercent)
	}
	fmt.Println()

	// Ejecutar decisiones
	if len(decisions) > 0 {
		fmt.Printf("🎯 Tomando %d decisiones...\n", len(decisions))
		if err := docker.ExecuteDecisions(decisions); err != nil {
			return fmt.Errorf("error ejecutando decisiones: %w", err)
		}
	}

	// Guardar en base de datos
	if err := dbConn.SaveMetrics(sysData.TotalRAMMB, sysData.FreeRAMMB, sysData.UsedRAMMB,
		containerMetrics, convertDecisions(decisions)); err != nil {
		return fmt.Errorf("error guardando métricas: %w", err)
	}

	fmt.Printf("✅ Ciclo completado. Decisiones: %d\n", len(decisions))
	return nil
}

func analyzeContainers(containers []docker.ContainerInfo, sysData kernel.KernelData) []docker.Decision {
	var decisions []docker.Decision
	lowCount := docker.CountLowConsumption(containers)   // <10% CPU y Mem
	highCount := docker.CountHighConsumption(containers) // >50% CPU o Mem
	totalContainers := len(containers)

	fmt.Printf("📊 %d contenedores bajos, %d altos (objetivo: 3 bajos, 2 altos), total: %d\n",
		lowCount, highCount, totalContainers)

	usedRAMPercent := float64(sysData.UsedRAMMB) / float64(sysData.TotalRAMMB) * 100

	// Eliminar excedentes para mantener 5 contenedores totales (3 bajos, 2 altos)
	if totalContainers > 5 {
		ramSorted := docker.SortContainersByRAM(containers)
		for i, container := range ramSorted {
			if i >= 5 && !strings.Contains(container.Names, "grafana") {
				decisions = append(decisions, docker.Decision{
					Action:      "stop_and_remove",
					ContainerID: container.ID,
					Reason:      fmt.Sprintf("Excede límite total (RAM usada: %.1f%%)", usedRAMPercent),
				})
			}
		}
	}

	// Ajustar a 3 bajos
	if lowCount > 3 {
		ramSorted := docker.SortContainersByRAM(containers)
		for _, container := range ramSorted {
			if lowCount > 3 && container.MemPercent < 10 && container.CPUPercent < 10 &&
				!strings.Contains(container.Names, "grafana") {
				decisions = append(decisions, docker.Decision{
					Action:      "stop_and_remove",
					ContainerID: container.ID,
					Reason:      "Excede límite de 3 bajos",
				})
				lowCount--
			}
		}
	}

	// Ajustar a 2 altos
	if highCount > 2 {
		cpuSorted := docker.SortContainersByCPU(containers)
		for _, container := range cpuSorted {
			if highCount > 2 && (container.CPUPercent > 50 || container.MemPercent > 50) &&
				!strings.Contains(container.Names, "grafana") {
				decisions = append(decisions, docker.Decision{
					Action:      "stop_and_remove",
					ContainerID: container.ID,
					Reason:      "Excede límite de 2 altos",
				})
				highCount--
			}
		}
	}

	// Crear si faltan bajo carga razonable
	if highCount < 2 && usedRAMPercent < 60 {
		decisions = append(decisions, docker.Decision{
			Action: "create_high",
			Reason: fmt.Sprintf("Faltan %d altos (RAM usada: %.1f%%)", 2-highCount, usedRAMPercent),
		})
	}
	if lowCount < 3 && usedRAMPercent < 60 {
		decisions = append(decisions, docker.Decision{
			Action: "create_low",
			Reason: fmt.Sprintf("Faltan %d bajos (RAM usada: %.1f%%)", 3-lowCount, usedRAMPercent),
		})
	}

	return decisions
}

func convertDecisions(dockerDecisions []docker.Decision) []db.Decision {
	var dbDecisions []db.Decision
	for _, d := range dockerDecisions {
		dbDecisions = append(dbDecisions, db.Decision{
			Action:      d.Action,
			ContainerID: d.ContainerID,
			Reason:      d.Reason,
		})
	}
	return dbDecisions
}

func cleanup(dbConn *db.DBConn) {
	fmt.Println("🧹 Limpiando recursos...")

	// 1. Eliminar cronjob
	if err := docker.RemoveCronjob(); err != nil {
		log.Printf("⚠️  Error removiendo cronjob: %v", err)
	}

	// 2. Descargar módulos del kernel
	if err := kernel.UnloadModules(); err != nil {
		log.Printf("⚠️  Error descargando módulos: %v", err)
	}

	// 3. Parar Grafana
	if err := grafana.StopGrafana(); err != nil {
		log.Printf("⚠️  Error parando Grafana: %v", err)
	}

	fmt.Println("✅ Limpieza completada")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func CleanupExcessContainers() {
	containers, _ := docker.ListContainers(CONTINFO_PATH, SYSINFO_PATH)
	if len(containers) > 5 {
		for _, c := range containers[5:] {
			if !strings.Contains(c.Names, "grafana") {
				docker.ExecuteDecisions([]docker.Decision{{Action: "stop_and_remove", ContainerID: c.ID}})
			}
		}
	}
}
