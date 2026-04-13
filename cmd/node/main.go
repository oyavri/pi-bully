package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/oyavri/pi-bully/cluster"
	"github.com/oyavri/pi-bully/config"
	"github.com/oyavri/pi-bully/server"
	"github.com/oyavri/pi-bully/storage"
	"github.com/oyavri/pi-bully/task"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/oyavri/pi-bully/gen/bully"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := cfg.Log.BuildLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("starting pi_bully node",
		zap.Uint64("nodeID", cfg.Node.ID),
		zap.String("address", cfg.Node.Address),
	)

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	// PostgreSQL
	pool, err := task.NewPool(ctx, cfg.Database.DSN)
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	taskStore := task.NewPostgresStore(pool, logger)

	// Storage
	storageClient, err := storage.New(cfg.Storage, logger)
	if err != nil {
		logger.Fatal("failed to create storage client", zap.Error(err))
	}
	logger.Info("storage client ready")

	// gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Server.Port))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	pb.RegisterElectionServiceServer(grpcServer, server.NewElectionServer(logger))
	pb.RegisterWorkerServiceServer(grpcServer, server.NewWorkerServer(logger))
	pb.RegisterSchedulerServiceServer(grpcServer, server.NewSchedulerServer(logger))
	logger.Info("gRPC servers registered")

	// Start gRPC server in background
	go func() {
		logger.Info("gRPC server listening", zap.String("port", cfg.Server.Port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	// Cluster
	cl, err := cluster.New(cfg.Memberlist, cfg.Node, cfg.Server, logger)
	if err != nil {
		logger.Fatal("failed to create cluster", zap.Error(err))
	}

	if err := cl.Join(cfg.Memberlist.Seeds); err != nil {
		logger.Fatal("failed to join cluster")
	}

	<-ctx.Done()
	logger.Info("shutting down")

	grpcServer.GracefulStop()

	if err := cl.Leave(); err != nil {
		logger.Error("failed to leave cluster cleanly")
	}

	// to suppress for now
	_ = taskStore
	_ = storageClient
}
