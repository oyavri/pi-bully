package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/oyavri/pi-bully/config"
	"github.com/oyavri/pi-bully/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/oyavri/pi-bully/gen/bully"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		zap.L().Fatal("failed to load config", zap.Error(err))
		os.Exit(1)
	}

	logger, err := cfg.Log.BuildLogger()
	if err != nil {
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

	// gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Server.Port))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	pb.RegisterElectionServiceServer(grpcServer, server.NewElectionServer(logger))
	pb.RegisterWorkerServiceServer(grpcServer, server.NewWorkerServer(logger))
	pb.RegisterSchedulerServiceServer(grpcServer, server.NewSchedulerServer(logger))

	// Start gRPC server in background
	go func() {
		logger.Info("gRPC server listening", zap.String("port", cfg.Server.Port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
}
