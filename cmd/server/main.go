package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makseli/medi-pill-check/internal/config"
	"github.com/makseli/medi-pill-check/internal/database"
	"github.com/makseli/medi-pill-check/internal/handlers"
	"github.com/makseli/medi-pill-check/internal/middleware"
	"github.com/makseli/medi-pill-check/internal/routes"
)

func main() {

	cfg := config.Load()

	if cfg.DBType == "sqlite" {
		if err := os.MkdirAll("data/sqlite", 0755); err != nil {
			log.Printf("[WARN] Klasör oluşturulamadı: %v", err)
		}
	}

	db, err := database.Init(cfg)
	if err != nil {
		log.Printf("[WARN] Veritabanı başlatılamadı: %v", err)
		// db nil olabilir, health endpointi bunu gösterecek
	}

	if err := database.InitRedis(cfg); err != nil {
		log.Printf("[WARN] Redis'e bağlanılamadı: %v", err)
		// RedisClient nil olabilir, health endpointi bunu gösterecek
	}

	r := gin.Default()
	r.Use(middleware.SecureCORS(cfg))
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	h := handlers.New(db, cfg)
	mh := handlers.NewMedicineHandler(db)
	routes.Setup(r, h, mh)

	log.Printf("Server is running on port %s...", cfg.Port)

	// Graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-quit
	log.Println("Sunucu kapatılıyor... (graceful shutdown)")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Zorunlu kapatma: %v", err)
	}
	log.Println("Sunucu başarıyla kapatıldı.")
}
