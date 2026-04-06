package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"omega/backend/internal/api"
	"omega/backend/internal/bank"
	"omega/backend/internal/contracts"
	"omega/backend/internal/registry"
	"omega/backend/internal/store"
)

func main() {
	port := flag.Int("port", 8090, "HTTP port")
	dbPath := flag.String("db", "omega.db", "SQLite database path")
	flag.Parse()

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	reg := registry.New(db)
	bnk := bank.New(db)
	cts := contracts.New(db, bnk)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: api.NewServer(db, reg, bnk, cts),
	}

	go func() {
		log.Printf("omega gov server listening on http://localhost:%d", *port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	srv.Shutdown(context.Background())
}
