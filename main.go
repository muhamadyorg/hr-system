package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"hr-system/goserver"
)

func main() {
	log.Println("[HR System] Ishga tushmoqda...")

	if err := goserver.InitDatabase(); err != nil {
		log.Fatalf("Database xatoligi: %v", err)
	}
	log.Println("[HR System] Database ulandi")

	if err := goserver.SeedDatabase(); err != nil {
		log.Printf("[HR System] Seed xatoligi: %v", err)
	}

	goserver.InitTelegramBotFromDb()

	autoSync, _ := goserver.GetSetting("hikvision_auto_sync")
	if autoSync != nil && *autoSync == "true" {
		goserver.StartAutoSync(10)
	}

	mux := http.NewServeMux()
	goserver.RegisterRoutes(mux)

	distDir := filepath.Join(".", "client", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir = filepath.Join(".", "dist", "public")
	}

	fs := http.FileServer(http.Dir(distDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(distDir, r.URL.Path)
		if _, err := os.Stat(path); err == nil && r.URL.Path != "/" {
			fs.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("[HR System] Server %s portda ishga tushdi", port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server xatoligi: %v", err)
	}
}
