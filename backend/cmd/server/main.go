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

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/httpapi"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/repository"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/service"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// DB の起動待ち（compose の depends_on.healthy で基本的に不要だがローカル単独起動用）
	store, closeFn, err := waitForStore(ctx, dbURL, 30*time.Second)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer closeFn()

	svc := service.New(store)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewServer(svc).Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("cmaas listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func waitForStore(ctx context.Context, url string, d time.Duration) (repository.Store, func(), error) {
	deadline := time.Now().Add(d)
	var lastErr error
	for time.Now().Before(deadline) {
		store, closeFn, err := repository.NewStoreFromURL(ctx, url)
		if err == nil {
			return store, closeFn, nil
		}
		lastErr = err
		log.Printf("waiting for db... (%v)", err)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return nil, nil, lastErr
}
