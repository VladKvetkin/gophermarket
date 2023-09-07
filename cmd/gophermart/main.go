package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/VladKvetkin/gophermart/internal/accrualer"
	"github.com/VladKvetkin/gophermart/internal/config"
	"github.com/VladKvetkin/gophermart/internal/server"
	"github.com/VladKvetkin/gophermart/internal/storage"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	os.Exit(start())
}

func start() int {
	config, err := config.NewConfig()
	if err != nil {
		zap.L().Info("error create config", zap.Error(err))
		return 1
	}

	defer zap.L().Sync()

	db, err := sqlx.Connect("postgres", config.DatabaseURI)
	if err != nil {
		zap.L().Info("error failed to connect to db: %w", zap.Error(err))
		return 1
	}

	defer db.Close()

	postgresStorage, err := storage.NewPostgresStorage(db)
	if err != nil {
		zap.L().Info("error failed to create postgres storage: %w", zap.Error(err))
		return 1
	}

	var (
		accrualer = accrualer.NewAccrualer(
			config.AccrualSystemAddress,
			postgresStorage,
		)
	)

	server := server.NewServer(config, postgresStorage)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		if err := server.Start(); err != nil {
			zap.L().Info("error starting server", zap.Error(err))
			return err
		}

		return nil
	})

	eg.Go(func() error {
		if err := accrualer.Start(ctx); err != nil {
			zap.L().Info("error starting accrualer", zap.Error(err))
			return err
		}

		return nil
	})

	<-ctx.Done()

	eg.Go(func() error {
		if err := server.Stop(); err != nil {
			zap.L().Info("error stopping server", zap.Error(err))
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return 1
	}

	return 0
}
