package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DBConn struct {
	*sql.DB
}

type Metric struct {
	ID             int       `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	TotalRAMMB     int64     `json:"total_ram_mb"`
	FreeRAMMB      int64     `json:"free_ram_mb"`
	UsedRAMMB      int64     `json:"used_ram_mb"`
	ContainerID    string    `json:"container_id"`
	ContainerPID   int       `json:"container_pid"`
	Image          string    `json:"image"`
	Names          string    `json:"names"`
	RSSKB          int64     `json:"rss_kb"`
	VSZKB          int64     `json:"vsz_kb"`
	CPUPercent     int       `json:"cpu_percent"`
	MemPercent     int       `json:"mem_percent"`
	Action         string    `json:"action"`
	DecisionReason string    `json:"decision_reason"`
}

type Decision struct {
	ContainerID string `json:"container_id"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
}

func InitDB(dbPath string) (*DBConn, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Crear tablas si no existen
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS system_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		total_ram_mb INTEGER,
		free_ram_mb INTEGER,
		used_ram_mb INTEGER
	);

	CREATE TABLE IF NOT EXISTS container_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		container_id TEXT,
		container_pid INTEGER,
		image TEXT,
		names TEXT,
		rss_kb INTEGER,
		vsz_kb INTEGER,
		cpu_percent INTEGER,
		mem_percent INTEGER,
		action TEXT,
		decision_reason TEXT
	);

	CREATE TABLE IF NOT EXISTS decisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		container_id TEXT,
		action TEXT,
		decision_reason TEXT
	);
	`

	_, err = db.Exec(createTablesSQL)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Base de datos SQLite inicializada en %s", dbPath)
	return &DBConn{db}, nil
}

func (conn *DBConn) SaveMetrics(totalRAM, freeRAM, usedRAM int64,
	containerMetrics []Metric, decisions []Decision) error {

	// 1. Guardar métricas del sistema
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO system_metrics (total_ram_mb, free_ram_mb, used_ram_mb) VALUES (?, ?, ?)",
		totalRAM, freeRAM, usedRAM,
	)
	if err != nil {
		return err
	}

	// 2. Guardar métricas de contenedores
	stmt, err := tx.Prepare(
		`INSERT INTO container_metrics (timestamp, container_id, container_pid, image, names, rss_kb, vsz_kb, 
		 cpu_percent, mem_percent, action, decision_reason) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, metric := range containerMetrics {
		_, err := stmt.Exec(
			metric.Timestamp, metric.ContainerID, metric.ContainerPID, metric.Image, metric.Names,
			metric.RSSKB, metric.VSZKB, metric.CPUPercent, metric.MemPercent,
			metric.Action, metric.DecisionReason,
		)
		if err != nil {
			log.Printf("⚠️  Error guardando métrica de %s: %v", metric.ContainerID, err)
		}
	}

	// 3. Guardar decisiones
	for _, decision := range decisions {
		_, err := tx.Exec(
			`INSERT INTO decisions (timestamp, container_id, action, decision_reason) 
			 VALUES (?, ?, ?, ?)`,
			time.Now(), decision.ContainerID, decision.Action, decision.Reason,
		)
		if err != nil {
			log.Printf("⚠️  Error guardando decisión %s: %v", decision.ContainerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("💾 Guardadas %d métricas de contenedores y %d decisiones",
		len(containerMetrics), len(decisions))
	return nil
}

func (conn *DBConn) GetContainerRemovals() ([]RemovalCount, error) {
	rows, err := conn.Query(`
		SELECT DATE(timestamp) as date, COUNT(*) as count 
		FROM container_metrics 
		WHERE action = 'removed' 
		GROUP BY DATE(timestamp) 
		ORDER BY date
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var removals []RemovalCount
	for rows.Next() {
		var r RemovalCount
		err := rows.Scan(&r.Date, &r.Count)
		if err != nil {
			return nil, err
		}
		removals = append(removals, r)
	}
	return removals, nil
}

func (conn *DBConn) GetRAMUsage() ([]RAMUsage, error) {
	rows, err := conn.Query(`
		SELECT timestamp, used_ram_mb 
		FROM system_metrics 
		ORDER BY timestamp
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usage []RAMUsage
	for rows.Next() {
		var u RAMUsage
		err := rows.Scan(&u.Timestamp, &u.UsedRAMMB)
		if err != nil {
			return nil, err
		}
		usage = append(usage, u)
	}
	return usage, nil
}

func (conn *DBConn) GetTopContainers(limit int) ([]TopContainer, error) {
	query := fmt.Sprintf(`
		SELECT container_id, image, rss_kb, mem_percent 
		FROM container_metrics 
		WHERE rss_kb > 0 
		ORDER BY rss_kb DESC 
		LIMIT %d
	`, limit)

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topContainers []TopContainer
	for rows.Next() {
		var tc TopContainer
		err := rows.Scan(&tc.ContainerID, &tc.Image, &tc.RSSKB, &tc.MemPercent)
		if err != nil {
			return nil, err
		}
		topContainers = append(topContainers, tc)
	}
	return topContainers, nil
}

// Estructuras auxiliares
type RemovalCount struct {
	Date  string
	Count int
}

type RAMUsage struct {
	Timestamp time.Time
	UsedRAMMB int64
}

type TopContainer struct {
	ContainerID string
	Image       string
	RSSKB       int64
	MemPercent  int
}
