package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonyloussararian/Go-CryptoPrice/database"
	"github.com/antonyloussararian/Go-CryptoPrice/handlers"
	"github.com/antonyloussararian/Go-CryptoPrice/kraken"
	"github.com/gin-gonic/gin"
)

func main() {
	db, err := database.NewDB("crypto.db")
	if err != nil {
		log.Fatalf("Erreur lors de l'initialisation de la base de données: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(); err != nil {
		log.Fatalf("Erreur lors de l'initialisation du schéma: %v", err)
	}

	krakenClient := kraken.NewClient()

	h := handlers.NewHandler(db, krakenClient)

	log.Println("Premier enregistrement des données...")
	if err := h.SaveDataToDB(); err != nil {
		log.Printf("Erreur lors du premier enregistrement: %v", err)
	} else {
		log.Println("Premier enregistrement effectué avec succès")
	}

	h.StartAutoSave()

	r := gin.Default()

	r.GET("/api/status", h.GetServerStatus)
	r.GET("/api/pairs", h.GetTradingPairs)
	r.GET("/api/pairs/:pair", h.GetPairInfo)
	r.GET("/api/historical", h.DownloadHistoricalData)
	r.GET("/api/db", h.GetDBData)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erreur serveur HTTP: %v", err)
		}
	}()

	<-stopChan
	log.Println("Arrêt du serveur...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Erreur lors de l'arrêt du serveur: %v", err)
	}
}
