package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"caffe-podjok-sendja/internal/database"
	"caffe-podjok-sendja/internal/httpapi"
	"caffe-podjok-sendja/internal/persistence"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Inisialisasi penyimpanan lokal (backup / fallback) terlebih dahulu
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	persistence.Init(dataDir)

	// 2. Inisialisasi basis data MySQL / TiDB secara asinkron di latar belakang
	// Ini memastikan HTTP server langsung bind ke 0.0.0.0:$PORT dalam hitungan milidetik,
	// mencegah terjadinya "Port scan timeout" pada platform cloud hosting seperti Render.com.
	go func() {
		if _, err := database.Init(); err != nil {
			log.Printf("[main] MySQL/TiDB tidak aktif atau belum siap (%v). Server tetap melayani permintaan dengan penyimpanan lokal.", err)
		} else {
			log.Println("[main] MySQL/TiDB berhasil terhubung dan tersinkronisasi.")
		}
	}()

	// 3. Wajib bind secara eksplisit ke 0.0.0.0:<PORT> agar terdeteksi oleh port scanner & load balancer Render
	bindAddr := "0.0.0.0:" + port

	server := &http.Server{
		Addr:              bindAddr,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("server aktif dan mendengarkan pada http://%s (bind 0.0.0.0:%s)", bindAddr, port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("menutup server dan menyimpan data...")
	persistence.SaveAll()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
