package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/log"
	"ride-hail/internal/common/rabbitmq"
	commonws "ride-hail/internal/common/ws"

	api "ride-hail/internal/ride/adapters/httpadapter"
	"ride-hail/internal/ride/adapters/repository"
	queue "ride-hail/internal/ride/adapters/rmq"
	passengerws "ride-hail/internal/ride/adapters/ws"
	"ride-hail/internal/ride/app"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.New("ride-service")
	log.Info(ctx, logger, "init_start", "Ride Service initializing...")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Error(ctx, logger, "config_load_fail", "Failed to load config", err)
		os.Exit(1)
	}

	dbPool, err := db.ConnectPostgres(ctx, cfg.DB)
	if err != nil {
		log.Error(ctx, logger, "db_connect_fail", "Failed to connect database", err)
		os.Exit(1)
	}

	rmq := rabbitmq.NewMQ(cfg.RMQ, logger)
	if err := rmq.Connect(ctx); err != nil {
		log.Error(ctx, logger, "rmq_connect_fail", "Failed to connect RabbitMQ", err)
		os.Exit(1)
	}
	if err := rmq.DeclareTopology(); err != nil {
		log.Error(ctx, logger, "rmq_declare_fail", "Failed to declare RMQ topology", err)
		os.Exit(1)
	}

	hub := commonws.NewHub(logger)
	passengerWSHandler := passengerws.NewPassengerWSHandler(logger, hub)
	wsTalker := passengerws.NewTalker(hub)

	rideRepo := repository.NewRideRepository(dbPool)
	publisher := queue.NewRidePublisher(rmq, logger)
	coreService := app.NewAppService(rideRepo, publisher, wsTalker)

	apiHandler := api.NewHandler(coreService, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/passengers/", passengerWSHandler.HandlePassengerWS)
	mux.Handle("/", apiHandler.Router())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.WS.Port),
		Handler: mux,
	}

	go func() {
		log.Info(ctx, logger, "http_start", "Ride service HTTP server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, logger, "http_fail", "HTTP server failed", err)
			cancel()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-stop:
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	rmq.Close()
	dbPool.Close()
	log.InfoX(logger, "shutdown_complete", "Ride Service stopped")
}
