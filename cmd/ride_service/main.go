package main

import (
	"context"
	"os"
	"os/signal"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/db"
	"ride-hail/internal/common/log"
	"ride-hail/internal/common/rabbitmq"
	"ride-hail/internal/ride/adapters/httpadapter"
	"ride-hail/internal/ride/adapters/repository"
	rabbit "ride-hail/internal/ride/adapters/rmq"
	"ride-hail/internal/ride/app"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.New("ride-service")
	log.Info(ctx, logger, "init_start", "Ride Service initializing...")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Error(ctx, logger, "config_load_fail", "Failed to load config file", err)
		os.Exit(1)
	}
	log.Info(ctx, logger, "config_loaded", "Configuration loaded successfully")

	dbPool, err := db.ConnectPostgres(ctx, cfg.DB)
	if err != nil {
		log.Error(ctx, logger, "connect_db_fail", "Failed to connect to database", err)
		os.Exit(1)
	}
	log.Info(ctx, logger, "db_connected", "Successfully connected to database")

	rmq := rabbitmq.NewMQ(cfg.RMQ, logger)
	if err := rmq.Connect(ctx); err != nil {
		log.Error(ctx, logger, "rmq_connect_fail", "Failed to connect rabbit MQ", err)
		os.Exit(1)
	}

	if err := rmq.DeclareTopology(); err != nil {
		log.Error(ctx, logger, "rmq_declare_topology_fail", "Failed to declare RMQ topology", err)
		os.Exit(1)
	}
	log.Info(ctx, logger, "rmq_ready", "RabbitMQ connected and topology declared")

	repo     := repository.NewRideRepository(dbPool)
	pub      := rabbit.NewPublisher(rmq, logger)
	_ = rabbit.NewConsumer(rmq, logger)

	// Домашний сервис приложений
	rideSvc := app.NewRideService(repo, pub)

	// // --- Старт консьюмеров (фоновые горутины) ---
	// // Ответы водителей
	// if err := consumer.StartDriverResponses(ctx, func(m queue.DriverResponse) error {
	// 	// Пример: если водитель принял — обновим статус
	// 	if m.Accepted {
	// 		return rideSrv.UpdateRideStatus(ctx, m.RideID, "ACCEPTED")
	// 	}
	// 	// Иначе можно зафиксировать отказ/продолжить матчинг
	// 	return nil
	// }); err != nil {
	// 	log.Error(ctx, logger, "consumer_start_fail", "driver responses consumer failed to start", err)
	// 	os.Exit(1)
	// }

	// // Апдейты локаций (fanout)
	// if err := consumer.StartLocationUpdates(ctx, func(m queue.LocationUpdate) error {
	// 	// Пример: можно обновлять ETA/координаты в БД и пушить по WS
	// 	// rideSrv.UpdateDriverLocation(ctx, m)  // если есть такой метод
	// 	return nil
	// }); err != nil {
	// 	log.Error(ctx, logger, "consumer_start_fail", "location updates consumer failed to start", err)
	// 	os.Exit(1)
	// }
	// log.Info(ctx, logger, "consumers_started", "All RMQ consumers started")

	httpSrv := httpadapter.NewServer(rideSvc, logger)

	go func() {
		if err := httpSrv.Start(ctx, ":3000"); err != nil {
			log.Error(ctx, logger, "http_start_fail", "HTTP server stopped with error", err)
		}
	}()
	log.Info(ctx, logger, "http_listening", "HTTP server is listening at :3000")

	// --- Graceful shutdown ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info(ctx, logger, "shutdown", "Ride Service shutting down...")
	cancel()                       // остановим фоновые горутины/HTTP
	time.Sleep(1 * time.Second)    // короткая пауза на graceful
	log.Info(ctx, logger, "shutdown_complete", "Service stopped successfully")
}
