package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/makseli/medi-pill-check/internal/config"
	"github.com/makseli/medi-pill-check/internal/database"
	"github.com/makseli/medi-pill-check/internal/handlers"
	"github.com/makseli/medi-pill-check/internal/routes"
)

func main() {
	cfg := config.Load()

	if cfg.DBType == "sqlite" {
		if err := os.MkdirAll("data/sqlite", 0755); err != nil {
			log.Fatalf("Klasör oluşturulamadı: %v", err)
		}
	}

	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("Veritabanı başlatılamadı: %v", err)
	}

	r := gin.Default()

	h := handlers.New(db, cfg)
	routes.Setup(r, h)

	log.Printf("Sunucu %s portunda çalışıyor...", cfg.Port)
	r.Run(":" + cfg.Port)
}
