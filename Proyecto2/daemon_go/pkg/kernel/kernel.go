package kernel

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type KernelData struct {
	TotalRAMMB int64      `json:"total_ram_mb"`
	FreeRAMMB  int64      `json:"free_ram_mb"`
	UsedRAMMB  int64      `json:"used_ram_mb"`
	Containers []ProcInfo `json:"containers"` // Para continfo
	Processes  []ProcInfo `json:"processes"`  // Para sysinfo
}

type ProcInfo struct {
	PID        int    `json:"pid"`
	Name       string `json:"name"`
	Cmdline    string `json:"cmdline"`
	VSZKB      int64  `json:"vsz_kb"`
	RSSKB      int64  `json:"rss_kb"`
	MemPercent int    `json:"mem_percent"`
	CPUPercent int    `json:"cpu_percent"`
	State      string `json:"state,omitempty"`
}

func ReadProcData(contPath, sysPath string) (KernelData, KernelData, error) {
	var contData, sysData KernelData
	var err error

	// Leer sysinfo
	if content, err := os.ReadFile(sysPath); err == nil {
		//log.Printf("📋 Contenido de %s: %s", sysPath, string(content))
		if err = json.Unmarshal(content, &sysData); err != nil {
			log.Printf("⚠️  Error parseando %s: %v", sysPath, err)
		}
	} else {
		log.Printf("⚠️  No se pudo leer %s: %v", sysPath, err)
	}

	// Leer continfo
	if content, err := os.ReadFile(contPath); err == nil {
		//log.Printf("📋 Contenido de %s: %s", contPath, string(content))
		if err = json.Unmarshal(content, &contData); err != nil {
			log.Printf("⚠️  Error parseando %s: %v", contPath, err)
		}
	} else {
		log.Printf("⚠️  No se pudo leer %s: %v", contPath, err)
	}

	return contData, sysData, err
}

func LoadModules() error {
	cmd := exec.Command("sudo", "insmod", "/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/kernel_continfo/continfo.ko")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error cargando continfo: %w. Output: %s", err, string(output))
	}
	cmd = exec.Command("sudo", "insmod", "/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/kernel_sysinfo/sysinfo.ko")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error cargando sysinfo: %w. Output: %s", err, string(output))
	}
	log.Println("✅ Módulos del kernel cargados")
	return nil
}

func UnloadModules() error {
	if output, err := exec.Command("lsmod").Output(); err == nil {
		if !strings.Contains(string(output), "continfo") {
			log.Println("ℹ️  Módulo continfo no cargado")
		} else {
			cmd := exec.Command("sudo", "rmmod", "continfo")
			if _, err := cmd.CombinedOutput(); err != nil {
				log.Printf("⚠️  Error descargando módulo continfo: %v", err)
			}
		}
		if !strings.Contains(string(output), "sysinfo") {
			log.Println("ℹ️  Módulo sysinfo no cargado")
		} else {
			cmd := exec.Command("sudo", "rmmod", "sysinfo")
			if _, err := cmd.CombinedOutput(); err != nil {
				log.Printf("⚠️  Error descargando módulo sysinfo: %v", err)
			}
		}
	}
	log.Println("✅ Módulos del kernel descargados")
	return nil
}
